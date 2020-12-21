package mempool

import (
	"github.com/google/btree"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/sirupsen/logrus"
	"sync"
)

type mempoolImpl struct {
	localID    uint64
	batchSize  uint64
	batchSeqNo uint64 // track the sequence number of block
	poolSize   uint64
	logger     logrus.FieldLogger
	txStore    *transactionStore // store all transactions info
}

func newMempoolImpl(config *Config) *mempoolImpl {
	mpi := &mempoolImpl{
		localID:    config.ID,
		batchSeqNo: config.ChainHeight,
		logger:     config.Logger,
	}
	mpi.txStore = newTransactionStore()
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
	mpi.logger.Infof("MemPool batch size = %d", mpi.batchSize)
	mpi.logger.Infof("MemPool batch seqNo = %d", mpi.batchSeqNo)
	mpi.logger.Infof("MemPool pool size = %d", mpi.poolSize)
	return mpi
}

func (mpi *mempoolImpl) ProcessTransactions(txs []*pb.Transaction, isLeader bool) *raftproto.RequestBatch {
	validTxs := make(map[string][]*pb.Transaction)
	for _, tx := range txs {
		// check the sequence number of tx
		txAccount := tx.Account()
		currentSeqNo := mpi.txStore.nonceCache.getPendingNonce(txAccount)
		if tx.Nonce < currentSeqNo {
			mpi.logger.Warningf("Account %s, current sequence number is %d, required %d", txAccount, tx.Nonce, currentSeqNo+1)
			continue
		}
		// check the existence of hash of this tx
		txHash := tx.TransactionHash.String()
		if txPointer := mpi.txStore.txHashMap[txHash]; txPointer != nil {
			mpi.logger.Warningf("Tx [account: %s, nonce: %d, hash: %s] already received", txAccount, tx.Nonce, txHash)
			continue
		}
		_, ok := validTxs[txAccount]
		if !ok {
			validTxs[txAccount] = make([]*pb.Transaction, 0)
		}
		validTxs[txAccount] = append(validTxs[txAccount], tx)
	}

	// Process all the new transaction and merge any errors into the original slice
	dirtyAccounts := mpi.txStore.insertTxs(validTxs)

	// send tx to mempool store
	mpi.processDirtyAccount(dirtyAccounts)

	// generator batch by block size
	if isLeader && mpi.txStore.priorityNonBatchSize >= mpi.batchSize {
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
	for account := range dirtyAccounts {
		if list, ok := mpi.txStore.allTxs[account]; ok {
			// search for related sequential txs in allTxs
			// and add these txs into priorityIndex and parkingLotIndex
			pendingNonce := mpi.txStore.nonceCache.getPendingNonce(account)
			readyTxs, nonReadyTxs, nextDemandNonce := list.filterReady(pendingNonce)
			mpi.txStore.nonceCache.setPendingNonce(account, nextDemandNonce)

			// inset ready txs into priorityIndex.
			for _, tx := range readyTxs {
				mpi.txStore.priorityIndex.insertByOrderedQueueKey(account, tx)
			}
			mpi.txStore.priorityNonBatchSize = mpi.txStore.priorityNonBatchSize + uint64(len(readyTxs))

			// inset non-ready txs into parkingLotIndex.
			for _, tx := range nonReadyTxs {
				mpi.txStore.parkingLotIndex.insertByOrderedQueueKey(account, tx)
			}
		}
	}
}

// getBlock fetches next block of transactions for consensus,
// batchedTx are all txs sent to consensus but were not committed yet, mempool should filter out such txs.
func (mpi *mempoolImpl) generateBlock() (*raftproto.RequestBatch, error) {
	// txs has lower nonce will be observed first in priority index iterator.
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
		tx := a.(*orderedIndexKey)
		// if tx has existed in bathedTxs,
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
			ptr := orderedIndexKey{account: tx.account, nonce: tx.nonce}
			mpi.txStore.batchedTxs[ptr] = true
			result = append(result, ptr)
			if uint64(len(result)) == batchSize {
				return false
			}

			// check if we can now include some txs that were skipped before for given account
			skippedTxn := orderedIndexKey{account: tx.account, nonce: tx.nonce + 1}
			for {
				if _, ok := skippedTxs[skippedTxn]; !ok {
					break
				}
				mpi.txStore.batchedTxs[ptr] = true
				result = append(result, skippedTxn)
				if uint64(len(result)) == batchSize {
					return false
				}
				skippedTxn.nonce++
			}
		} else {
			skippedTxs[orderedIndexKey{tx.account, tx.nonce}] = true
		}
		return true
	})

	if len(result) == 0 && mpi.txStore.priorityNonBatchSize > 0 {
		mpi.logger.Error("===== NOTE!!! Leader generate a batch with 0 txs")
		mpi.txStore.priorityNonBatchSize = 0
		return nil, nil
	}

	// convert transaction pointers to real values
	txList := make([]*pb.Transaction, len(result))
	for i, v := range result {
		rawTransaction := mpi.txStore.getTxByOrderKey(v.account, v.nonce)
		txList[i] = rawTransaction
	}
	mpi.batchSeqNo++
	batchSeqNo := mpi.batchSeqNo
	batch := &raftproto.RequestBatch{
		TxList: txList,
		Height: batchSeqNo,
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
		txPointer := mpi.txStore.txHashMap[strHash]
		txPointer, ok := mpi.txStore.txHashMap[strHash]
		if !ok {
			mpi.logger.Warningf("Remove transaction %s failed, Can't find it from txHashMap", strHash)
			continue
		}
		preCommitNonce := mpi.txStore.nonceCache.getCommitNonce(txPointer.account)
		newCommitNonce := txPointer.nonce + 1
		if preCommitNonce < newCommitNonce {
			mpi.txStore.nonceCache.setCommitNonce(txPointer.account, newCommitNonce)
			// Note!!! updating pendingNonce to commitNonce for the restart node
			pendingNonce := mpi.txStore.nonceCache.getPendingNonce(txPointer.account)
			if pendingNonce < newCommitNonce {
				updateAccounts[txPointer.account] = newCommitNonce
				mpi.txStore.nonceCache.setPendingNonce(txPointer.account, newCommitNonce)
			}
		}
		delete(mpi.txStore.txHashMap, strHash)
		delete(mpi.txStore.batchedTxs, *txPointer)
		dirtyAccounts[txPointer.account] = true
	}
	// clean related txs info in cache
	for account := range dirtyAccounts {
		commitNonce := mpi.txStore.nonceCache.getCommitNonce(account)
		if list, ok := mpi.txStore.allTxs[account]; ok {
			// remove all previous seq number txs for this account.
			removedTxs := list.forward(commitNonce)
			// remove index smaller than commitNonce delete index.
			var wg sync.WaitGroup
			wg.Add(3)
			go func(ready map[string][]*pb.Transaction) {
				defer wg.Done()
				list.index.removeBySortedNonceKey(removedTxs)
			}(removedTxs)
			go func(ready map[string][]*pb.Transaction) {
				defer wg.Done()
				mpi.txStore.priorityIndex.removeByOrderedQueueKey(removedTxs)
			}(removedTxs)
			go func(ready map[string][]*pb.Transaction) {
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
	for account, pendingNonce := range updateAccounts {
		mpi.logger.Debugf("Account %s update its pendingNonce to %d by commitNonce", account, pendingNonce)
	}
	mpi.logger.Debugf("Replica %d removes batches in mempool, and now there are %d non-batched txs,"+
		"priority len: %d, parkingLot len: %d, batchedTx len: %d, txHashMap len: %d", mpi.localID, mpi.txStore.priorityNonBatchSize,
		 mpi.txStore.priorityIndex.size(), mpi.txStore.parkingLotIndex.size(), len(mpi.txStore.batchedTxs), len(mpi.txStore.txHashMap))
}
