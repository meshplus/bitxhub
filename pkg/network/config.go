package network

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/sirupsen/logrus"
)

type Config struct {
	localAddr  string
	privKey    crypto.PrivKey
	protocolID protocol.ID
	logger     logrus.FieldLogger
}

type Option func(*Config)

func WithPrivateKey(privKey crypto.PrivKey) Option {
	return func(config *Config) {
		config.privKey = privKey
	}
}

func WithLocalAddr(addr string) Option {
	return func(config *Config) {
		config.localAddr = addr
	}
}

func WithProtocolID(id protocol.ID) Option {
	return func(config *Config) {
		config.protocolID = id
	}
}

func WithLogger(logger logrus.FieldLogger) Option {
	return func(config *Config) {
		config.logger = logger
	}
}

func checkConfig(config *Config) error {
	if config.logger == nil {
		config.logger = log.NewWithModule("p2p")
	}

	if config.localAddr == "" {
		return fmt.Errorf("empty local address")
	}

	return nil
}

func generateConfig(opts ...Option) (*Config, error) {
	conf := &Config{}
	for _, opt := range opts {
		opt(conf)
	}

	if err := checkConfig(conf); err != nil {
		return nil, fmt.Errorf("create p2p: %w", err)
	}

	return conf, nil
}
