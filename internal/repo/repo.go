package repo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type Repo struct {
	Config        *Config
	NetworkConfig *NetworkConfig
	Key           *Key
	Certs         *Certs
}

func Load(repoRoot string) (*Repo, error) {
	config, err := UnmarshalConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	networkConfig, err := loadNetworkConfig(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("load network config: %w", err)
	}

	certs, err := loadCerts(repoRoot)
	if err != nil {
		return nil, err
	}

	key, err := loadPrivKey(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	return &Repo{
		Config:        config,
		NetworkConfig: networkConfig,
		Key:           key,
		Certs:         certs,
	}, nil
}

func GetAPI(repoRoot string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join(repoRoot, APIName))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func GetKeyPath(repoRoot string) string {
	return filepath.Join(repoRoot, KeyName)
}

func GetStoragePath(repoRoot string, subPath ...string) string {
	p := filepath.Join(repoRoot, "storage")
	for _, s := range subPath {
		p = filepath.Join(p, s)
	}

	return p
}
