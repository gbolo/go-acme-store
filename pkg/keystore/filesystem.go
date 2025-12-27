package keystore

import (
	"encoding/json"
	"fmt"
	"go-acme-store/pkg/crypto"
	"go-acme-store/pkg/log"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// FilesystemKeystore is a filesystem-based implementation of the Keystore interface
type FilesystemKeystore struct {
	basePath       string
	accountFile    string
	domainsDir     string
	managedFile    string
	lock           *sync.RWMutex
}

// NewFilesystemKeystoreFromViper creates a new filesystem keystore from viper configuration
func NewFilesystemKeystoreFromViper(accountEmail string, needsAcmeAccount bool) (Keystore, error) {
	basePath := viper.GetString("filesystem.base_path")
	if basePath == "" {
		return nil, fmt.Errorf("filesystem.base_path not configured")
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %v", err)
	}

	domainsDir := filepath.Join(basePath, "domains")
	if err := os.MkdirAll(domainsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create domains directory: %v", err)
	}

	ks := &FilesystemKeystore{
		basePath:    basePath,
		accountFile: filepath.Join(basePath, "account.json"),
		domainsDir:  domainsDir,
		managedFile: filepath.Join(basePath, "managed-domains.json"),
		lock:        new(sync.RWMutex),
	}

	// Only handle ACME account if needed (e.g., for the daemon, not for the fetcher)
	if needsAcmeAccount {
		// Attempt to load existing account
		storedAccount, err := ks.loadAccount()
		if err != nil {
			return nil, err
		}
		if storedAccount != nil {
			log.Infof("loaded saved acme account %s from %s", storedAccount.Email, ks.accountFile)
			return ks, nil
		}

		// Create a new account key because one does not exist
		accountKeyPEM, err := crypto.GenerateECKeyPEM()
		if err != nil {
			return nil, err
		}

		err = ks.storeAccount(AcmeAccount{
			Email: accountEmail,
			Key:   accountKeyPEM,
		})
		if err == nil {
			log.Infof("account pki generated and stored for %s at %s", accountEmail, ks.accountFile)
		}
		return ks, err
	}

	return ks, nil
}

// loadAccount reads the ACME account from the filesystem
func (f *FilesystemKeystore) loadAccount() (*AcmeAccount, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	data, err := os.ReadFile(f.accountFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read account file: %v", err)
	}

	var account AcmeAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %v", err)
	}

	return &account, nil
}

// storeAccount writes the ACME account to the filesystem
func (f *FilesystemKeystore) storeAccount(account AcmeAccount) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account: %v", err)
	}

	if err := os.WriteFile(f.accountFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write account file: %v", err)
	}

	return nil
}

// GetAcmeAccount returns the stored ACME account
func (f *FilesystemKeystore) GetAcmeAccount() (AcmeAccount, error) {
	accountPtr, err := f.loadAccount()
	if err != nil {
		return AcmeAccount{}, err
	}
	if accountPtr == nil {
		return AcmeAccount{}, fmt.Errorf("no account found")
	}
	return *accountPtr, nil
}

// GetAllDomains returns a list of all domains that have certificates stored
func (f *FilesystemKeystore) GetAllDomains() (domains []string, err error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	entries, err := os.ReadDir(f.domainsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read domains directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			// Remove .json extension to get domain name
			domain := entry.Name()[:len(entry.Name())-5]
			domains = append(domains, domain)
		}
	}

	return domains, nil
}

// getDomainFilePath returns the file path for a domain's certificate data
func (f *FilesystemKeystore) getDomainFilePath(domain string) string {
	return filepath.Join(f.domainsDir, domain+".json")
}

// GetCertAndKey retrieves the certificate and key for a domain
func (f *FilesystemKeystore) GetCertAndKey(domain string) (data CertAndKey, err error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	filePath := f.getDomainFilePath(domain)
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return CertAndKey{}, nil
		}
		return CertAndKey{}, fmt.Errorf("failed to read certificate file: %v", err)
	}

	if err := json.Unmarshal(fileData, &data); err != nil {
		return CertAndKey{}, fmt.Errorf("failed to unmarshal certificate data: %v", err)
	}

	return data, nil
}

// GetLeafCert retrieves only the leaf certificate for a domain
func (f *FilesystemKeystore) GetLeafCert(domain string) (cert string, err error) {
	data, err := f.GetCertAndKey(domain)
	if err != nil {
		return "", err
	}
	return data.LeafCertPEM, nil
}

