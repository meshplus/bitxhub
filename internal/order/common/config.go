package common

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/sirupsen/logrus"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/peermgr"
	"github.com/axiomesh/axiom/pkg/repo"
)

type Config struct {
	Config                                      *repo.OrderConfig
	Logger                                      logrus.FieldLogger
	StoragePath                                 string
	StorageType                                 string
	OrderType                                   string
	PrivKey                                     *ecdsa.PrivateKey
	SelfAccountAddress                          string
	GenesisEpochInfo                            *rbft.EpochInfo
	PeerMgr                                     peermgr.PeerManager
	Applied                                     uint64
	Digest                                      string
	GetCurrentEpochInfoFromEpochMgrContractFunc func() (*rbft.EpochInfo, error)
	GetEpochInfoFromEpochMgrContractFunc        func(epoch uint64) (*rbft.EpochInfo, error)
	GetChainMetaFunc                            func() *types.ChainMeta
	GetAccountBalance                           func(address *types.Address) *big.Int
	GetAccountNonce                             func(address *types.Address) uint64
}

type Option func(*Config)

func WithConfig(cfg *repo.OrderConfig) Option {
	return func(config *Config) {
		config.Config = cfg
	}
}

func WithGenesisEpochInfo(genesisEpochInfo *rbft.EpochInfo) Option {
	return func(config *Config) {
		config.GenesisEpochInfo = genesisEpochInfo
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

func WithSelfAccountAddress(selfAccountAddress string) Option {
	return func(config *Config) {
		config.SelfAccountAddress = selfAccountAddress
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

func WithGetAccountBalanceFunc(f func(address *types.Address) *big.Int) Option {
	return func(config *Config) {
		config.GetAccountBalance = f
	}
}

func WithGetAccountNonceFunc(f func(address *types.Address) uint64) Option {
	return func(config *Config) {
		config.GetAccountNonce = f
	}
}

func WithGetEpochInfoFromEpochMgrContractFunc(f func(epoch uint64) (*rbft.EpochInfo, error)) Option {
	return func(config *Config) {
		config.GetEpochInfoFromEpochMgrContractFunc = f
	}
}

func WithGetCurrentEpochInfoFromEpochMgrContractFunc(f func() (*rbft.EpochInfo, error)) Option {
	return func(config *Config) {
		config.GetCurrentEpochInfoFromEpochMgrContractFunc = f
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
