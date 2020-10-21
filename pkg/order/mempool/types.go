package mempool

import (
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/peermgr"

	cmap "github.com/orcaman/concurrent-map"
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

	DefaultBatchTick       = 500 * time.Millisecond
	DefaultTxSetTick       = 100 * time.Millisecond
	DefaultFetchTxnTimeout = 3 * time.Second
)

// batch timer reasons
const (
	StartReason1 = "first transaction set"
	StartReason2 = "finish executing a batch"

	StopReason1 = "generated a batch by batch timer"
	StopReason2 = "generated a batch by batch size"
	StopReason3 = "restart batch timer"
)

type LocalMissingTxnEvent struct {
	Height             uint64
	MissingTxnHashList map[uint64]string
	WaitC              chan bool
}

type subscribeEvent struct {
	txForwardC           chan *TxSlice
	localMissingTxnEvent chan *LocalMissingTxnEvent
	fetchTxnRequestC     chan *FetchTxnRequest
	fetchTxnResponseC    chan *FetchTxnResponse
	getBlockC            chan *constructBatchEvent
	commitTxnC           chan *raftproto.Ready
	updateLeaderC        chan uint64
	pendingNonceC        chan *getNonceRequest
}

type mempoolBatch struct {
	missingTxnHashList map[uint64]string
	txList             []*pb.Transaction
}

type constructBatchEvent struct {
	ready  *raftproto.Ready
	result chan *mempoolBatch
}

type Config struct {
	ID uint64

	BatchSize      uint64
	PoolSize       uint64
	TxSliceSize    uint64
	BatchTick      time.Duration
	FetchTimeout   time.Duration
	TxSliceTimeout time.Duration

	PeerMgr            peermgr.PeerManager
	GetTransactionFunc func(hash *types.Hash) (*pb.Transaction, error)
	ChainHeight        uint64
	Logger             logrus.FieldLogger
}

type timerManager struct {
	timeout       time.Duration      // default timeout of this timer
	isActive      cmap.ConcurrentMap // track all the timers with this timerName if it is active now
	timeoutEventC chan bool
}

type txItem struct {
	account string
	tx      *pb.Transaction
}

type getNonceRequest struct {
	account string
	waitC   chan uint64
}
