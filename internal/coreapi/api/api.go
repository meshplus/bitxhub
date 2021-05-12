package api

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxid"
)

//go:generate mockgen -destination mock_api/mock_api.go -package mock_api -source api.go
type CoreAPI interface {
	Broker() BrokerAPI
	Network() NetworkAPI
	Chain() ChainAPI
	Feed() FeedAPI
	Account() AccountAPI
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

	// AddPier
	AddPier(did bitxid.DID, pierID string, isUnion bool) (chan *pb.InterchainTxWrappers, error)

	// RemovePier
	RemovePier(did bitxid.DID, pierID string, isUnion bool)

	GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error

	GetInterchainTxWrappers(did string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error

	// OrderReady
	OrderReady() error

	// DelVPNode delete a vp node by given id.
	DelVPNode(delID uint64) error

	FetchSignsFromOtherPeers(content string, typ pb.GetMultiSignsRequest_Type) map[string][]byte
	GetSign(content string, typ pb.GetMultiSignsRequest_Type) (string, []byte, error)
	GetBlockHeaders(start uint64, end uint64) ([]*pb.BlockHeader, error)
}

type NetworkAPI interface {
	PeerInfo() ([]byte, error)
	PierManager() peermgr.PierManager
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
	GetAccount(addr *types.Address) *ledger.Account
}
