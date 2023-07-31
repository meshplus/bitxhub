package mempool

import (
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/sirupsen/logrus"
)

const (
	btreeDegree = 10
)

const (
	DefaultPoolSize    = 50000
	DefaultTxCacheSize = 10000
	DefaultBatchSize   = 500
	DefaultTxSetSize   = 10
	DefaultTxSetTick   = 100 * time.Millisecond
)

type GetAccountNonceFunc func(address *types.Address) uint64

type Config struct {
	ID                 uint64
	BatchSize          uint64
	PoolSize           uint64
	IsTimed            bool
	NoTxBatchTimeout   time.Duration
	RebroadcastTimeout time.Duration
	TxSliceSize        uint64
	TxSliceTimeout     time.Duration
	ChainHeight        uint64
	Logger             logrus.FieldLogger
	StoragePath        string // db for persist mem pool meta data
	GetAccountNonce    GetAccountNonceFunc
}

type txItem struct {
	account string
	tx      *types.Transaction
	local   bool
}

type ChainState struct {
	Height     uint64
	BlockHash  *types.Hash
	TxHashList []*types.Hash
}

type GetTxReq struct {
	Hash *types.Hash
	Tx   chan *types.Transaction
}
type RequestBatch struct {
	Digest    string
	TxList    []*types.Transaction
	Height    uint64
	Timestamp int64
}

type Ready struct {
	TxHashes []*types.Hash
	Height   uint64
}
