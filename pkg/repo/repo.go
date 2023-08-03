package repo

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"path/filepath"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
)

type Repo struct {
	Config        *Config
	NetworkConfig *NetworkConfig
	OrderConfig   *OrderConfig
	NodeKey       *ecdsa.PrivateKey
	P2PKey        libp2pcrypto.PrivKey
	NodeAddress   string

	ConfigChangeFeed event.Feed
}

type signerOpts struct {
}

func (*signerOpts) HashFunc() crypto.Hash {
	return crypto.SHA3_256
}

var signOpt = &signerOpts{}

func (r *Repo) NodeKeySign(data []byte) ([]byte, error) {
	return r.NodeKey.Sign(rand.Reader, data, signOpt)
}

// TODO: need support? remove it
func (r *Repo) SubscribeConfigChange(ch chan *Repo) event.Subscription {
	return r.ConfigChangeFeed.Subscribe(ch)
}

func (r *Repo) Flush() error {
	if err := writeConfig(path.Join(r.Config.RepoRoot, cfgFileName), r.Config); err != nil {
		return errors.Wrap(err, "failed to write config")
	}
	if err := writeConfig(path.Join(r.Config.RepoRoot, networkCfgFileName), r.NetworkConfig); err != nil {
		return errors.Wrap(err, "failed to write network config")
	}
	if err := writeConfig(path.Join(r.Config.RepoRoot, orderCfgFileName), r.OrderConfig); err != nil {
		return errors.Wrap(err, "failed to write order config")
	}
	if err := WriteKey(path.Join(r.Config.RepoRoot, nodeKeyFileName), r.NodeKey); err != nil {
		return errors.Wrap(err, "failed to write node key")
	}
	return nil
}

func writeConfig(cfgPath string, config any) error {
	raw, err := MarshalConfig(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(cfgPath, []byte(raw), 0755); err != nil {
		return err
	}
	return nil
}

func MarshalConfig(config any) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	e := toml.NewEncoder(buf)
	e.SetIndentTables(true)
	e.SetArraysMultiline(true)
	err := e.Encode(config)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Default(repoRoot string) (*Repo, error) {
	return DefaultWithNodeIndex(repoRoot, 0)
}

func DefaultWithNodeIndex(repoRoot string, nodeIndex int) (*Repo, error) {
	var key *ecdsa.PrivateKey
	var err error
	if nodeIndex < 0 || nodeIndex > len(defaultNodeKeys)-1 {
		key, err = GenerateKey()
		nodeIndex = 0
	} else {
		key, err = ParseKey([]byte(defaultNodeKeys[nodeIndex]))
	}
	if err != nil {
		return nil, err
	}

	p2pKey, err := P2PKeyFromECDSAKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ecdsa key : %w", err)
	}
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	cfg := DefaultConfig(repoRoot)
	networkCfg := DefaultNetworkConfig()
	networkCfg.Genesis = cfg.Genesis
	networkCfg.ID = uint64(nodeIndex + 1)
	networkCfg.LocalAddr = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 4001+nodeIndex)
	return &Repo{
		Config:        cfg,
		NetworkConfig: networkCfg,
		OrderConfig:   DefaultOrderConfig(),
		NodeKey:       key,
		P2PKey:        p2pKey,
		NodeAddress:   addr.String(),
	}, nil
}

// load config from the repo, which is automatically initialized when the repo is empty
func Load(repoRoot string) (*Repo, error) {
	cfg, err := LoadConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	networkCfg, err := LoadNetworkConfig(cfg.RepoRoot, cfg.Genesis)
	if err != nil {
		return nil, err
	}
	orderCfg, err := LoadOrderConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	key, err := LoadNodeKey(cfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load node key: %w", err)
	}
	p2pKey, err := P2PKeyFromECDSAKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ecdsa key : %w", err)
	}
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)
	id, err := KeyToNodeID(key)
	if err != nil {
		return nil, err
	}
	networkCfg.Nodes[networkCfg.ID-1].Pid = id
	networkCfg.Nodes[networkCfg.ID-1].Account = addr.String()

	if err := writeConfig(path.Join(repoRoot, networkCfgFileName), networkCfg); err != nil {
		return nil, errors.Wrap(err, "failed to write network config")
	}

	repo := &Repo{
		Config:        cfg,
		NetworkConfig: networkCfg,
		OrderConfig:   orderCfg,
		NodeKey:       key,
		P2PKey:        p2pKey,
		NodeAddress:   addr.String(),
	}

	return repo, nil
}

func GetStoragePath(repoRoot string, subPath ...string) string {
	p := filepath.Join(repoRoot, "storage")
	for _, s := range subPath {
		p = filepath.Join(p, s)
	}

	return p
}
