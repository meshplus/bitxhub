package mempool

import (
	"math"
	"os"
	"sync"
	"time"

	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/google/btree"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type mempoolImpl struct {
	localID     uint64
	batchSize   uint64
	txSliceSize uint64
	batchSeqNo  uint64 // track the sequence number of block
	poolSize    uint64
	isTimed     bool
	logger      logrus.FieldLogger
	txStore     *transactionStore // store all transactions info
	txFeed      event.Feed
}

func newMempoolImpl(config *Config) (*mempoolImpl, error) {
	mpi := &mempoolImpl{
		localID:     config.ID,
		batchSeqNo:  config.ChainHeight,
		logger:      config.Logger,
		txSliceSize: config.TxSliceSize,
		isTimed:     config.IsTimed,
	}
	mpi.txStore = newTransactionStore(config.GetAccountNonce, config.Logger)
	if config.BatchSize == 0 {
		mpi.batchSize = DefaultBatchSize
	} else {
		mpi.batchSize = config.BatchSize
	}
	if config.PoolSize == 0 {
		mpi.poolSize = DefaultPoolSize
	} else {
		mpi.poolSize = config.PoolSize
	}
	if config.TxSliceSize == 0 {
		mpi.txSliceSize = DefaultTxSetSize
	} else {
		mpi.txSliceSize = config.TxSliceSize
	}
	mpi.logger.Infof("MemPool batch size = %d", mpi.batchSize)
	mpi.logger.Infof("MemPool tx slice size = %d", mpi.batchSize)
	mpi.logger.Infof("MemPool batch seqNo = %d", mpi.batchSeqNo)
	mpi.logger.Infof("MemPool pool size = %d", mpi.poolSize)
	return mpi, nil
}

func (mpi *mempoolImpl) ProcessTransactions(txs []pb.Transaction, isLeader, isLocal bool) *raftproto.RequestBatch {
	validTxs := make(map[string][]pb.Transaction)
	validTxList := make([]pb.Transaction, 0)

	for _, tx := range txs {
		// check the sequence number of tx
		txAccount := tx.GetFrom().String()
		currentSeqNo := mpi.txStore.nonceCache.getPendingNonce(txAccount)
		if tx.GetNonce() < currentSeqNo {
			mpi.logger.Warningf("Account %s, current sequence number is %d, required %d", txAccount, tx.GetNonce(), currentSeqNo)
			continue
		}
		// check the existence of hash of this tx
		txHash := tx.GetHash().String()
		if txPointer := mpi.txStore.txHashMap[txHash]; txPointer != nil {
			mpi.logger.Warningf("Tx [account: %s, nonce: %d, hash: %s] already received", txAccount, tx.GetNonce(), txHash)
			continue
		}
		_, ok := validTxs[txAccount]
		if !ok {
			validTxs[txAccount] = make([]pb.Transaction, 0)
		}
		validTxs[txAccount] = append(validTxs[txAccount], tx)
		validTxList = append(validTxList, tx)
	}

	mpi.postTxsEvent(validTxList)

	// Process all the new transaction and merge any errors into the original slice
	dirtyAccounts := mpi.txStore.insertTxs(validTxs, isLocal)

	// send tx to mempool store
	mpi.processDirtyAccount(dirtyAccounts)

	// if no timedBlock, generator batch by block size
	if isLeader && mpi.txStore.priorityNonBatchSize >= mpi.batchSize && !mpi.isTimed {
		batch, err := mpi.generateBlock()
		if err != nil {
			mpi.logger.Errorf("Generator batch failed")
			return nil
		}
		return batch
	}
	return nil
}

