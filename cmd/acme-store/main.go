package main

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"go-acme-store/pkg/config"
	"go-acme-store/pkg/log"
	"go-acme-store/pkg/meta"
	"os"
)

func main() {
	appName := "acme-store"

	// Parse command-line flags
	configFile := flag.String("config", "", "path to configuration file (default: ./config.yml or CONFIG_FILE env var)")
	showVersion := flag.Bool("version", false, "display version information")
	flag.Parse()

	// Handle version flag first (before config loading)
	if *showVersion {
		fmt.Println(meta.GetAppMetadata().ToString())
		return
	}

	// Determine config file path with precedence: flag > env var > default
	cfg := *configFile
	if cfg == "" {
		cfg = os.Getenv("CONFIG_FILE")
	}

	// set config defaults then load config
	viper.SetDefault("server.bind_address", "0.0.0.0")
	viper.SetDefault("server.bind_port", "8080")
	viper.SetDefault("acme.directory", "https://acme-staging-v02.api.letsencrypt.org/directory")
	viper.SetDefault("acme.account_email", "acme-store@devnull")
	viper.SetDefault("vault.tls_skip_verify", false)
	viper.SetDefault("keystore.backend", "vault")
	config.MustInitViperAndLogger(appName, cfg)

	// ensure we have the required config values
	// checks that the config is correctly defined
	// these values cannot be empty
	keysThatCannotBeEmpty := []string{
		"acme.dns_provider",
	}

	// Add backend-specific required keys
	backend := viper.GetString("keystore.backend")
	switch backend {
	case "vault":
		keysThatCannotBeEmpty = append(keysThatCannotBeEmpty,
			"vault.address",
			"vault.kv2_mount",
			"vault.kv2_secret_path",
		)
	case "filesystem":
		keysThatCannotBeEmpty = append(keysThatCannotBeEmpty,
			"filesystem.base_path",
		)
	case "memory":
		// Memory backend has no required config
	default:
		log.Fatalf("unsupported keystore backend: %s (supported: vault, filesystem, memory)", backend)
	}

	for _, key := range keysThatCannotBeEmpty {
		if viper.GetString(key) == "" {
			log.Fatalf("%s cannot be empty", key)
		}
	}

	// initialize shared resources
	initKeystore()
	loadDomainsFromVault()

	// start vault retry loop in background
	go keystoreRetryLoop()

	// kick off the daemon on its own go routine
	go acmeDaemon()

	// configure and start http server
	httpServerDaemon(appName)
}
