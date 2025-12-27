package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"go-acme-store/pkg/config"
	"go-acme-store/pkg/log"
	"go-acme-store/pkg/meta"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func main() {
	appName := "acme-store-fetcher"

	// Parse command-line flags
	configFile := flag.String("config", "", "path to configuration file (default: ./config.yml or CONFIG_FILE env var)")
	outputDir := flag.String("output-dir", "./certs", "directory to save certificate files")
	traefikConfig := flag.String("traefik-config", "", "path to write Traefik configuration file (optional)")
	daemon := flag.Bool("daemon", false, "run as daemon, continuously checking for certificate updates")
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

	// Determine daemon mode with precedence: flag > config > false
	daemonMode := *daemon
	if !daemonMode && viper.IsSet("fetcher.daemon") {
		daemonMode = viper.GetBool("fetcher.daemon")
	}

	// Get check interval for daemon mode (default: 5 minutes)
	checkInterval := viper.GetDuration("fetcher.check_interval")
	if checkInterval == 0 {
		checkInterval = 5 * time.Minute
	}

	// Enforce minimum check interval of 30 seconds
	minInterval := 30 * time.Second
	if checkInterval < minInterval {
		log.Warnf("check_interval %v is too short, enforcing minimum of %v", checkInterval, minInterval)
		checkInterval = minInterval
	}

	// Get API URL
	apiURL := viper.GetString("fetcher.api_url")
	if apiURL == "" {
		log.Fatalf("fetcher.api_url is required (e.g., http://127.0.0.1:15872/api)")
	}

	// Validate API URL by checking health endpoint
	if err := checkAPIHealth(apiURL); err != nil {
		log.Fatalf("failed to connect to acme-store API at %s: %v", apiURL, err)
	}

	log.Infof("connected to acme-store API at %s", apiURL)

	// Ensure output directory exists
	if err := os.MkdirAll(finalOutputDir, 0755); err != nil {
		log.Fatalf("failed to create output directory %s: %v", finalOutputDir, err)
	}

	log.Infof("output directory: %s", finalOutputDir)

	// Run in daemon mode if requested
	if daemonMode {
		log.Infof("running in daemon mode, checking every %v", checkInterval)
		runDaemon(apiURL, finalOutputDir, finalTraefikConfig, checkInterval)
		return
	}

	// Run single fetch
	exitCode := performFetch(apiURL, finalOutputDir, finalTraefikConfig)
	os.Exit(exitCode)
}

// runDaemon runs the fetcher in daemon mode
func runDaemon(apiURL, outputDir, traefikConfig string, checkInterval time.Duration) {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run initial fetch immediately
	log.Infof("performing initial certificate fetch")
	performFetch(apiURL, outputDir, traefikConfig)

	// Start ticker for periodic checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Infof("daemon started, will check for updates every %v", checkInterval)
	log.Infof("press Ctrl+C to stop gracefully")

	for {
		select {
		case <-ctx.Done():
			log.Infof("context cancelled, shutting down")
			return
		case sig := <-sigChan:
			log.Infof("received signal %v, shutting down gracefully", sig)
			return
		case <-ticker.C:
			log.Infof("checking for certificate updates")
			performFetch(apiURL, outputDir, traefikConfig)
		}
	}
}

