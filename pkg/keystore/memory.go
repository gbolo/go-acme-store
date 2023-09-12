package keystore

import (
	"sync"
)

// MemoryKeystore is an in-memory implementation of the Keystore interface
// NOTE: This should ONLY be used for testing purposes...
type MemoryKeystore struct {
	account     AcmeAccount
	certAndKeys map[string]CertAndKey
	lock        *sync.RWMutex
}

func NewMemoryKeystore(account AcmeAccount) Keystore {
	ks := MemoryKeystore{
		account:     account,
		certAndKeys: make(map[string]CertAndKey),
		lock:        new(sync.RWMutex),
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
