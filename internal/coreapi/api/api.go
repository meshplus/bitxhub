package api

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
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
	HandleTransaction(tx *pb.Transaction) error
	HandleView(tx *pb.Transaction) (*pb.Receipt, error)
	GetTransaction(types.Hash) (*pb.Transaction, error)
	GetTransactionMeta(types.Hash) (*pb.TransactionMeta, error)
	GetReceipt(types.Hash) (*pb.Receipt, error)
	GetBlock(mode string, key string) (*pb.Block, error)
	GetBlocks(start uint64, end uint64) ([]*pb.Block, error)

	// AddPier
	AddPier(pid string, isUnion bool) (chan *pb.InterchainTxWrappers, error)

	// RemovePier
	RemovePier(pid string, isUnion bool)

	GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error

	GetInterchainTxWrappers(pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error

	// OrderReady
	OrderReady() bool

	FetchSignsFromOtherPeers(content string, typ pb.GetMultiSignsRequest_Type) map[string][]byte
	GetSign(content string, typ pb.GetMultiSignsRequest_Type) (string, []byte, error)
}

type NetworkAPI interface {
	PeerInfo() ([]byte, error)
}

type ChainAPI interface {
	Status() string
	Meta() (*pb.ChainMeta, error)
}

type FeedAPI interface {
	SubscribeNewBlockEvent(chan<- events.NewBlockEvent) event.Subscription
}

type AccountAPI interface {
	GetAccount(addr types.Address) *ledger.Account
}
