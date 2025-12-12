package main

import (
	"flag"
	"go-acme-store/pkg/config"
	"go-acme-store/pkg/log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func main() {
	appName := "acme-store"

	// Parse command-line flags
	configFile := flag.String("config", "", "path to configuration file (default: ./config.yml or CONFIG_FILE env var)")
	flag.Parse()

	// Determine config file path with precedence: flag > env var > default
	cfg := *configFile
	if cfg == "" {
		cfg = os.Getenv("CONFIG_FILE")
	}
	if cfg == "" {
		cfg = "./config.yml"
	}

	config.MustInitViperAndLogger(appName, cfg)

	// initialize shared resources
	initKeystore()
	loadDomainsFromVault()

	// watch for config changes to re-initialize keystore
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Warnf("config change detected")
		initKeystore()
	})

	// kick off the daemon on its own go routine
	go acmeDaemon()

	// configure and start http server
	httpServerDaemon(appName)
}