// performFetch performs a single fetch operation and returns exit code
func performFetch(apiURL, outputDir, traefikConfig string) int {
	// Fetch managed domains from API
	managedDomains, err := getManagedDomains(apiURL)
	if err != nil {
		log.Errorf("failed to get managed domains: %v", err)
		return 1
	}

	log.Infof("found %d managed domain(s)", len(managedDomains))

	if len(managedDomains) == 0 {
		log.Warnf("no managed domains found")
		return 0
	}

	// Fetch all certificates once (optimization: avoid calling /api/certs multiple times)
	certInfoMap, err := getAllCerts(apiURL)
	if err != nil {
		log.Errorf("failed to get certificates: %v", err)
		return 1
	}

	// Process each domain
	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, domain := range managedDomains {
		log.Infof("processing domain: %s", domain)

		// Lookup certificate info from map
		certInfo, found := certInfoMap[domain]
		if !found {
			log.Errorf("certificate not found for %s", domain)
			errorCount++
			continue
		}

		// Skip if not managed (should not happen, but safety check)
		if !certInfo.Managed {
			log.Warnf("skipping %s (marked as unmanaged)", domain)
			skipCount++
			continue
		}

		// Skip if status is not "issued"
		if certInfo.Status != "issued" && certInfo.Status != "" {
			log.Warnf("skipping %s (status: %s)", domain, certInfo.Status)
			skipCount++
			continue
		}

		// Skip if certificate data is empty
		if certInfo.LeafCertPEM == "" {
			log.Warnf("skipping %s (certificate data is empty)", domain)
			skipCount++
			continue
		}

		// Generate safe filename (replace * with _wild_)
		safeDomain := strings.ReplaceAll(domain, "*", "_wild_")

		certFilename := filepath.Join(outputDir, fmt.Sprintf("%s_cert-chain.pem", safeDomain))
		keyFilename := filepath.Join(outputDir, fmt.Sprintf("%s_key.pem", safeDomain))

		// Use full chain from API (already contains leaf + intermediates)
		// CertChainPEM contains the complete chain, LeafCertPEM is just the first cert extracted
		fullChain := certInfo.CertChainPEM
		if fullChain == "" {
			// Fallback to just leaf cert if chain is empty (shouldn't happen normally)
			fullChain = certInfo.LeafCertPEM
		}

		// Try to load existing private key from disk
		var privateKey string
		existingKeyData, err := os.ReadFile(keyFilename)
		if err == nil {
			// We have an existing key, check if it matches the certificate
			existingKey := string(existingKeyData)
			if verifyKeyPairMatch(fullChain, existingKey) {
				// Existing key matches, use it
				privateKey = existingKey
				log.Debugf("using existing private key for domain=%s", domain)
			} else {
				// Existing key doesn't match, need to fetch from API
				log.Warnf("existing key doesn't match certificate for domain=%s, fetching from API", domain)
			}
		}

		// Fetch private key from API if we don't have a valid one
		if privateKey == "" {
			privateKey, err = getPrivateKey(apiURL, domain)
			if err != nil {
				log.Errorf("failed to get private key for %s: %v", domain, err)
				errorCount++
				continue
			}
		}

		// Check if we need to update the files
		needsUpdate, reason := needsCertificateUpdate(certFilename, keyFilename, fullChain, privateKey)

		if !needsUpdate {
			log.Debugf("skipping domain=%s reason=%s", domain, reason)
			skipCount++
			continue
		}

		// Write certificate chain
		if err := os.WriteFile(certFilename, []byte(fullChain), 0644); err != nil {
			log.Errorf("failed to write certificate file for %s: %v", domain, err)
			errorCount++
			continue
		}

		// Write private key
		if err := os.WriteFile(keyFilename, []byte(privateKey), 0600); err != nil {
			log.Errorf("failed to write key file for %s: %v", domain, err)
			errorCount++
			// Try to clean up cert file if key write failed
			os.Remove(certFilename)
			continue
		}

		log.Infof("saved certificate for domain=%s cert=%s key=%s reason=%s", domain, certFilename, keyFilename, reason)
		successCount++
	}

	// Generate Traefik configuration if requested
	if traefikConfig != "" && successCount > 0 {
		if err := generateTraefikConfig(traefikConfig, outputDir, managedDomains, certInfoMap); err != nil {
			log.Errorf("failed to generate Traefik config: %v", err)
			errorCount++
		} else {
			log.Infof("generated Traefik config=%s certificates=%d", traefikConfig, successCount)
		}
	}

	// Summary
	summaryMsg := fmt.Sprintf("fetch complete: total=%d saved=%d skipped=%d errors=%d",
		len(managedDomains), successCount, skipCount, errorCount)
	if traefikConfig != "" && successCount > 0 {
		summaryMsg += fmt.Sprintf(" traefik_config=%s", traefikConfig)
	}
	log.Infof(summaryMsg)

	if errorCount > 0 {
		return 1
	}
	return 0
}

// Certificate comparison functions

