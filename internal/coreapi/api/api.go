package api

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/pkg/model/events"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger"
)

//go:generate mockgen -destination mock_api/mock_api.go -package mock_api -source api.go -typed
type CoreAPI interface {
	Broker() BrokerAPI
	Chain() ChainAPI
	Feed() FeedAPI
	Account() AccountAPI
	Gas() GasAPI
}

type BrokerAPI interface {
	HandleTransaction(tx *types.Transaction) error
	HandleView(tx *types.Transaction) (*types.Receipt, error)
	GetTransaction(*types.Hash) (*types.Transaction, error)
	GetTransactionMeta(*types.Hash) (*types.TransactionMeta, error)
	GetReceipt(*types.Hash) (*types.Receipt, error)
	GetBlock(mode string, key string) (*types.Block, error)
	GetBlocks(start uint64, end uint64) ([]*types.Block, error)
	GetPendingTxCountByAccount(account string) uint64
	GetTotalPendingTxCount() uint64
	GetPoolTransaction(hash *types.Hash) *types.Transaction
	GetStateLedger() ledger.StateLedger
	GetEvm(mes *vm.Message, vmConfig *vm.Config) (*vm.EVM, error)
	GetSystemContract(addr *ethcommon.Address) (common.SystemContract, bool)

	// OrderReady
	OrderReady() error

	GetBlockHeaders(start uint64, end uint64) ([]*types.BlockHeader, error)
}

type NetworkAPI interface {
	PeerInfo() ([]byte, error)
}

type ChainAPI interface {
	Status() string
	Meta() (*types.ChainMeta, error)
	TPS(begin, end uint64) (uint64, error)
}

type FeedAPI interface {
	SubscribeLogsEvent(chan<- []*types.EvmLog) event.Subscription
	SubscribeNewTxEvent(chan<- []*types.Transaction) event.Subscription
	SubscribeNewBlockEvent(chan<- events.ExecutedEvent) event.Subscription
	BloomStatus() (uint64, uint64)
}

type AccountAPI interface {
	GetAccount(addr *types.Address) ledger.IAccount
}

type GasAPI interface {
	GetGasPrice() (uint64, error)
	GetCurrentGasPrice(blockHeight uint64) (uint64, error)
}