func (mpi *mempoolImpl) processDirtyAccount(dirtyAccounts map[string]bool) {
	// updateAccounts track the biggest pending nonces to update
	updateAccounts := make(map[string]uint64)
	for account := range dirtyAccounts {
		if list, ok := mpi.txStore.allTxs[account]; ok {
			// search for related sequential txs in allTxs
			// and add these txs into priorityIndex and parkingLotIndex
			pendingNonce := mpi.txStore.nonceCache.getPendingNonce(account)
			readyTxs, nonReadyTxs, nextDemandNonce := list.filterReady(pendingNonce)
			updateAccounts[account] = nextDemandNonce
			// insert ready txs into priorityIndex.
			for _, tx := range readyTxs {
				if !mpi.txStore.priorityIndex.data.Has(makeTimeoutKey(account, tx)) {
					mpi.txStore.priorityIndex.insertByTimeoutKey(account, tx)
				}
			}
			mpi.txStore.updateEarliestTimestamp()
			mpi.txStore.priorityNonBatchSize = mpi.txStore.priorityNonBatchSize + uint64(len(readyTxs))

			// inset non-ready txs into parkingLotIndex.
			for _, tx := range nonReadyTxs {
				mpi.txStore.parkingLotIndex.insertByOrderedQueueKey(account, tx)
			}
		}
	}
	// update pending nonce in batch mode for performance consideration
	mpi.txStore.nonceCache.updatePendingNonce(updateAccounts)
}

// getBlock fetches next block of transactions for consensus,
// batchedTx are all txs sent to consensus but were not committed yet, mempool should filter out such txs.
func (mpi *mempoolImpl) generateBlock() (*raftproto.RequestBatch, error) {
	// tx which has lower timestamp will be observed first in priority index iterator.
	// and if first seen tx's nonce isn't the required nonce for the account,
	// it will be stored in skip DS first.
	mpi.logger.Debugf("Length of non-batched transactions: %d", mpi.txStore.priorityNonBatchSize)
	var batchSize uint64
	if poolLen := mpi.txStore.priorityNonBatchSize; poolLen > mpi.batchSize {
		batchSize = mpi.batchSize
	} else {
		batchSize = mpi.txStore.priorityNonBatchSize
	}

	skippedTxs := make(map[orderedIndexKey]bool)
	result := make([]orderedIndexKey, 0, mpi.batchSize)
	mpi.txStore.priorityIndex.data.Ascend(func(a btree.Item) bool {
		tx := a.(*orderedTimeoutKey)
		// if tx has existed in bathedTxs, ignore this tx
		if _, ok := mpi.txStore.batchedTxs[orderedIndexKey{tx.account, tx.nonce}]; ok {
			return true
		}
		txSeq := tx.nonce
		commitNonce := mpi.txStore.nonceCache.getCommitNonce(tx.account)
		var seenPrevious bool
		if txSeq >= 1 {
			_, seenPrevious = mpi.txStore.batchedTxs[orderedIndexKey{account: tx.account, nonce: txSeq - 1}]
		}
		// include transaction if it's "next" for given account or
		// we've already sent its ancestor to Consensus
		if seenPrevious || (txSeq == commitNonce) {
			ptr := orderedIndexKey{account: tx.account, nonce: txSeq}
			mpi.txStore.batchedTxs[ptr] = true
			result = append(result, ptr)
			if uint64(len(result)) == batchSize {
				return false
			}

			// check if we can now include some txs that were skipped before for given account
			skippedTxn := orderedIndexKey{account: tx.account, nonce: txSeq + 1}
			for {
				if _, ok := skippedTxs[skippedTxn]; !ok {
					break
				}
				mpi.txStore.batchedTxs[skippedTxn] = true
				result = append(result, skippedTxn)
				if uint64(len(result)) == batchSize {
					return false
				}
				skippedTxn.nonce++
			}
		} else {
			skippedTxs[orderedIndexKey{tx.account, txSeq}] = true
		}
		return true
	})

	if !mpi.isTimed && len(result) == 0 && mpi.txStore.priorityNonBatchSize > 0 {
		mpi.logger.Error("===== NOTE!!! Leader generate a batch with 0 txs")
		mpi.txStore.priorityNonBatchSize = 0
		return nil, nil
	}

	// convert transaction pointers to real values
	txList := make([]pb.Transaction, len(result))
	for i, v := range result {
		rawTransaction := mpi.txStore.getTxByOrderKey(v.account, v.nonce)
		txList[i] = rawTransaction
	}
	mpi.batchSeqNo++
	batchSeqNo := mpi.batchSeqNo
	batch := &raftproto.RequestBatch{
		TxList:    &pb.Transactions{Transactions: txList},
		Height:    batchSeqNo,
		Timestamp: time.Now().UnixNano(),
	}
	if mpi.txStore.priorityNonBatchSize >= uint64(len(txList)) {
		mpi.txStore.priorityNonBatchSize = mpi.txStore.priorityNonBatchSize - uint64(len(txList))
	}
	mpi.logger.Debugf("Leader generate a batch with %d txs, which height is %d, and now there are %d pending txs.", len(txList), batchSeqNo, mpi.txStore.priorityNonBatchSize)
	return batch, nil
}

