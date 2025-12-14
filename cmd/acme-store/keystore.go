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
	ks, err = keystore.NewVaultKeystoreFromViper(viper.GetString("acme.account_email"), true)
	if err != nil {
		log.Errorf("failed to init vault client: %v", err)
		log.Warnf("application will start in degraded mode - vault operations will fail until vault is available")
		ks = nil
		return
	}
	log.Infof("vault client was initialized")
}

// ensureKeystore checks if keystore is initialized and returns an error if not
func ensureKeystore() error {
	if ks == nil {
		return fmt.Errorf("vault keystore not initialized - vault may be unavailable")
	}
	return nil
}

// keystoreRetryLoop attempts to reconnect to vault periodically if initialization fails
func keystoreRetryLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Only retry if keystore is not initialized
		if ks == nil {
			log.Infof("attempting to reconnect to vault...")
			initKeystore()
			if ks != nil {
				log.Infof("successfully reconnected to vault, loading domains...")
				loadDomainsFromVault()
			}
		}
	}
}
