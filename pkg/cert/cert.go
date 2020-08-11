package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	ecdsa2 "github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
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

func ParsePrivateKey(data []byte, opt crypto.KeyType) (*ecdsa2.PrivateKey, error) {
	if data == nil {
		return nil, fmt.Errorf("empty data")
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("empty block")
	}

	return ecdsa2.UnmarshalPrivateKey(block.Bytes, opt)
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

func GenerateCert(privKey *ecdsa.PrivateKey, isCA bool, organization string) (*x509.Certificate, error) {
	sn, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, err
	}
	notBefore := time.Now().Add(-5 * time.Minute).UTC()

	template := &x509.Certificate{
		SerialNumber:          sn,
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(50 * 365 * 24 * time.Hour).UTC(),
		BasicConstraintsValid: true,
		IsCA:                  isCA,
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Locality:           []string{"HangZhou"},
			Province:           []string{"ZheJiang"},
			OrganizationalUnit: []string{"BitXHub"},
			Organization:       []string{organization},
			StreetAddress:      []string{"street", "address"},
			PostalCode:         []string{"324000"},
			CommonName:         "bitxhub.cn",
		},
	}
	template.SubjectKeyId = priKeyHash(privKey)

	return template, nil
}

func priKeyHash(priKey *ecdsa.PrivateKey) []byte {
	hash := sha256.New()

	_, err := hash.Write(elliptic.Marshal(priKey.Curve, priKey.PublicKey.X, priKey.PublicKey.Y))
	if err != nil {
		fmt.Printf("Get private key hash: %s", err.Error())
		return nil
	}

	return hash.Sum(nil)
}
