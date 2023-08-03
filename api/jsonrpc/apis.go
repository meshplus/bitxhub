package jsonrpc

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom/api/jsonrpc/namespaces/eth"
	"github.com/axiomesh/axiom/api/jsonrpc/namespaces/eth/filters"
	"github.com/axiomesh/axiom/api/jsonrpc/namespaces/net"
	"github.com/axiomesh/axiom/api/jsonrpc/namespaces/web3"
	"github.com/axiomesh/axiom/internal/coreapi/api"
	"github.com/axiomesh/axiom/pkg/repo"
)

// RPC namespaces and API version
const (
	Web3Namespace = "web3"
	EthNamespace  = "eth"
	NetNamespace  = "net"

	apiVersion = "1.0"
)

// GetAPIs returns the list of all APIs from the Ethereum namespaces
func GetAPIs(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) ([]rpc.API, error) {
	var apis []rpc.API

	apis = append(apis,
		rpc.API{
			Namespace: EthNamespace,
			Version:   apiVersion,
			Service:   eth.NewBlockChainAPI(config, api, logger),
			Public:    true,
		},
	)

	apis = append(apis,
		rpc.API{
			Namespace: EthNamespace,
			Version:   apiVersion,
			Service:   eth.NewAxiomAPI(config, api, logger),
			Public:    true,
		},
	)

	apis = append(apis,
		rpc.API{
			Namespace: EthNamespace,
			Version:   apiVersion,
			Service:   filters.NewAPI(api, logger),
			Public:    true,
		},
	)

	apis = append(apis,
		rpc.API{
			Namespace: EthNamespace,
			Version:   apiVersion,
			Service:   eth.NewTransactionAPI(config, api, logger),
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
