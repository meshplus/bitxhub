package cert

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

func VerifySign(subCert *x509.Certificate, caCert *x509.Certificate) error {
	if err := subCert.CheckSignatureFrom(caCert); err != nil {
		return fmt.Errorf("check sign: %w", err)
	}

	if subCert.NotBefore.After(time.Now()) || subCert.NotAfter.Before(time.Now()) {
		return fmt.Errorf("cert expired")
	}

	return nil
}

func ParsePrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	if data == nil {
		return nil, fmt.Errorf("empty data")
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("empty block")
	}

	return x509.ParseECPrivateKey(block.Bytes)
}

func ParseCert(data []byte) (*x509.Certificate, error) {
	if data == nil {
		return nil, fmt.Errorf("empty data")
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("empty block")
	}

	return x509.ParseCertificate(block.Bytes)
}
