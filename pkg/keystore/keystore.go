package keystore

import (
	"fmt"
	"go-acme-store/pkg/crypto"
)

type Keystore interface {
	GetAcmeAccount() (AcmeAccount, error)
	GetAllDomains() (domains []string, err error)
	StoreCertAndKey(domain string, data CertAndKey) error
	GetCertAndKey(domain string) (data CertAndKey, err error)
	GetLeafCert(domain string) (cert string, err error)
	DeleteCertAndKey(domain string) error
	// Domain management methods
	GetManagedDomains() (domains []string, err error)
	AddManagedDomain(domain string) error
	AddManagedDomainWithSANs(domain string, sans []string) error
	RemoveManagedDomain(domain string) error
	IsManagedDomain(domain string) (bool, error)
}

type AcmeAccount struct {
	Email string `json:"email" mapstructure:"email"`
	Key   string `json:"key" mapstructure:"key"`
}

type CertAndKey struct {
	CertChainPEM  string   `json:"cert_chain_pem" mapstructure:"cert_chain_pem"`
	LeafCertPEM   string   `json:"leaf_cert_pem" mapstructure:"leaf_cert_pem"`
	CertChainURL  string   `json:"cert_chain_url" mapstructure:"cert_chain_url"`
	PrivateKeyPEM string   `json:"private_key_pem" mapstructure:"private_key_pem"`
	Issuers       []string `json:"issuers" mapstructure:"issuers"`
	IssuedOn      string   `json:"issued_on" mapstructure:"issued_on"`
	Expiration    string   `json:"expires_on" mapstructure:"expires_on"`
	CommonName    string   `json:"common_name" mapstructure:"common_name"`
	SANs          []string `json:"sans" mapstructure:"sans"`
	Managed       bool     `json:"managed" mapstructure:"managed"`
}

func (c *CertAndKey) populateMissingFields() (err error) {
	// decode cert
	cert := crypto.GetLeafCert(c.CertChainPEM)
	if cert == nil {
		return fmt.Errorf("could not decode certificate")
	}
	// populate
	c.LeafCertPEM = crypto.CertToPEM(cert)
	c.CommonName = cert.Subject.CommonName
	c.IssuedOn = cert.NotBefore.String()
	c.Expiration = cert.NotAfter.String()
	c.SANs = cert.DNSNames
	c.Issuers = crypto.GetIssuersCN(c.CertChainPEM)
	return
}

func NewCertAndKey(certChain, privateKey, certURL string) (CertAndKey, error) {
	data := CertAndKey{
		CertChainPEM:  certChain,
		CertChainURL:  certURL,
		PrivateKeyPEM: privateKey,
		Managed:       true, // New certificates are managed by default
	}
	err := data.populateMissingFields()
	return data, err
}
