package repo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	libp2pcert "github.com/meshplus/go-libp2p-cert"
)

type Repo struct {
	Config        *Config
	NetworkConfig *NetworkConfig
	Key           *Key
	Certs         *libp2pcert.Certs
}

var (
	ConfigPath  string
	NetworkPath string
	OrderPath   string
)

func Load(repoRoot string, passwd string) (*Repo, error) {
	config, err := UnmarshalConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	var networkConfig *NetworkConfig
	if len(NetworkPath) == 0 {
		networkConfig, err = loadNetworkConfig(repoRoot, config.Genesis)
	} else {
		networkConfig, err = loadNetworkConfig(NetworkPath, config.Genesis)
	}
	if err != nil {
		return nil, fmt.Errorf("load network config: %w", err)
	}

	certs, err := libp2pcert.LoadCerts(repoRoot, config.NodeCertPath, config.AgencyCertPath, config.CACertPath)
	if err != nil {
		return nil, err
	}

	key, err := loadPrivKey(repoRoot, passwd)
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
