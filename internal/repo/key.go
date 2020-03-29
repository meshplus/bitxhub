package repo

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p-core/crypto"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/fileutil"
	"github.com/meshplus/bitxhub-kit/key"
	"github.com/meshplus/bitxhub/pkg/cert"
	"github.com/tidwall/gjson"
)

type Key struct {
	PID           string             `json:"pid"`
	Address       string             `json:"address"`
	PrivKey       crypto2.PrivateKey `json:"priv_key"`
	Libp2pPrivKey crypto.PrivKey
}

func LoadKey(path string) (*Key, error) {
	keyPath := filepath.Join(path)
	data, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	pid := gjson.GetBytes(data, "pid")
	address := gjson.GetBytes(data, "address")
	privKeyString := gjson.GetBytes(data, "priv_key")

	fmt.Println(string(data))
	fmt.Println(address)
	libp2pPrivKeyData, err := crypto.ConfigDecodeKey(privKeyString.String())
	if err != nil {
		return nil, err
	}

	libp2pPrivKey, err := crypto.UnmarshalPrivateKey(libp2pPrivKeyData)
	if err != nil {
		return nil, err
	}

	raw, err := libp2pPrivKey.Raw()
	if err != nil {
		return nil, err
	}

	privKey, err := x509.ParseECPrivateKey(raw)
	if err != nil {
		return nil, err
	}

	return &Key{
		PID:           pid.String(),
		Address:       address.String(),
		PrivKey:       &ecdsa.PrivateKey{K: privKey},
		Libp2pPrivKey: libp2pPrivKey,
	}, nil
}

func loadPrivKey(repoRoot string) (*Key, error) {
	data, err := ioutil.ReadFile(filepath.Join(repoRoot, "certs/node.priv"))
	if err != nil {
		return nil, err
	}

	stdPriv, err := cert.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}

	privKey := &ecdsa.PrivateKey{K: stdPriv}

	address, err := privKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	libp2pPrivKey, _, err := crypto.ECDSAKeyPairFromKey(stdPriv)
	if err != nil {
		return nil, err
	}

	pid := gjson.Get(string(data), "pid").String()

	keyPath := filepath.Join(repoRoot, KeyName)

	if !fileutil.Exist(keyPath) {
		k, err := key.NewWithPrivateKey(privKey, "bitxhub")
		if err != nil {
			return nil, err
		}

		data, err := k.Pretty()
		if err != nil {
			return nil, err
		}

		if err := ioutil.WriteFile(keyPath, []byte(data), os.ModePerm); err != nil {
			return nil, err
		}
	}

	return &Key{
		PID:           pid,
		Address:       address.Hex(),
		PrivKey:       privKey,
		Libp2pPrivKey: libp2pPrivKey,
	}, nil
}
