package repo

import (
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	crypto2 "github.com/axiomesh/axiom-kit/crypto"
	"github.com/axiomesh/axiom-kit/crypto/asym"
	ecdsa2 "github.com/axiomesh/axiom-kit/crypto/asym/ecdsa"
	"github.com/libp2p/go-libp2p/core/crypto"
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

	nodeKeyData, err := os.ReadFile(filepath.Join(repoRoot, "certs/node.priv"))
	if err != nil {
		return nil, fmt.Errorf("read %s error: %w", filepath.Join(repoRoot, "certs/node.priv"), err)
	}

	nodePrivKey, err := ParsePrivateKey(nodeKeyData, crypto2.ECDSA_P256)
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

func ParsePrivateKey(data []byte, opt crypto2.KeyType) (*ecdsa2.PrivateKey, error) {
	if data == nil {
		return nil, fmt.Errorf("empty data")
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("empty block")
	}

	return ecdsa2.UnmarshalPrivateKey(block.Bytes, opt)
}

func GeneratePrivateKey() (*Key, error) {
	sk, err := asym.GenerateKeyPair(crypto2.Secp256k1)
	if err != nil {
		return nil, err
	}

	address, err := sk.PublicKey().Address()
	if err != nil {
		return nil, fmt.Errorf("get address from public key failed: %w", err)
	}

	libp2pPrivKey, _, err := crypto.ECDSAKeyPairFromKey(sk.(*ecdsa2.PrivateKey).K)
	if err != nil {
		return nil, fmt.Errorf("generate ecdsa key failed: %w", err)
	}

	return &Key{
		Address:       address.String(),
		PrivKey:       sk,
		Libp2pPrivKey: libp2pPrivKey,
	}, nil
}
