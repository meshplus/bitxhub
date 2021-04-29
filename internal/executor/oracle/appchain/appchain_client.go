package appchain

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type Client struct {
	EthOracle *EthLightChainOracle
}

func NewAppchainClient(path string, logger logrus.FieldLogger) (*Client, error) {
	ropstenOracle, err := NewRopstenOracle(path, false, logger)
	if err != nil {
		return nil, fmt.Errorf("create eth ropsten error:%w", err)
	}
	return &Client{EthOracle: ropstenOracle}, nil
}
