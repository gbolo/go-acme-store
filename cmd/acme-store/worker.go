package main

import (
	"go-acme-store/pkg/acme"
	"go-acme-store/pkg/keystore"
	"go-acme-store/pkg/log"
	"sync"
	"time"
)

var mutex sync.Mutex
var triggerChan = make(chan bool, 1)

func loadDomainsFromVault() {
	domains, err := ks.GetManagedDomains()
	if err != nil {
		log.Warnf("failed to load managed domains from vault: %v", err)
		log.Warnf("if this is the first run, add domains via the API: POST /api/domains")
		return
	}
	log.Infof("loaded %d managed domain(s) from vault: %v", len(domains), domains)
}

func acmeDaemon() {
	// Do initial run
	doAcmeOrders()
	
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Regular 24-hour cycle
			doAcmeOrders()
		case <-triggerChan:
			// Manual trigger from API
			log.Infof("manual certificate renewal triggered")
			doAcmeOrders()
		}
	}
}

func triggerAcmeOrders() {
	// Non-blocking send to trigger channel
	select {
	case triggerChan <- true:
		log.Infof("ACME renewal trigger sent")
	default:
		log.Infof("ACME renewal already in progress")
	}
}

func doAcmeOrders() {
	mutex.Lock()
	defer mutex.Unlock()
	
	domains, err := ks.GetManagedDomains()
	if err != nil {
		log.Errorf("failed to get managed domains: %v", err)
		return
	}
	
	if len(domains) == 0 {
		log.Infof("no domains to manage")
		return
	}
	
	for _, domain := range domains {
		// Check if certificate exists and is managed
		certAndKey, err := ks.GetCertAndKey(domain)
		if err == nil && certAndKey.CommonName != "" && !certAndKey.Managed {
			log.Infof("skipping domain %s (marked as unmanaged)", domain)
			continue
		}
		
		// Get SANs for this domain (if any)
		var sans []string
		if vks, ok := ks.(*keystore.VaultKeystore); ok {
			managedDomain, err := vks.GetManagedDomainWithSANs(domain)
			if err == nil && managedDomain != nil && managedDomain.SANs != nil {
				sans = managedDomain.SANs
			}
		}
		
		acmeErr := acme.RenewCertIfNeeded(ks, domain, sans)
		if acmeErr != nil {
			log.Errorf("failed to obtain certificate for domain %s: %v", domain, acmeErr)
			
			// Update certificate status to failed with error message
			existingCert, getErr := ks.GetCertAndKey(domain)
			if getErr == nil && existingCert.CommonName != "" {
				existingCert.Status = "failed"
				existingCert.Error = acmeErr.Error()
				if updateErr := ks.StoreCertAndKey(domain, existingCert); updateErr != nil {
					log.Errorf("failed to update error status for %s: %v", domain, updateErr)
				} else {
					log.Infof("marked certificate for %s as failed", domain)
				}
			}
		}
	}
}
