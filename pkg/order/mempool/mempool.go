package mempool

import (
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
)

var _ MemPool = (*mempoolImpl)(nil)

type MemPool interface {
	// ProcessTransactions process transaction from api and other vp nodes.
	ProcessTransactions(txs []pb.Transaction, isLeader, isLocal bool) *raftproto.RequestBatch

	// GenerateBlock generate a block
	GenerateBlock() *raftproto.RequestBatch

	// CommitTransactions Remove removes the committed transactions from mempool
	CommitTransactions(state *ChainState)

	// HasPendingRequest checks if there is non-batched tx(s) in mempool pool or not
	HasPendingRequest() bool

	SetBatchSeqNo(batchSeq uint64)

	GetTimeoutTransactions(rebroadcastDuration time.Duration) [][]pb.Transaction

	SubscribeTxEvent(chan<- pb.Transactions) event.Subscription

	External
}

// External is a concurrent and safe interface, which can be called by api module directly.
type External interface {

	// GetPendingNonceByAccount will return the latest pending nonce of a given account
	GetPendingNonceByAccount(account string) uint64

	GetPendingTransactions(max int) []pb.Transaction

	GetTransaction(hash *types.Hash) pb.Transaction

	// IsPoolFull check if memPool has exceeded the limited txSize.
	IsPoolFull() bool
}

// NewMempool return the mempool instance.
func NewMempool(config *Config) (MemPool, error) {
	return newMempoolImpl(config)
}

// GenerateBlock generates a transaction batch and post it
// to outside if there are transactions in txPool.
func (mpi *mempoolImpl) GenerateBlock() *raftproto.RequestBatch {
	if !mpi.isTimed && mpi.txStore.priorityNonBatchSize == 0 {
		mpi.logger.Debug("Mempool is empty")
		return nil
	}
	batch, err := mpi.generateBlock()
	if err != nil {
		mpi.logger.Error("Generator batch failed")
		return nil
	}
	return batch
}

func (mpi *mempoolImpl) HasPendingRequest() bool {
	return mpi.txStore.priorityNonBatchSize > 0
}

func (mpi *mempoolImpl) CommitTransactions(state *ChainState) {
	gcStartTime := time.Now()
	mpi.processCommitTransactions(state)
	duration := time.Now().Sub(gcStartTime).Nanoseconds()
	mpi.logger.Debugf("GC duration %v", duration)
}

func (mpi *mempoolImpl) GetPendingNonceByAccount(account string) uint64 {
	return mpi.txStore.nonceCache.getPendingNonce(account)
}

func (mpi *mempoolImpl) IsPoolFull() bool {
	return uint64(len(mpi.txStore.txHashMap)) >= mpi.poolSize
}

func (mpi *mempoolImpl) SetBatchSeqNo(batchSeq uint64) {
	mpi.batchSeqNo = batchSeq
}