// processCommitTransactions removes the transactions in ready.
func (mpi *mempoolImpl) processCommitTransactions(state *ChainState) {
	dirtyAccounts := make(map[string]bool)
	// update current cached commit nonce for account
	updateAccounts := make(map[string]uint64)
	// update current cached commit nonce for account
	for _, txHash := range state.TxHashList {
		strHash := txHash.String()
		txPointer, ok := mpi.txStore.txHashMap[strHash]
		if !ok {
			mpi.logger.Warningf("Remove transaction %s failed, Can't find it from txHashMap", strHash)
			continue
		}
		preCommitNonce := mpi.txStore.nonceCache.getCommitNonce(txPointer.account)
		newCommitNonce := txPointer.nonce + 1
		if updateAccounts[txPointer.account] < newCommitNonce && preCommitNonce < newCommitNonce {
			updateAccounts[txPointer.account] = newCommitNonce
		}
		delete(mpi.txStore.txHashMap, strHash)
		delete(mpi.txStore.batchedTxs, *txPointer)
		dirtyAccounts[txPointer.account] = true
	}
	// batch commit new Nonce
	mpi.txStore.nonceCache.updateCommittedNonce(updateAccounts)

	// clean related txs info in cache
	for account := range dirtyAccounts {
		commitNonce := mpi.txStore.nonceCache.getCommitNonce(account)
		if list, ok := mpi.txStore.allTxs[account]; ok {
			// remove all previous seq number txs for this account.
			removedTxs := list.forward(commitNonce)
			// remove index smaller than commitNonce delete index.
			var wg sync.WaitGroup
			wg.Add(4)
			go func(ready map[string][]pb.Transaction) {
				defer wg.Done()
				list.index.removeBySortedNonceKey(removedTxs)
			}(removedTxs)
			go func(ready map[string][]pb.Transaction) {
				defer wg.Done()
				mpi.txStore.priorityIndex.removeByTimeoutKey(removedTxs)
			}(removedTxs)
			go func(ready map[string][]pb.Transaction) {
				defer wg.Done()
				mpi.txStore.ttlIndex.removeByTtlKey(removedTxs)
				mpi.txStore.updateEarliestTimestamp()
			}(removedTxs)
			go func(ready map[string][]pb.Transaction) {
				defer wg.Done()
				mpi.txStore.parkingLotIndex.removeByOrderedQueueKey(removedTxs)
			}(removedTxs)
			wg.Wait()
		}
	}
	readyNum := uint64(mpi.txStore.priorityIndex.size())
	// set priorityNonBatchSize to min(nonBatchedTxs, readyNum),
	if mpi.txStore.priorityNonBatchSize > readyNum {
		mpi.txStore.priorityNonBatchSize = readyNum
	}
	mpi.logger.Debugf("Replica %d removes batches in mempool, and now there are %d non-batched txs,"+
		"priority len: %d, parkingLot len: %d, batchedTx len: %d, txHashMap len: %d", mpi.localID, mpi.txStore.priorityNonBatchSize,
		mpi.txStore.priorityIndex.size(), mpi.txStore.parkingLotIndex.size(), len(mpi.txStore.batchedTxs), len(mpi.txStore.txHashMap))
}

