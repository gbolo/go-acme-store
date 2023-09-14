package main

import (
	"go-acme-store/pkg/acme"
	"go-acme-store/pkg/log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var domains []string
var mutex sync.Mutex

func loadDomainsFromConfig() {
	mutex.Lock()
	defer mutex.Unlock()
	domains = viper.GetStringSlice("domains")
	log.Infof("loaded %d domain(s) from configuration: %v", len(domains), domains)
}

func acmeDaemon() {
	for {
		doAcmeOrders()
		// this daemon mostly sleeps ;)
		time.Sleep(1 * 24 * time.Hour)
	}
}

func doAcmeOrders() {
	mutex.Lock()
	defer mutex.Unlock()
	for _, domain := range domains {
		err := acme.RenewCertIfNeeded(ks, domain, nil)
		if err != nil {
			log.Errorf("failed to obtain certificate for domain %s: %v", domain, err)
		}
	}
}
