package keystore

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go-acme-store/pkg/crypto"
	"go-acme-store/pkg/log"
	"net/http"

	vault "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type VaultKeystore struct {
	client     *vault.Client
	mountPath  string
	secretPath string
}

func (v *VaultKeystore) getBasePath() string {
	return fmt.Sprintf("%s/data/%s", v.mountPath, v.secretPath)
}

func (v *VaultKeystore) getAccountPath() string {
	return fmt.Sprintf("%s/account", v.getBasePath())
}

func (v *VaultKeystore) getDomainListPath() string {
	return fmt.Sprintf("%s/metadata/%s/domain", v.mountPath, v.secretPath)
}

func (v *VaultKeystore) getDomainPath(domain string) string {
	return fmt.Sprintf("%s/domain/%s", v.getBasePath(), domain)
}

func (v *VaultKeystore) getManagedDomainsPath() string {
	return fmt.Sprintf("%s/managed-domains", v.getBasePath())
}

func (v *VaultKeystore) storeAccount(account AcmeAccount) (err error) {
	vaultData, err := encodeForVault(account)
	if err != nil {
		return
	}
	result, err := v.client.Logical().Write(v.getAccountPath(), vaultData)
	if err == nil && result != nil {
		log.Infof("stored account info for %s -> %s version %v", account.Email, v.getAccountPath(), result.Data["version"])
	}
	return
}

func (v *VaultKeystore) getAccount() (account *AcmeAccount, err error) {
	err = v.readDataIntoInterface(v.getAccountPath(), &account)
	return
}

func (v *VaultKeystore) readDataIntoInterface(path string, destInterface interface{}) (err error) {
	vaultData, err := v.client.Logical().Read(path)
	if err != nil {
		return
	}
	if vaultData == nil {
		log.Debugf("no secrets found in path %s", path)
		return nil
	}

	if _, exists := vaultData.Data["data"]; !exists {
		log.Debugf("no data found in secret path %s", path)
		return nil
	}

	log.Debugf("loaded data from secret path %s", path)
	err = mapstructure.Decode(vaultData.Data["data"], &destInterface)
	return
}

