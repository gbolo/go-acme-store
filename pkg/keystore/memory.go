package keystore

import (
	"fmt"
	"sync"
)

// MemoryKeystore is an in-memory implementation of the Keystore interface
// NOTE: This should ONLY be used for testing purposes...
type MemoryKeystore struct {
	account        AcmeAccount
	certAndKeys    map[string]CertAndKey
	managedDomains []string
	lock           *sync.RWMutex
}

func NewMemoryKeystore(account AcmeAccount) Keystore {
	ks := MemoryKeystore{
		account:        account,
		certAndKeys:    make(map[string]CertAndKey),
		managedDomains: []string{},
		lock:           new(sync.RWMutex),
	}
	return &ks
}

func (m *MemoryKeystore) GetAcmeAccount() (AcmeAccount, error) {
	return m.account, nil
}

func (m *MemoryKeystore) GetAllDomains() (domains []string, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	for domain := range m.certAndKeys {
		domains = append(domains, domain)
	}
	return
}

func (m *MemoryKeystore) GetCertAndKey(domain string) (data CertAndKey, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if value, ok := m.certAndKeys[domain]; ok {
		data = value
	}
	return
}

func (m *MemoryKeystore) GetLeafCert(domain string) (cert string, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if value, ok := m.certAndKeys[domain]; ok {
		cert = value.LeafCertPEM
	}
	return
}

func (m *MemoryKeystore) StoreCertAndKey(domain string, data CertAndKey) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.certAndKeys[domain] = data
	return nil
}

func (m *MemoryKeystore) DeleteCertAndKey(domain string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.certAndKeys, domain)
	return nil
}

// Domain management methods

func (m *MemoryKeystore) GetManagedDomains() (domains []string, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	// Return a copy to avoid external modifications
	domains = make([]string, len(m.managedDomains))
	copy(domains, m.managedDomains)
	return domains, nil
}

func (m *MemoryKeystore) AddManagedDomain(domain string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	
	// Check if domain already exists
	for _, d := range m.managedDomains {
		if d == domain {
			return fmt.Errorf("domain %s is already managed", domain)
		}
	}
	
	m.managedDomains = append(m.managedDomains, domain)
	
	// Check if certificate exists and is unmanaged, if so, flip it to managed
	if cert, exists := m.certAndKeys[domain]; exists && !cert.Managed {
		cert.Managed = true
		m.certAndKeys[domain] = cert
	}
	
	return nil
}

func (m *MemoryKeystore) RemoveManagedDomain(domain string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	
	// Find and remove the domain
	found := false
	newDomains := []string{}
	for _, d := range m.managedDomains {
		if d == domain {
			found = true
			continue
		}
		newDomains = append(newDomains, d)
	}
	
	if !found {
		return fmt.Errorf("domain %s is not managed", domain)
	}
	
	m.managedDomains = newDomains
	return nil
}

func (m *MemoryKeystore) IsManagedDomain(domain string) (bool, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	
	for _, d := range m.managedDomains {
		if d == domain {
			return true, nil
		}
	}
	
	return false, nil
}