// StoreCertAndKey stores the certificate and key for a domain
func (f *FilesystemKeystore) StoreCertAndKey(domain string, data CertAndKey) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	filePath := f.getDomainFilePath(domain)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal certificate data: %v", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write certificate file: %v", err)
	}

	log.Infof("stored cert and key for %s -> %s", domain, filePath)
	return nil
}

// DeleteCertAndKey deletes the certificate and key for a domain
func (f *FilesystemKeystore) DeleteCertAndKey(domain string) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	filePath := f.getDomainFilePath(domain)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete certificate file: %v", err)
	}

	log.Infof("deleted cert and key for %s from %s", domain, filePath)
	return nil
}

// Domain management methods

type filesystemManagedDomains struct {
	Domains []ManagedDomain `json:"domains"`
}

// loadManagedDomains reads the managed domains list from the filesystem
func (f *FilesystemKeystore) loadManagedDomains() (*filesystemManagedDomains, error) {
	data, err := os.ReadFile(f.managedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &filesystemManagedDomains{Domains: []ManagedDomain{}}, nil
		}
		return nil, fmt.Errorf("failed to read managed domains file: %v", err)
	}

	var md filesystemManagedDomains
	if err := json.Unmarshal(data, &md); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managed domains: %v", err)
	}

	if md.Domains == nil {
		md.Domains = []ManagedDomain{}
	}

	return &md, nil
}

// storeManagedDomains writes the managed domains list to the filesystem
func (f *FilesystemKeystore) storeManagedDomains(md *filesystemManagedDomains) error {
	data, err := json.MarshalIndent(md, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal managed domains: %v", err)
	}

	if err := os.WriteFile(f.managedFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write managed domains file: %v", err)
	}

	return nil
}

// GetManagedDomains returns a list of all managed domains
func (f *FilesystemKeystore) GetManagedDomains() (domains []string, err error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	md, err := f.loadManagedDomains()
	if err != nil {
		return nil, err
	}

	for _, d := range md.Domains {
		domains = append(domains, d.Domain)
	}

	return domains, nil
}

// AddManagedDomain adds a domain to the managed domains list
func (f *FilesystemKeystore) AddManagedDomain(domain string) error {
	return f.AddManagedDomainWithSANs(domain, nil)
}

// AddManagedDomainWithSANs adds a domain with SANs to the managed domains list
func (f *FilesystemKeystore) AddManagedDomainWithSANs(domain string, sans []string) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	md, err := f.loadManagedDomains()
	if err != nil {
		return err
	}

	// Check if domain already exists
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

	if err := f.storeManagedDomains(md); err != nil {
		return err
	}

	if len(sans) > 0 {
		log.Infof("added managed domain: %s with SANs: %v", domain, sans)
	} else {
		log.Infof("added managed domain: %s", domain)
	}

	// Check if certificate exists and is unmanaged, if so, flip it to managed
	// Need to unlock before calling GetCertAndKey to avoid deadlock
	f.lock.Unlock()
	certAndKey, err := f.GetCertAndKey(domain)
	f.lock.Lock()

	if err == nil && certAndKey.CommonName != "" && !certAndKey.Managed {
		log.Infof("found existing unmanaged certificate for %s, marking as managed", domain)
		certAndKey.Managed = true

		// Temporarily unlock to call StoreCertAndKey
		f.lock.Unlock()
		err = f.StoreCertAndKey(domain, certAndKey)
		f.lock.Lock()

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

		// Temporarily unlock to call StoreCertAndKey
		f.lock.Unlock()
		err = f.StoreCertAndKey(domain, placeholder)
		f.lock.Lock()

		if err != nil {
			log.Warnf("failed to create placeholder cert for %s: %v", domain, err)
		} else {
			log.Infof("created pending placeholder for %s", domain)
		}
	}

	return nil
}

// RemoveManagedDomain removes a domain from the managed domains list
func (f *FilesystemKeystore) RemoveManagedDomain(domain string) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	md, err := f.loadManagedDomains()
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

	md.Domains = newDomains
	if err := f.storeManagedDomains(md); err != nil {
		return err
	}

	log.Infof("removed managed domain: %s", domain)
	return nil
}

// IsManagedDomain checks if a domain is in the managed domains list
func (f *FilesystemKeystore) IsManagedDomain(domain string) (bool, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	md, err := f.loadManagedDomains()
	if err != nil {
		return false, err
	}

	for _, d := range md.Domains {
		if d.Domain == domain {
			return true, nil
		}
	}

	return false, nil
}