// needsCertificateUpdate checks if certificate files need to be updated
// Returns (needsUpdate bool, reason string)
func needsCertificateUpdate(certFile, keyFile, newCert, newKey string) (bool, string) {
	// Check if files exist
	certExists := fileExists(certFile)
	keyExists := fileExists(keyFile)

	// If either file doesn't exist, we need to write them
	if !certExists || !keyExists {
		if !certExists && !keyExists {
			return true, "files_missing"
		}
		if !certExists {
			return true, "cert_missing"
		}
		return true, "key_missing"
	}

	// Read existing files
	existingCert, err := os.ReadFile(certFile)
	if err != nil {
		return true, "read_error"
	}

	existingKey, err := os.ReadFile(keyFile)
	if err != nil {
		return true, "read_error"
	}

	// Compare certificate content (normalize whitespace)
	if !bytes.Equal(normalizePEM(existingCert), normalizePEM([]byte(newCert))) {
		return true, "cert_changed"
	}

	// Compare key content (normalize whitespace)
	if !bytes.Equal(normalizePEM(existingKey), normalizePEM([]byte(newKey))) {
		return true, "key_changed"
	}

	// Verify the key matches the certificate
	if !verifyKeyPairMatch(newCert, newKey) {
		return true, "key_mismatch"
	}

	// Everything matches, no update needed
	return false, "unchanged"
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// normalizePEM normalizes PEM content by trimming whitespace
func normalizePEM(pemData []byte) []byte {
	return bytes.TrimSpace(pemData)
}

// verifyKeyPairMatch verifies that the private key matches the certificate's public key
func verifyKeyPairMatch(certPEM, keyPEM string) bool {
	// Parse certificate
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}

	// Parse private key
	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil {
		return false
	}

	var privateKey interface{}
	var parseErr error

	// Try different key types
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		privateKey, parseErr = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		privateKey, parseErr = x509.ParseECPrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		privateKey, parseErr = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	default:
		return false
	}

	if parseErr != nil {
		return false
	}

	// Compare public keys
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		priv, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return false
		}
		return pub.N.Cmp(priv.N) == 0 && pub.E == priv.E
	case *ecdsa.PublicKey:
		priv, ok := privateKey.(*ecdsa.PrivateKey)
		if !ok {
			return false
		}
		return pub.X.Cmp(priv.X) == 0 && pub.Y.Cmp(priv.Y) == 0
	default:
		return false
	}
}

// API client functions

type apiDomainsResponse struct {
	Domains []string `json:"domains"`
	Count   int      `json:"count"`
}

type apiCertInfo struct {
	CommonName   string   `json:"common_name"`
	SANs         []string `json:"sans"`
	LeafCertPEM  string   `json:"leaf_cert_pem"`
	CertChainPEM string   `json:"cert_chain_pem"`
	CertChainURL string   `json:"cert_chain_url"`
	Issuers      []string `json:"issuers"`
	IssuedOn     string   `json:"issued_on"`
	ExpiresOn    string   `json:"expires_on"`
	Managed      bool     `json:"managed"`
	Status       string   `json:"status"`
	Error        string   `json:"error,omitempty"`
}

type apiPrivateKeyResponse struct {
	Domain     string `json:"domain"`
	PrivateKey string `json:"private_key"`
}

type apiErrorResponse struct {
	Error string `json:"error"`
}

func checkAPIHealth(apiURL string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("%s/healthz", apiURL))
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func getManagedDomains(apiURL string) ([]string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("%s/domains", apiURL))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result apiDomainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Domains, nil
}

func getAllCerts(apiURL string) (map[string]apiCertInfo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Get all certs
	resp, err := client.Get(fmt.Sprintf("%s/certs", apiURL))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var certs []apiCertInfo
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Build map of domain -> certInfo
	certMap := make(map[string]apiCertInfo)
	for _, cert := range certs {
		if cert.CommonName != "" {
			certMap[cert.CommonName] = cert
		}
	}

	return certMap, nil
}

func getPrivateKey(apiURL, domain string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("%s/certs/%s/private-key", apiURL, domain))
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp apiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return "", fmt.Errorf("API error: %s", errResp.Error)
	}

	var result apiPrivateKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.PrivateKey, nil
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

func generateTraefikConfig(configPath string, certDir string, domains []string, certInfoMap map[string]apiCertInfo) error {
	var certificates []traefikCertificate

	// Build certificate list
	for _, domain := range domains {
		// Check if certificate exists and is valid
		certInfo, found := certInfoMap[domain]
		if !found || !certInfo.Managed || certInfo.Status != "issued" && certInfo.Status != "" {
			continue
		}

		if certInfo.LeafCertPEM == "" {
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

	return nil
}