func NewVaultKeystoreFromViper(accountEmail string, needsAcmeAccount bool) (Keystore, error) {
	config := vault.DefaultConfig()
	config.Address = viper.GetString("vault.address")

	// Configure TLS settings if skip verify is enabled
	if viper.GetBool("vault.tls_skip_verify") {
		log.Warnf("Vault TLS certificate verification is DISABLED - not recommended for production")
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		config.HttpClient.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	client, _ := vault.NewClient(config)

	// set a token only if one is specified
	if viper.GetString("vault.token") != "" {
		client.SetToken(viper.GetString("vault.token"))
	}

	ks := VaultKeystore{
		client:     client,
		mountPath:  viper.GetString("vault.kv2_mount"),
		secretPath: viper.GetString("vault.kv2_secret_path"),
	}

	// Only handle ACME account if needed (e.g., for the daemon, not for the fetcher)
	if needsAcmeAccount {
		// attempt to reuse existing acme account
		storedAccount, err := ks.getAccount()
		if err != nil {
			return nil, err
		}
		if storedAccount != nil {
			log.Infof("loaded saved acme account %s from %s", storedAccount.Email, ks.getAccountPath())
			return &ks, nil
		}

		// create a new account key because one does not exist
		accountKeyPEM, err := crypto.GenerateECKeyPEM()
		if err != nil {
			return nil, err
		}

		err = ks.storeAccount(AcmeAccount{
			Email: accountEmail,
			Key:   accountKeyPEM,
		})
		if err == nil {
			log.Infof("account pki generated and stored for %s", accountEmail)
		}
		return &ks, err
	}
	return &ks, nil
}

func (v *VaultKeystore) GetAcmeAccount() (account AcmeAccount, err error) {
	accountPtr, err := v.getAccount()
	if accountPtr != nil {
		account = *accountPtr
	}
	return
}

func (v *VaultKeystore) GetAllDomains() (domains []string, err error) {
	result, err := v.client.Logical().List(v.getDomainListPath())
	if err != nil {
		return
	}

	// Return empty list if result is nil (path doesn't exist)
	if result == nil {
		return []string{}, nil
	}

	// Return empty list if no keys exist
	if _, exists := result.Data["keys"]; !exists {
		return []string{}, nil
	}

	for _, domain := range result.Data["keys"].([]interface{}) {
		domains = append(domains, fmt.Sprintf("%v", domain))
	}
	return
}

func (v *VaultKeystore) GetCertAndKey(domain string) (data CertAndKey, err error) {
	err = v.readDataIntoInterface(v.getDomainPath(domain), &data)
	return
}

func (v *VaultKeystore) GetLeafCert(domain string) (cert string, err error) {
	data, err := v.GetCertAndKey(domain)
	if err != nil {
		return
	}
	cert = data.LeafCertPEM
	return
}

func (v *VaultKeystore) StoreCertAndKey(domain string, data CertAndKey) (err error) {
	vaultData, err := encodeForVault(data)
	if err != nil {
		return
	}
	result, err := v.client.Logical().Write(v.getDomainPath(domain), vaultData)
	if err == nil {
		log.Infof("stored cert and key for %s -> %s version %v", domain, v.getDomainPath(domain), result.Data["version"])
	}
	return
}

func (v *VaultKeystore) DeleteCertAndKey(domain string) error {
	_, err := v.client.Logical().Delete(v.getDomainPath(domain))
	if err != nil {
		return fmt.Errorf("failed to delete cert for domain %s: %v", domain, err)
	}
	log.Infof("deleted cert and key for %s from %s", domain, v.getDomainPath(domain))
	return nil
}

// Domain management methods

type ManagedDomain struct {
	Domain string   `json:"domain" mapstructure:"domain"`
	SANs   []string `json:"sans,omitempty" mapstructure:"sans"`
}

func (v *VaultKeystore) GetManagedDomains() (domains []string, err error) {
	type managedDomains struct {
		Domains []ManagedDomain `json:"domains" mapstructure:"domains"`
	}
	var md managedDomains
	err = v.readDataIntoInterface(v.getManagedDomainsPath(), &md)
	if err != nil {
		return nil, err
	}
	if md.Domains == nil {
		return []string{}, nil
	}
	// Return just the domain names for backward compatibility
	for _, d := range md.Domains {
		domains = append(domains, d.Domain)
	}
	return domains, nil
}

func (v *VaultKeystore) GetManagedDomainWithSANs(domain string) (*ManagedDomain, error) {
	type managedDomains struct {
		Domains []ManagedDomain `json:"domains" mapstructure:"domains"`
	}
	var md managedDomains
	err := v.readDataIntoInterface(v.getManagedDomainsPath(), &md)
	if err != nil {
		return nil, err
	}
	for _, d := range md.Domains {
		if d.Domain == domain {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("domain %s not found", domain)
}

func (v *VaultKeystore) AddManagedDomain(domain string) error {
	return v.AddManagedDomainWithSANs(domain, nil)
}

func (v *VaultKeystore) AddManagedDomainWithSANs(domain string, sans []string) error {
	// Get existing managed domains with SANs
	type managedDomains struct {
		Domains []ManagedDomain `json:"domains" mapstructure:"domains"`
	}
	var md managedDomains
	err := v.readDataIntoInterface(v.getManagedDomainsPath(), &md)
	if err != nil {
		// If it doesn't exist yet, create empty list
		md.Domains = []ManagedDomain{}
	}

	// Check if domain already exists in managed list
	for _, d := range md.Domains {
		if d.Domain == domain {
			return fmt.Errorf("domain %s is already managed", domain)
		}
	}

	// Add new domain with SANs
	newDomain := ManagedDomain{
		Domain: domain,
		SANs:   sans,
	}
	md.Domains = append(md.Domains, newDomain)

	// Store back to vault
	vaultData, err := encodeForVault(map[string]interface{}{
		"domains": md.Domains,
	})
	if err != nil {
		return err
	}

	_, err = v.client.Logical().Write(v.getManagedDomainsPath(), vaultData)
	if err != nil {
		return fmt.Errorf("failed to store managed domains: %v", err)
	}

	if len(sans) > 0 {
		log.Infof("added managed domain: %s with SANs: %v", domain, sans)
	} else {
		log.Infof("added managed domain: %s", domain)
	}

	// Check if certificate exists and is unmanaged, if so, flip it to managed
	certAndKey, err := v.GetCertAndKey(domain)
	if err == nil && certAndKey.CommonName != "" && !certAndKey.Managed {
		log.Infof("found existing unmanaged certificate for %s, marking as managed", domain)
		certAndKey.Managed = true
		err = v.StoreCertAndKey(domain, certAndKey)
		if err != nil {
			log.Warnf("failed to update certificate managed status for %s: %v", domain, err)
		} else {
			log.Infof("certificate for %s is now managed and will be renewed", domain)
		}
	} else if err != nil || certAndKey.CommonName == "" {
		// No existing certificate, create a placeholder with pending status
		placeholder := CertAndKey{
			CommonName: domain,
			SANs:       sans,
			Managed:    true,
			Status:     "pending",
		}
		err = v.StoreCertAndKey(domain, placeholder)
		if err != nil {
			log.Warnf("failed to create placeholder cert for %s: %v", domain, err)
		} else {
			log.Infof("created pending placeholder for %s", domain)
		}
	}

	return nil
}

func (v *VaultKeystore) RemoveManagedDomain(domain string) error {
	// Get existing managed domains with SANs
	type managedDomains struct {
		Domains []ManagedDomain `json:"domains" mapstructure:"domains"`
	}
	var md managedDomains
	err := v.readDataIntoInterface(v.getManagedDomainsPath(), &md)
	if err != nil {
		return err
	}

	// Find and remove the domain
	found := false
	newDomains := []ManagedDomain{}
	for _, d := range md.Domains {
		if d.Domain == domain {
			found = true
			continue
		}
		newDomains = append(newDomains, d)
	}

	if !found {
		return fmt.Errorf("domain %s is not managed", domain)
	}

	// Store back to vault
	vaultData, err := encodeForVault(map[string]interface{}{
		"domains": newDomains,
	})
	if err != nil {
		return err
	}

	_, err = v.client.Logical().Write(v.getManagedDomainsPath(), vaultData)
	if err != nil {
		return fmt.Errorf("failed to store managed domains: %v", err)
	}

	log.Infof("removed managed domain: %s", domain)
	return nil
}

func (v *VaultKeystore) IsManagedDomain(domain string) (bool, error) {
	domains, err := v.GetManagedDomains()
	if err != nil {
		return false, err
	}

	for _, d := range domains {
		if d == domain {
			return true, nil
		}
	}

	return false, nil
}

func encodeForVault(input interface{}) (output map[string]interface{}, err error) {
	var inputData map[string]interface{}
	dataBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("could not marshal json: %v", err)
	}
	err = json.Unmarshal(dataBytes, &inputData)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %v", err)
	}
	output = make(map[string]interface{})
	output["data"] = inputData
	return
}
