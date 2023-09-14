package main

import (
	"go-acme-store/pkg/keystore"
	"go-acme-store/pkg/log"

	"github.com/spf13/viper"
)

var ks keystore.Keystore

func initKeystore() {
	mutex.Lock()
	defer mutex.Unlock()

	var err error
	ks, err = keystore.NewVaultKeystoreFromViper(viper.GetString("acme.account_email"))
	if err != nil {
		log.Fatalf("failed to init vault client: %v", err)
	}
	log.Infof("vault client was initialized")
}
