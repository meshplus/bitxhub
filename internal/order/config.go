package order

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/peermgr"
	"github.com/axiomesh/axiom/pkg/repo"
)

type Config struct {
	ID               uint64
	IsNew            bool
	Config           *repo.OrderConfig
	StoragePath      string
	StorageType      string
	OrderType        string
	PeerMgr          peermgr.PeerManager
	PrivKey          *ecdsa.PrivateKey
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

func WithConfig(cfg *repo.OrderConfig) Option {
	return func(config *Config) {
		config.Config = cfg
	}
}

func WithStoragePath(path string) Option {
	return func(config *Config) {
		config.StoragePath = path
	}
}

func WithStorageType(typ string) Option {
	return func(config *Config) {
		config.StorageType = typ
	}
}

func WithOrderType(typ string) Option {
	return func(config *Config) {
		config.OrderType = typ
	}
}

func WithPeerManager(peerMgr peermgr.PeerManager) Option {
	return func(config *Config) {
		config.PeerMgr = peerMgr
	}
}

func WithPrivKey(privKey *ecdsa.PrivateKey) Option {
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
		return errors.New("logger is nil")
	}

	return nil
}

func GenerateConfig(opts ...Option) (*Config, error) {
	config := &Config{}
	for _, opt := range opts {
		opt(config)
	}

	if err := checkConfig(config); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	return config, nil
}

type Logger struct {
	logrus.FieldLogger
}

// Trace implements rbft.Logger.
func (lg *Logger) Trace(name string, stage string, content any) {
	lg.Info(name, stage, content)
}

func (lg *Logger) Critical(v ...any) {
	lg.Info(v...)
}

func (lg *Logger) Criticalf(format string, v ...any) {
	lg.Infof(format, v...)
}

func (lg *Logger) Notice(v ...any) {
	lg.Info(v...)
}

func (lg *Logger) Noticef(format string, v ...any) {
	lg.Infof(format, v...)
}
