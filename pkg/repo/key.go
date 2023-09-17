package repo

import (
	"crypto/ecdsa"
	"encoding/hex"
	"os"
	"path"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/axiomesh/axiom-kit/fileutil"
)

func loadKey(repoRoot string, keyFileName string) (*ecdsa.PrivateKey, error) {
	keyPath := path.Join(repoRoot, keyFileName)
	if !fileutil.Exist(keyPath) {
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

func LoadP2PKey(repoRoot string) (*ecdsa.PrivateKey, error) {
	return loadKey(repoRoot, p2pKeyFileName)
}

func LoadAccountKey(repoRoot string) (*ecdsa.PrivateKey, error) {
	return loadKey(repoRoot, AccountKeyFileName)
}

// GenerateKey use secp256k1
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ethcrypto.GenerateKey()
}

func Libp2pKeyFromECDSAKey(sk *ecdsa.PrivateKey) (libp2pcrypto.PrivKey, error) {
	return libp2pcrypto.UnmarshalSecp256k1PrivateKey(ethcrypto.FromECDSA(sk))
}

func KeyToNodeID(sk *ecdsa.PrivateKey) (string, error) {
	pk, err := Libp2pKeyFromECDSAKey(sk)
	if err != nil {
		return "", err
	}

	id, err := peer.IDFromPublicKey(pk.GetPublic())
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func KeyString(key *ecdsa.PrivateKey) string {
	return hex.EncodeToString(ethcrypto.FromECDSA(key))
}

func WriteKey(keyPath string, key *ecdsa.PrivateKey) error {
	return os.WriteFile(keyPath, []byte(KeyString(key)), 0600)
}

func ParseKey(keyBytes []byte) (*ecdsa.PrivateKey, error) {
	return ethcrypto.HexToECDSA(string(keyBytes))
}

func ReadKey(keyPath string) (*ecdsa.PrivateKey, error) {
	keyFile, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	return ParseKey(keyFile)
}
