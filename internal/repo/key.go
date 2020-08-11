package repo

import (
	"io/ioutil"
	"path/filepath"

	"github.com/libp2p/go-libp2p-core/crypto"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/fileutil"
	"github.com/meshplus/bitxhub/pkg/cert"
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

func loadPrivKey(repoRoot string) (*Key, error) {
	keyData, err := ioutil.ReadFile(filepath.Join(repoRoot, "certs/key.priv"))
	if err != nil {
		return nil, err
	}

	privKey, err := cert.ParsePrivateKey(keyData, crypto2.Secp256k1)
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

	nodePrivKey, err := cert.ParsePrivateKey(nodeKeyData, crypto2.ECDSA_P256)
	if err != nil {
		return nil, err
	}

	libp2pPrivKey, _, err := crypto.ECDSAKeyPairFromKey(nodePrivKey.K)
	if err != nil {
		return nil, err
	}

	keyPath := filepath.Join(repoRoot, KeyName)

	if !fileutil.Exist(keyPath) {
		privKey, err := asym.PrivateKeyFromStdKey(privKey.K)
		if err != nil {
			return nil, err
		}

		if err := asym.StorePrivateKey(privKey, keyPath, "bitxhub"); err != nil {
			return nil, err
		}
	}

	return &Key{
		Address:       address.Hex(),
		PrivKey:       privKey,
		Libp2pPrivKey: libp2pPrivKey,
	}, nil
}
