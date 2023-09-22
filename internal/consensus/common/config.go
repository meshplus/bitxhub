package common

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/sirupsen/logrus"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/network"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

type Config struct {
	Config                                      *repo.ConsensusConfig
	Logger                                      logrus.FieldLogger
	ConsensusType                               string
	PrivKey                                     *ecdsa.PrivateKey
	SelfAccountAddress                          string
	GenesisEpochInfo                            *rbft.EpochInfo
	Network                                     network.Network
	Applied                                     uint64
	Digest                                      string
	GenesisDigest                               string
	GetCurrentEpochInfoFromEpochMgrContractFunc func() (*rbft.EpochInfo, error)
	GetEpochInfoFromEpochMgrContractFunc        func(epoch uint64) (*rbft.EpochInfo, error)
	GetChainMetaFunc                            func() *types.ChainMeta
	GetBlockFunc                                func(height uint64) (*types.Block, error)
	GetAccountBalance                           func(address *types.Address) *big.Int
	GetAccountNonce                             func(address *types.Address) uint64
}

type Option func(*Config)

func WithConfig(cfg *repo.ConsensusConfig) Option {
	return func(config *Config) {
		config.Config = cfg
	}
}

func WithGenesisEpochInfo(genesisEpochInfo *rbft.EpochInfo) Option {
	return func(config *Config) {
		config.GenesisEpochInfo = genesisEpochInfo
	}
}

func WithConsensusType(typ string) Option {
	return func(config *Config) {
		config.ConsensusType = typ
	}
}

func WithNetwork(net network.Network) Option {
	return func(config *Config) {
		config.Network = net
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

func WithGenesisDigest(digest string) Option {
	return func(config *Config) {
		config.GenesisDigest = digest
	}
}

func WithGetChainMetaFunc(f func() *types.ChainMeta) Option {
	return func(config *Config) {
		config.GetChainMetaFunc = f
	}
}

func WithGetBlockFunc(f func(height uint64) (*types.Block, error)) Option {
	return func(config *Config) {
		config.GetBlockFunc = f
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
		return nil, fmt.Errorf("create consensus: %w", err)
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
	lg.Error(v...)
}

func (lg *Logger) Criticalf(format string, v ...any) {
	lg.Errorf(format, v...)
}

func (lg *Logger) Notice(v ...any) {
	lg.Info(v...)
}

func (lg *Logger) Noticef(format string, v ...any) {
	lg.Infof(format, v...)
}
