package api

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/eth-kit/ledger"
)

//go:generate mockgen -destination mock_api/mock_api.go -package mock_api -source api.go
type CoreAPI interface {
	Broker() BrokerAPI
	Network() NetworkAPI
	Chain() ChainAPI
	Feed() FeedAPI
	Account() AccountAPI
	Audit() AuditAPI
}

type BrokerAPI interface {
	HandleTransaction(tx pb.Transaction) error
	HandleView(tx pb.Transaction) (*pb.Receipt, error)
	GetTransaction(*types.Hash) (pb.Transaction, error)
	GetTransactionMeta(*types.Hash) (*pb.TransactionMeta, error)
	GetReceipt(*types.Hash) (*pb.Receipt, error)
	GetBlock(mode string, key string, fullTx bool) (*pb.Block, error)
	GetBlocks(start uint64, end uint64, fullTx bool) ([]*pb.Block, error)
	GetPendingNonceByAccount(account string) uint64
	GetPendingTransactions(max int) []pb.Transaction
	GetPoolTransaction(hash *types.Hash) pb.Transaction
	GetStateLedger() ledger.StateLedger

	// AddPier
	AddPier(pierID string) (chan *pb.InterchainTxWrappers, error)

	// RemovePier
	RemovePier(pierID string)

	GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error

	GetInterchainTxWrappers(did string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error

	// OrderReady
	OrderReady() error

	FetchSignsFromOtherPeers(req *pb.GetSignsRequest) map[string][]byte
	FetchTssInfoFromOtherPeers() []*pb.TssInfo
	SetTssNotParties(tssReq *pb.GetSignsRequest, singers []string) error

	GetSign(req *pb.GetSignsRequest, signers []string) (string, []byte, []string, error)
	GetBlockHeaders(start uint64, end uint64) ([]*pb.BlockHeader, error)
	GetQuorum() uint64
	GetTssPubkey() (string, *ecdsa.PublicKey, error)
	GetTssInfo() (*pb.TssInfo, error)
	GetPrivKey() *repo.Key
}

type NetworkAPI interface {
	PeerInfo() ([]byte, error)
	PierManager() peermgr.PierManager
	OtherPeers() map[uint64]*peer.AddrInfo
	LocalPeerID() uint64
}

type ChainAPI interface {
	Status() string
	Meta() (*pb.ChainMeta, error)
	TPS(begin, end uint64) (string, error)
}

type FeedAPI interface {
	SubscribeLogsEvent(chan<- []*pb.EvmLog) event.Subscription
	SubscribeNewTxEvent(chan<- pb.Transactions) event.Subscription
	SubscribeNewBlockEvent(chan<- events.ExecutedEvent) event.Subscription
	SubscribeTssSignRes(ch chan<- *pb.Message) event.Subscription
	SubscribeTssCulprits(ch chan<- *pb.Message) event.Subscription
	BloomStatus() (uint64, uint64)
}

type AccountAPI interface {
	GetAccount(addr *types.Address) ledger.IAccount
}

type AuditAPI interface {
	HandleAuditNodeSubscription(dataCh chan<- *pb.AuditTxInfo, auditNodeID string, blockStart uint64) error
	SubscribeAuditEvent(chan<- *pb.AuditTxInfo) event.Subscription
}
