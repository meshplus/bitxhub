package mempool

import (
	"time"

	"github.com/google/btree"

	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
)

var _ MemPool = (*mempoolImpl)(nil)

type MemPool interface {
	// ProcessTransactions process transaction from api and other vp nodes.
	ProcessTransactions(txs []*pb.Transaction, isLeader bool) *raftproto.RequestBatch

	// GenerateBlock generate a block
	GenerateBlock() *raftproto.RequestBatch

	// Remove removes the committed transactions from mempool
	CommitTransactions(state *ChainState)

	// HasPendingRequest checks if there is non-batched tx(s) in mempool pool or not
	HasPendingRequest() bool

	SetBatchSeqNo(batchSeq uint64)

	GetTimeoutTransactions(rebroadcastDuration time.Duration) [][]*pb.Transaction

	External
}

// External is a concurrent and safe interface, which can be called by api module directly.
type External interface {

	// GetPendingNonceByAccount will return the latest pending nonce of a given account
	GetPendingNonceByAccount(account string) uint64

	// IsPoolFull check if memPool has exceeded the limited txSize.
	IsPoolFull() bool
}

// NewMempool return the mempool instance.
func NewMempool(config *Config) MemPool {
	return newMempoolImpl(config)
}

// GenerateRequestBatch generates a transaction batch and post it
// to outside if there are transactions in txPool.
func (mpi *mempoolImpl) GenerateBlock() *raftproto.RequestBatch {
	if mpi.txStore.priorityNonBatchSize == 0 {
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

func (mpi *mempoolImpl) GetTimeoutTransactions(rebroadcastDuration time.Duration) [][]*pb.Transaction {
	txList := make([]*pb.Transaction, 0, mpi.txStore.ttlIndex.index.Len())
	// all the tx whose live time is less than lowBoundTime should be rebroadcast
	mpi.logger.Debugf("------- Start gathering timeout txs, ttl index len is %d -----------", mpi.txStore.ttlIndex.index.Len())
	allItem := make([]*sortedTtlKey, 0, mpi.txStore.ttlIndex.index.Len())
	currentTime := time.Now().UnixNano()
	mpi.txStore.ttlIndex.index.Ascend(func(i btree.Item) bool {
		item := i.(*sortedTtlKey)
		timeoutTime := item.liveTime + rebroadcastDuration.Nanoseconds()
		if txMap, ok := mpi.txStore.allTxs[item.account]; ok && currentTime > timeoutTime {
			tx := txMap.items[item.nonce].tx
			txList = append(txList, tx)
			allItem = append(allItem, item)
		}
		return true
	})
	if len(txList) == 0 {
		return nil
	}
	for _, item := range allItem {
		// update the liveTime of each tx
		item.liveTime = currentTime
		mpi.txStore.ttlIndex.items[makeKey(item.account, item.nonce)] = currentTime
	}
	// shard txList into fixed size in case txList is too large to broadcast one time
	return shardTxList(txList, mpi.txSliceSize)
}

func shardTxList(txs []*pb.Transaction, batchLen uint64) [][]*pb.Transaction {
	begin := uint64(0)
	end := uint64(len(txs)) - 1
	totalLen := uint64(len(txs))

	// shape txs to fixed size in case totalLen is too large
	batchNums := totalLen / batchLen
	if totalLen%batchLen != 0 {
		batchNums++
	}
	shardedLists := make([][]*pb.Transaction, 0, batchNums)
	for i := uint64(0); i <= batchNums; i++ {
		actualLen := batchLen
		if end-begin+1 < batchLen {
			actualLen = end - begin + 1
		}
		if actualLen == 0 {
			continue
		}
		shardedList := make([]*pb.Transaction, actualLen)
		for j := uint64(0); j < batchLen && begin <= end; j++ {
			shardedList[j] = txs[begin]
			begin++
		}
		shardedLists = append(shardedLists, shardedList)
	}
	return shardedLists
}
