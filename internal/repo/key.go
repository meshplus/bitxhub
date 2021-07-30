package repo

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type Key struct {
	Address       string             `json:"address"`
	PrivKey       crypto2.PrivateKey `json:"priv_key"`
	Libp2pPrivKey crypto.PrivKey
}

func LoadKey(path string) (*Key, error) {
	privKey, err := asym.RestorePrivateKey(path, "bitxhub")
	if err != nil {
		return nil, err
	}

	address, err := privKey.PublicKey().Address()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	address, err := privKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	nodeKeyData, err := ioutil.ReadFile(filepath.Join(repoRoot, "certs/node.priv"))
	if err != nil {
		return nil, err
	}

	nodePrivKey, err := libp2pcert.ParsePrivateKey(nodeKeyData, crypto2.ECDSA_P256)
	if err != nil {
		return nil, err
	}

	libp2pPrivKey, _, err := crypto.ECDSAKeyPairFromKey(nodePrivKey.K)
	if err != nil {
		return nil, err
	}

	return &Key{
		Address:       address.String(),
		PrivKey:       privKey,
		Libp2pPrivKey: libp2pPrivKey,
	}, nil
}
