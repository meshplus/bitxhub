package api

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
	vm "github.com/meshplus/eth-kit/evm"
	"github.com/meshplus/eth-kit/ledger"
)

//go:generate mockgen -destination mock_api/mock_api.go -package mock_api -source api.go
type CoreAPI interface {
	Broker() BrokerAPI
	Network() NetworkAPI
	Chain() ChainAPI
	Feed() FeedAPI
	Account() AccountAPI
	Gas() GasAPI
}

type BrokerAPI interface {
	HandleTransaction(tx pb.Transaction) error
	HandleView(tx pb.Transaction) (*pb.Receipt, error)
	GetTransaction(*types.Hash) (pb.Transaction, error)
	GetTransactionMeta(*types.Hash) (*pb.TransactionMeta, error)
	GetReceipt(*types.Hash) (*pb.Receipt, error)
	GetBlock(mode string, key string) (*pb.Block, error)
	GetBlocks(start uint64, end uint64) ([]*pb.Block, error)
	GetPendingNonceByAccount(account string) uint64
	GetPendingTransactions(max int) []pb.Transaction
	GetPoolTransaction(hash *types.Hash) pb.Transaction
	GetStateLedger() ledger.StateLedger
	GetEvm(mes *vm.Message, vmConfig *vm.Config) *vm.EVM
	// OrderReady
	OrderReady() error
	GetBlockHeaders(start uint64, end uint64) ([]*pb.BlockHeader, error)
}

type NetworkAPI interface {
	PeerInfo() ([]byte, error)
	OtherPeers() map[uint64]*peer.AddrInfo
}

type ChainAPI interface {
	Status() string
	Meta() (*pb.ChainMeta, error)
	TPS(begin, end uint64) (uint64, error)
}

type FeedAPI interface {
	SubscribeLogsEvent(chan<- []*pb.EvmLog) event.Subscription
	SubscribeNewTxEvent(chan<- pb.Transactions) event.Subscription
	SubscribeNewBlockEvent(chan<- events.ExecutedEvent) event.Subscription
	BloomStatus() (uint64, uint64)
}

type AccountAPI interface {
	GetAccount(addr *types.Address) ledger.IAccount
}

type GasAPI interface {
	GetGasPrice() (uint64, error)
}
