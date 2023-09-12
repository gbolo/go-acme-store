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
}

type AcmeAccount struct {
	Email string
	Key   string
}

type CertAndKey struct {
	CertChainPEM  string
	LeafCertPEM   string
	CertChainURL  string
	PrivateKeyPEM string
	Issuers       []string
	IssuedOn      string
	Expiration    string
	CommonName    string
	SANs          []string
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
	}
	err := data.populateMissingFields()
	return data, err
}