func (mpi *mempoolImpl) GetTimeoutTransactions(rebroadcastDuration time.Duration) [][]pb.Transaction {
	// all the tx whose live time is less than lowBoundTime should be rebroadcast
	mpi.logger.Debugf("Start gathering timeout txs, ttl index len is %d", mpi.txStore.ttlIndex.index.Len())
	currentTime := time.Now().UnixNano()
	if currentTime < mpi.txStore.earliestTimestamp+rebroadcastDuration.Nanoseconds() {
		// if the latest incoming tx has not exceeded the timeout limit, then none will be timeout
		return [][]pb.Transaction{}
	}

	timeoutItems := make([]*orderedTimeoutKey, 0)
	mpi.txStore.ttlIndex.index.Ascend(func(i btree.Item) bool {
		item := i.(*orderedTimeoutKey)
		if item.timestamp > math.MaxInt64 {
			// TODO(tyx): if this tx has rebroadcast many times and exceeded a final limit,
			// it is expired and will be removed from mempool
			return true
		}
		// if this tx has not exceeded the rebroadcast duration, break iteration
		timeoutTime := item.timestamp + rebroadcastDuration.Nanoseconds()
		_, ok := mpi.txStore.allTxs[item.account]
		if !ok || currentTime < timeoutTime {
			return false
		}
		timeoutItems = append(timeoutItems, item)
		return true
	})
	for _, item := range timeoutItems {
		// update the liveTime of timeout txs
		mpi.txStore.ttlIndex.updateByTtlKey(item, currentTime)
	}
	// shard txList into fixed size in case txList is too large to broadcast one time
	return mpi.shardTxList(timeoutItems, mpi.txSliceSize)
}

func (mpi *mempoolImpl) GetPendingTransactions(max int) []pb.Transaction {
	if max < 0 {
		max = int(mpi.txStore.priorityNonBatchSize)
	}

	// TODO：Not implemented yet

	return nil
}

func (mpi *mempoolImpl) GetTransaction(hash *types.Hash) pb.Transaction {
	key, ok := mpi.txStore.txHashMap[hash.String()]
	if !ok {
		return nil
	}

	txMap, ok := mpi.txStore.allTxs[key.account]
	if !ok {
		return nil
	}

	item, ok := txMap.items[key.nonce]
	if !ok {
		return nil
	}

	return item.tx
}

func (mpi *mempoolImpl) shardTxList(timeoutItems []*orderedTimeoutKey, batchLen uint64) [][]pb.Transaction {
	begin := uint64(0)
	end := uint64(len(timeoutItems)) - 1
	totalLen := uint64(len(timeoutItems))

	// shape timeout txs to batch size in case totalLen is too large
	batchNums := totalLen / batchLen
	if totalLen%batchLen != 0 {
		batchNums++
	}
	shardedLists := make([][]pb.Transaction, 0, batchNums)
	for i := uint64(0); i < batchNums; i++ {
		actualLen := batchLen
		if end-begin+1 < batchLen {
			actualLen = end - begin + 1
		}

		shardedList := make([]pb.Transaction, actualLen)
		for j := uint64(0); j < batchLen && begin <= end; j++ {
			txMap, _ := mpi.txStore.allTxs[timeoutItems[begin].account]
			shardedList[j] = txMap.items[timeoutItems[begin].nonce].tx
			begin++
		}
		shardedLists = append(shardedLists, shardedList)
	}
	return shardedLists
}

func loadOrCreateStorage(memPoolDir string) (storage.Storage, error) {
	if !fileutil.Exist(memPoolDir) {
		if err := os.MkdirAll(memPoolDir, os.ModePerm); err != nil {
			return nil, errors.Errorf("failed to mkdir '%s' for mem pool: %s", memPoolDir, err)
		}
	}
	return leveldb.New(memPoolDir)
}

func (mpi *mempoolImpl) SubscribeTxEvent(ch chan<- pb.Transactions) event.Subscription {
	return mpi.txFeed.Subscribe(ch)
}

func (mpi *mempoolImpl) postTxsEvent(txList []pb.Transaction) {
	go mpi.txFeed.Send(pb.Transactions{Transactions: txList})
}
