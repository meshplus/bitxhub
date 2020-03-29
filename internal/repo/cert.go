package repo

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/meshplus/bitxhub/pkg/cert"
)

type Certs struct {
	NodeCertData   []byte
	AgencyCertData []byte
	CACertData     []byte
	NodeCert       *x509.Certificate
	AgencyCert     *x509.Certificate
	CACert         *x509.Certificate
}

func loadCerts(repoRoot string) (*Certs, error) {
	nodeCert, nodeCertData, err := loadCert(filepath.Join(repoRoot, "certs/node.cert"))
	if err != nil {
		return nil, fmt.Errorf("load node cert: %w", err)
	}

	agencyCert, agencyCertData, err := loadCert(filepath.Join(repoRoot, "certs/agency.cert"))
	if err != nil {
		return nil, fmt.Errorf("load agency cert: %w", err)
	}
	caCert, caCertData, err := loadCert(filepath.Join(repoRoot, "certs/ca.cert"))
	if err != nil {
		return nil, fmt.Errorf("load ca cert: %w", err)
	}

	return &Certs{
		NodeCertData:   nodeCertData,
		AgencyCertData: agencyCertData,
		CACertData:     caCertData,
		NodeCert:       nodeCert,
		AgencyCert:     agencyCert,
		CACert:         caCert,
	}, nil
}

func loadCert(certPath string) (*x509.Certificate, []byte, error) {
	data, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read cert: %w", err)
	}

	cert, err := cert.ParseCert(data)
	if err != nil {
		return nil, nil, fmt.Errorf("parse cert: %w", err)
	}

	return cert, data, nil
}
