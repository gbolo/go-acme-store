package crypto

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// EncodePrivateKey encodes an ECDSA private key to PEM format.
func EncodePrivateKey(key *ecdsa.PrivateKey) (string, error) {
	derKey, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", err
	}

	keyBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derKey,
	}

	return string(pem.EncodeToMemory(keyBlock)), nil
}

// DecodePrivateKey decodes a PEM-encoded ECDSA private key.
func DecodePrivateKey(ecKeyPEM string) (*ecdsa.PrivateKey, error) {
	var skippedTypes []string
	var block *pem.Block
	encodedKey := []byte(ecKeyPEM)

	for {
		block, encodedKey = pem.Decode(encodedKey)

		if block == nil {
			return nil, fmt.Errorf("failed to find EC PRIVATE KEY in PEM data after skipping types %v", skippedTypes)
		}

		if block.Type == "EC PRIVATE KEY" {
			break
		} else {
			skippedTypes = append(skippedTypes, block.Type)
			continue
		}
	}

	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return privKey, nil
}
