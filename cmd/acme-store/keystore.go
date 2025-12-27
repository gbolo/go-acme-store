package main

import (
	"fmt"
	"go-acme-store/pkg/keystore"
	"go-acme-store/pkg/log"
	"time"

	"github.com/spf13/viper"
)

var ks keystore.Keystore

func initKeystore() {
	mutex.Lock()
	defer mutex.Unlock()

	var err error
	ks, err = keystore.NewKeystoreFromViper(viper.GetString("acme.account_email"), true)
	if err != nil {
		log.Errorf("failed to init keystore: %v", err)
		log.Warnf("application will start in degraded mode - keystore operations will fail until it is available")
		ks = nil
		return
	}
	backend := viper.GetString("keystore.backend")
	if backend == "" {
		backend = "vault"
	}
	log.Infof("%s keystore was initialized", backend)
}

// ensureKeystore checks if keystore is initialized and returns an error if not
func ensureKeystore() error {
	if ks == nil {
		return fmt.Errorf("keystore not initialized - keystore may be unavailable")
	}
	return nil
}

// keystoreRetryLoop attempts to reconnect to keystore periodically if initialization fails
func keystoreRetryLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Only retry if keystore is not initialized
		if ks == nil {
			log.Infof("attempting to reconnect to keystore...")
			initKeystore()
			if ks != nil {
				log.Infof("successfully reconnected to keystore, loading domains...")
				loadDomainsFromVault()
			}
		}
	}
}
