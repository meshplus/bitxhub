package order

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

type Config struct {
	ID               uint64
	IsNew            bool
	RepoRoot         string
	StoragePath      string
	OrderType        string
	PeerMgr          peermgr.OrderPeerManager
	PrivKey          crypto.PrivateKey
	Logger           logrus.FieldLogger
	Nodes            map[uint64]*types.VpInfo
	Applied          uint64
	Digest           string
	GetChainMetaFunc func() *types.ChainMeta
	GetBlockByHeight func(height uint64) (*types.Block, error)
	GetAccountNonce  func(address *types.Address) uint64
}

type Option func(*Config)

func WithID(id uint64) Option {
	return func(config *Config) {
		config.ID = id
	}
}

func WithIsNew(isNew bool) Option {
	return func(config *Config) {
		config.IsNew = isNew
	}
}

func WithRepoRoot(path string) Option {
	return func(config *Config) {
		config.RepoRoot = path
	}
}

func WithStoragePath(path string) Option {
	return func(config *Config) {
		config.StoragePath = path
	}
}

func WithOrderType(typ string) Option {
	return func(config *Config) {
		config.OrderType = typ
	}
}

func WithPeerManager(peerMgr peermgr.OrderPeerManager) Option {
	return func(config *Config) {
		config.PeerMgr = peerMgr
	}
}

func WithPrivKey(privKey crypto.PrivateKey) Option {
	return func(config *Config) {
		config.PrivKey = privKey
	}
}

func WithLogger(logger logrus.FieldLogger) Option {
	return func(config *Config) {
		config.Logger = logger
	}
}

func WithNodes(nodes map[uint64]*types.VpInfo) Option {
	return func(config *Config) {
		config.Nodes = nodes
	}
}

func WithApplied(height uint64) Option {
	return func(config *Config) {
		config.Applied = height
	}
}

func WithDigest(digest string) Option {
	return func(config *Config) {
		config.Digest = digest
	}
}

func WithGetChainMetaFunc(f func() *types.ChainMeta) Option {
	return func(config *Config) {
		config.GetChainMetaFunc = f
	}
}

func WithGetBlockByHeightFunc(f func(height uint64) (*types.Block, error)) Option {
	return func(config *Config) {
		config.GetBlockByHeight = f
	}
}

func WithGetAccountNonceFunc(f func(address *types.Address) uint64) Option {
	return func(config *Config) {
		config.GetAccountNonce = f
	}
}

func checkConfig(config *Config) error {
	if config.Logger == nil {
		return fmt.Errorf("logger is nil")
	}

	return nil
}

func GenerateConfig(opts ...Option) (*Config, error) {
	config := &Config{}
	for _, opt := range opts {
		opt(config)
	}

	if err := checkConfig(config); err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	return config, nil
}
