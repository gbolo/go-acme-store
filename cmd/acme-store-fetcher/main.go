package main

import (
	"flag"
	"fmt"
	"go-acme-store/pkg/config"
	"go-acme-store/pkg/keystore"
	"go-acme-store/pkg/log"
	"go-acme-store/pkg/meta"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func main() {
	appName := "acme-store-fetcher"

	// Parse command-line flags
	configFile := flag.String("config", "", "path to configuration file (default: ./config.yml or CONFIG_FILE env var)")
	outputDir := flag.String("output-dir", "./certs", "directory to save certificate files")
	traefikConfig := flag.String("traefik-config", "", "path to write Traefik configuration file (optional)")
	showVersion := flag.Bool("version", false, "display version information")
	flag.Parse()

	// Handle version flag first (before config loading)
	if *showVersion {
		fmt.Println(meta.GetAppMetadata().ToString())
		return
	}

	// Determine config file path
	cfg := *configFile
	if cfg == "" {
		cfg = os.Getenv("CONFIG_FILE")
	}
	if cfg == "" {
		cfg = "./config.yml"
	}

	config.MustInitViperAndLogger(appName, cfg)

	// Determine output directory with precedence: flag > config > default
	finalOutputDir := *outputDir
	if finalOutputDir == "./certs" && viper.IsSet("fetcher.output_dir") {
		finalOutputDir = viper.GetString("fetcher.output_dir")
	}

	// Determine traefik config with precedence: flag > config > none
	finalTraefikConfig := *traefikConfig
	if finalTraefikConfig == "" && viper.IsSet("fetcher.traefik_config") {
		finalTraefikConfig = viper.GetString("fetcher.traefik_config")
	}

	// Initialize keystore (use account email from config for vault path)
	accountEmail := viper.GetString("acme.account_email")
	ks, err := keystore.NewVaultKeystoreFromViper(accountEmail)
	if err != nil {
		log.Fatalf("failed to initialize vault keystore: %v", err)
	}

	log.Infof("initialized vault keystore at %s", viper.GetString("vault.address"))

	// Ensure output directory exists
	if err := os.MkdirAll(finalOutputDir, 0755); err != nil {
		log.Fatalf("failed to create output directory %s: %v", finalOutputDir, err)
	}

	log.Infof("output directory: %s", finalOutputDir)

	// Fetch managed domains
	managedDomains, err := ks.GetManagedDomains()
	if err != nil {
		log.Fatalf("failed to get managed domains: %v", err)
	}

	log.Infof("found %d managed domain(s)", len(managedDomains))

	if len(managedDomains) == 0 {
		log.Warnf("no managed domains found")
		return
	}

	// Process each domain
	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, domain := range managedDomains {
		log.Infof("processing domain: %s", domain)

		// Fetch certificate and key
		certAndKey, err := ks.GetCertAndKey(domain)
		if err != nil {
			log.Errorf("failed to get certificate for %s: %v", domain, err)
			errorCount++
			continue
		}

		// Skip if not managed (should not happen, but safety check)
		if !certAndKey.Managed {
			log.Warnf("skipping %s (marked as unmanaged)", domain)
			skipCount++
			continue
		}

		// Skip if status is not "issued"
		if certAndKey.Status != "issued" && certAndKey.Status != "" {
			log.Warnf("skipping %s (status: %s)", domain, certAndKey.Status)
			skipCount++
			continue
		}

		// Skip if certificate data is empty
		if certAndKey.LeafCertPEM == "" || certAndKey.PrivateKeyPEM == "" {
			log.Warnf("skipping %s (certificate or key data is empty)", domain)
			skipCount++
			continue
		}

		// Generate safe filename (replace * with _wild_)
		safeDomain := strings.ReplaceAll(domain, "*", "_wild_")

		certFilename := filepath.Join(finalOutputDir, fmt.Sprintf("%s_cert-chain.pem", safeDomain))
		keyFilename := filepath.Join(finalOutputDir, fmt.Sprintf("%s_key.pem", safeDomain))

		// Build full chain (leaf + intermediates)
		fullChain := certAndKey.LeafCertPEM
		if certAndKey.CertChainPEM != "" {
			fullChain = fullChain + "\n" + certAndKey.CertChainPEM
		}

		// Write certificate chain
		if err := os.WriteFile(certFilename, []byte(fullChain), 0644); err != nil {
			log.Errorf("failed to write certificate file for %s: %v", domain, err)
			errorCount++
			continue
		}

		// Write private key
		if err := os.WriteFile(keyFilename, []byte(certAndKey.PrivateKeyPEM), 0600); err != nil {
			log.Errorf("failed to write key file for %s: %v", domain, err)
			errorCount++
			// Try to clean up cert file if key write failed
			os.Remove(certFilename)
			continue
		}

		log.Infof("saved certificate for %s:", domain)
		log.Infof("  - cert chain: %s", certFilename)
		log.Infof("  - private key: %s", keyFilename)
		successCount++
	}

	// Generate Traefik configuration if requested
	if finalTraefikConfig != "" && successCount > 0 {
		log.Infof("generating Traefik configuration file: %s", finalTraefikConfig)

		if err := generateTraefikConfig(finalTraefikConfig, finalOutputDir, managedDomains, ks); err != nil {
			log.Errorf("failed to generate Traefik config: %v", err)
			errorCount++
		} else {
			log.Infof("Traefik configuration file created successfully")
		}
	}

	// Summary
	log.Infof("========================================")
	log.Infof("Summary:")
	log.Infof("  Total domains: %d", len(managedDomains))
	log.Infof("  Successfully saved: %d", successCount)
	log.Infof("  Skipped: %d", skipCount)
	log.Infof("  Errors: %d", errorCount)
	if finalTraefikConfig != "" && successCount > 0 {
		log.Infof("  Traefik config: %s", finalTraefikConfig)
	}
	log.Infof("========================================")

	if errorCount > 0 {
		os.Exit(1)
	}
}

type traefikCertificate struct {
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

type traefikTLSOptions struct {
	MinVersion string `yaml:"minVersion"`
}

type traefikTLS struct {
	Options      map[string]traefikTLSOptions `yaml:"options"`
	Certificates []traefikCertificate         `yaml:"certificates"`
}

type traefikConfig struct {
	TLS traefikTLS `yaml:"tls"`
}

func generateTraefikConfig(configPath string, certDir string, domains []string, ks keystore.Keystore) error {
	var certificates []traefikCertificate

	// Build certificate list
	for _, domain := range domains {
		// Check if certificate exists and is valid
		certAndKey, err := ks.GetCertAndKey(domain)
		if err != nil || !certAndKey.Managed || certAndKey.Status != "issued" && certAndKey.Status != "" {
			continue
		}

		if certAndKey.LeafCertPEM == "" || certAndKey.PrivateKeyPEM == "" {
			continue
		}

		// Generate safe filename (same as file writing logic)
		safeDomain := strings.ReplaceAll(domain, "*", "_wild_")
		certFile := filepath.Join(certDir, fmt.Sprintf("%s_cert-chain.pem", safeDomain))
		keyFile := filepath.Join(certDir, fmt.Sprintf("%s_key.pem", safeDomain))

		certificates = append(certificates, traefikCertificate{
			CertFile: certFile,
			KeyFile:  keyFile,
		})
	}

	if len(certificates) == 0 {
		return fmt.Errorf("no valid certificates to include in Traefik config")
	}

	// Build Traefik configuration
	config := traefikConfig{
		TLS: traefikTLS{
			Options: map[string]traefikTLSOptions{
				"default": {
					MinVersion: "VersionTLS12",
				},
				"mintls13": {
					MinVersion: "VersionTLS13",
				},
			},
			Certificates: certificates,
		},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Infof("wrote %d certificate(s) to Traefik config", len(certificates))
	return nil
}
