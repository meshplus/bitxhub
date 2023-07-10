package jsonrpc

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/eth"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/eth/filters"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/net"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/web3"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

// RPC namespaces and API version
const (
	BlockChainNamespace  = "blockhain"
	BitxhubNamespace     = "bitxhub"
	TransactionNamespace = "transaction"

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
			Namespace: BlockChainNamespace,
			Version:   apiVersion,
			Service:   eth.NewBlockChainAPI(config, api, logger),
			Public:    true,
		},
	)

	apis = append(apis,
		rpc.API{
			Namespace: BitxhubNamespace,
			Version:   apiVersion,
			Service:   eth.NewBitxhubAPI(config, api, logger),
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
			Namespace: TransactionNamespace,
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
