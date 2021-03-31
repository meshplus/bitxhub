package jsonrpc

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/eth"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/net"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/web3"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

// RPC namespaces and API version
const (
	Web3Namespace     = "web3"
	EthNamespace      = "eth"
	PersonalNamespace = "personal"
	NetNamespace      = "net"
	flagRPCAPI        = "rpc-api"

	apiVersion = "1.0"
)

// GetAPIs returns the list of all APIs from the Ethereum namespaces
func GetAPIs(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) ([]rpc.API, error) {
	var apis []rpc.API

	ethAPI, err := eth.NewAPI(config, api, logger)
	if err != nil {
		return nil, err
	}

	apis = append(apis,
		rpc.API{
			Namespace: EthNamespace,
			Version:   apiVersion,
			Service:   ethAPI,
			Public:    true,
		},
	)

	apis = append(apis,
		rpc.API{
			Namespace: Web3Namespace,
			Version:   apiVersion,
			Service:   web3.NewAPI(),
			Public:    true,
		},
	)

	apis = append(apis,
		rpc.API{
			Namespace: NetNamespace,
			Version:   apiVersion,
			Service:   net.NewAPI(config),
			Public:    true,
		},
	)

	return apis, nil
}
