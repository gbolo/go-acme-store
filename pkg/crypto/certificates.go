package crypto

import (
	"crypto/x509"
	"encoding/pem"
	"time"
)

// GetCerts will return ALL valid x509 certs
func GetCerts(certChain string) (certs []*x509.Certificate) {
	certPEMBlock := []byte(certChain)
	var certDERBlock *pem.Block
	// NOTE: this will recursively rewrite certPEMBlock
	//       for each pem block found in the string
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			x509Cert, err := x509.ParseCertificate(certDERBlock.Bytes)
			if err != nil {
				continue
			}
			certs = append(certs, x509Cert)
		}
	}
	return
}

// GetLeafCert will return the FIRST valid x509 certificate it finds
// in a chain of certs that is NOT a CA certificate
func GetLeafCert(certChain string) *x509.Certificate {
	for _, cert := range GetCerts(certChain) {
		if !cert.IsCA {
			return cert
		}
	}
	return nil
}

// CheckIfCertNeedsUpdate checks if the first leaf cert founds needs to be renewed.
// NOTE: will default to true if garbage data is given
func CheckIfCertNeedsUpdate(certChain string, deadlineDays int) bool {
	// TODO: make this configurable? for now its 21 days
	deadline := time.Now().Add(time.Duration(deadlineDays) * 24 * time.Hour)
	leafCert := GetLeafCert(certChain)
	if leafCert != nil {
		if !deadline.After(leafCert.NotAfter) {
			return false
		}
	}
	return true
}

func DaysRemainingUntilCertExpires(certChain string) (days float64) {
	remainingHours := 0.0
	leafCert := GetLeafCert(certChain)
	if leafCert != nil {
		remainingHours = leafCert.NotAfter.Sub(time.Now()).Hours()
	}
	return remainingHours / 24
}

func GetIssuersCN(certChain string) (issuersCN []string) {
	for _, cert := range GetCerts(certChain) {
		if cert.IsCA {
			issuersCN = append(issuersCN, cert.Subject.CommonName)
		}
	}
	return
}

// CertToPEM converts a x509 cert to a pem string
func CertToPEM(cert *x509.Certificate) string {
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	return string(pemBytes)
}
