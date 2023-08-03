package repo

import (
	"crypto/ecdsa"
	"encoding/hex"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/crypto"
	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/axiomesh/axiom-kit/fileutil"
)

func LoadNodeKey(repoRoot string) (*ecdsa.PrivateKey, error) {
	keyPath := path.Join(repoRoot, nodeKeyFileName)
	existConfig := fileutil.Exist(keyPath)
	if !existConfig {
		key, err := GenerateKey()
		if err != nil {
			return nil, err
		}
		if err := WriteKey(keyPath, key); err != nil {
			return nil, err
		}
		return key, nil
	}
	return ReadKey(keyPath)
}

// GenerateKey: use secp256k1
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

func P2PKeyFromECDSAKey(sk *ecdsa.PrivateKey) (ic.PrivKey, error) {
	return ic.UnmarshalSecp256k1PrivateKey(crypto.FromECDSA(sk))
}

func KeyToNodeID(sk *ecdsa.PrivateKey) (string, error) {
	pk, err := P2PKeyFromECDSAKey(sk)
	if err != nil {
		return "", err
	}

	id, err := peer.IDFromPublicKey(pk.GetPublic())
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func WriteKey(keyPath string, key *ecdsa.PrivateKey) error {
	keyBytes := hex.EncodeToString(crypto.FromECDSA(key))
	return os.WriteFile(keyPath, []byte(keyBytes), 0600)
}

func ParseKey(keyBytes []byte) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(string(keyBytes))
}

func ReadKey(keyPath string) (*ecdsa.PrivateKey, error) {
	keyFile, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	return ParseKey(keyFile)
}
