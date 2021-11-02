package repo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
)

type Key struct {
	Address       string             `json:"address"`
	PrivKey       crypto2.PrivateKey `json:"priv_key"`
	Libp2pPrivKey crypto.PrivKey
}

func LoadKey(path string) (*Key, error) {
	privKey, err := asym.RestorePrivateKey(path, "bitxhub")
	if err != nil {
		return nil, fmt.Errorf("restore private key failed: %w", err)
	}

	address, err := privKey.PublicKey().Address()
	if err != nil {
		return nil, fmt.Errorf("get address from public key failed: %w", err)
	}

	return &Key{
		Address: address.String(),
		PrivKey: privKey,
	}, nil
}

func loadPrivKey(repoRoot string, passwd string) (*Key, error) {
	if strings.TrimSpace(passwd) == "" {
		passwd = DefaultPasswd
	}

	privKey, err := asym.RestorePrivateKey(filepath.Join(repoRoot, KeyName), passwd)
	if err != nil {
		return nil, fmt.Errorf("restore private key failed: %w", err)
	}

	address, err := privKey.PublicKey().Address()
	if err != nil {
		return nil, fmt.Errorf("get address from public key failed: %w", err)
	}

	nodeKeyData, err := ioutil.ReadFile(filepath.Join(repoRoot, "certs/node.priv"))
	if err != nil {
		return nil, fmt.Errorf("read %s error: %w", filepath.Join(repoRoot, "certs/node.priv"), err)
	}

	nodePrivKey, err := libp2pcert.ParsePrivateKey(nodeKeyData, crypto2.ECDSA_P256)
	if err != nil {
		return nil, fmt.Errorf("parse private key failed: %w", err)
	}

	libp2pPrivKey, _, err := crypto.ECDSAKeyPairFromKey(nodePrivKey.K)
	if err != nil {
		return nil, fmt.Errorf("generate ecdsa key failed: %w", err)
	}

	return &Key{
		Address:       address.String(),
		PrivKey:       privKey,
		Libp2pPrivKey: libp2pPrivKey,
	}, nil
}
