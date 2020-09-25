package mempool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/loggers"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/storage"

	"github.com/google/btree"
	"github.com/sirupsen/logrus"
)

type mempoolImpl struct {
	localID    uint64
	leader     uint64 // leader node id
	batchSize  uint64
	batchSeqNo uint64 // track the sequence number of block
	logger     logrus.FieldLogger
	batchC     chan *raftproto.Ready
	close      chan bool

	txStore       *transactionStore // store all transactions info
	txCache       *TxCache          // cache the transactions received from api
	subscribe     *subscribeEvent
	storage       storage.Storage
	peerMgr       peermgr.PeerManager //network manager
	batchTimerMgr *timerManager
	ledgerHelper  func(hash types.Hash) (*pb.Transaction, error)
}

func newMempoolImpl(config *Config, storage storage.Storage, batchC chan *raftproto.Ready) *mempoolImpl {
	mpi := &mempoolImpl{
		localID:      config.ID,
		peerMgr:      config.PeerMgr,
		batchSeqNo:   config.ChainHeight,
		ledgerHelper: config.GetTransactionFunc,
		logger:       loggers.Logger(loggers.Order),
		batchC:       batchC,
		storage:      storage,
	}
	mpi.txStore = newTransactionStore()
	mpi.txCache = newTxCache(config.TxSliceTimeout)
	mpi.subscribe = newSubscribe()
	if config.BatchSize == 0 {
		mpi.batchSize = DefaultBatchSize
	} else {
		mpi.batchSize = config.BatchSize
	}
	var batchTick time.Duration
	if config.BatchTick == 0 {
		batchTick = DefaultBatchTick
	} else {
		batchTick = config.BatchTick
	}
	mpi.batchTimerMgr = newTimer(batchTick)
	return mpi
}

func (mpi *mempoolImpl) listenEvent() {
	waitC := make(chan bool)
	for {
		select {
		case <-mpi.close:
			mpi.logger.Info("----- Exit listen loop -----")
			return

		case newLeader := <-mpi.subscribe.updateLeaderC:
			if newLeader == mpi.localID {
				mpi.logger.Info("----- Become the leader node -----")
			}
			mpi.leader = newLeader

		case txSet := <-mpi.txCache.txSetC:
			// 1. send transactions to other peer
			data, err := txSet.Marshal()
			if err != nil {
				mpi.logger.Errorf("Marshal failed, err: %s", err.Error())
				return
			}
			pbMsg := mpi.msgToConsensusPbMsg(data, raftproto.RaftMessage_BROADCAST_TX)
			mpi.broadcast(pbMsg)

			// 2. process transactions
			if err := mpi.processTransactions(txSet.TxList); err != nil {
				mpi.logger.Errorf("Process transactions failed, err: %s", err.Error())
			}

		case txSlice := <-mpi.subscribe.txForwardC:
			if err := mpi.processTransactions(txSlice.TxList); err != nil {
				mpi.logger.Errorf("Process transactions failed, err: %s", err.Error())
			}

		case res := <-mpi.subscribe.getBlockC:
			result := mpi.getBlock(res.ready)
			res.result <- result

		case <-mpi.batchTimerMgr.timeoutEventC:
			if mpi.isBatchTimerActive() {
				mpi.stopBatchTimer(StopReason1)
				mpi.logger.Debug("Batch timer expired, try to create a batch")
				if mpi.txStore.priorityNonBatchSize > 0 {
					ready, err := mpi.generateBlock(true)
					if err != nil {
						mpi.logger.Errorf("Generator batch failed")
						continue
					}
					mpi.batchC <- ready
				} else {
					mpi.logger.Debug("The length of priorityIndex is 0, ")
				}
			}

		case commitReady := <-mpi.subscribe.commitTxnC:
			gcStartTime := time.Now()
			mpi.processCommitTransactions(commitReady)
			duration := time.Now().Sub(gcStartTime).Nanoseconds()
			mpi.logger.Debugf("GC duration %v", duration)

		case lostTxnEvent := <-mpi.subscribe.localMissingTxnEvent:
			if err := mpi.sendFetchTxnRequest(lostTxnEvent.Height, lostTxnEvent.MissingTxnHashList); err != nil {
				mpi.logger.Errorf("Process fetch txn failed, err: %s", err.Error())
				lostTxnEvent.WaitC <- false
			} else {
				mpi.logger.Debug("Process fetch txn success")
				waitC = lostTxnEvent.WaitC
			}

		case fetchRequest := <-mpi.subscribe.fetchTxnRequestC:
			if err := mpi.processFetchTxnRequest(fetchRequest); err != nil {
				mpi.logger.Error("Process fetchTxnRequest failed")
			}

		case fetchRes := <-mpi.subscribe.fetchTxnResponseC:
			if err := mpi.processFetchTxnResponse(fetchRes); err != nil {
				waitC <- false
				continue
			}
			waitC <- true
		}
	}
}

func (mpi *mempoolImpl) processTransactions(txs []*pb.Transaction) error {
	validTxs := make(map[string][]*pb.Transaction)
	for _, tx := range txs {
		// check if this tx signature is valid first
		ok, _ := asym.Verify(crypto.Secp256k1, tx.Signature, tx.SignHash().Bytes(), tx.From)

		if !ok {
			return fmt.Errorf("invalid signature")
		}
		// check the sequence number of tx
		// TODO refactor Transaction
		txAccount, err := getAccount(tx)
		if err != nil {
			return fmt.Errorf("get tx account failed, err: %s", err.Error())
		}
		currentSeqNo := mpi.txStore.nonceCache.getPendingNonce(txAccount)
		if tx.Nonce < currentSeqNo {
			mpi.logger.Warningf("account %s current sequence number is %d, required %d", txAccount, tx.Nonce, currentSeqNo+1)
			return nil
		}
		// check the existence of hash of this tx
		txHash := tx.TransactionHash.Hex()
		if txPointer := mpi.txStore.txHashMap[txHash]; txPointer != nil {
			mpi.logger.Warningf("Tx %s already received", txHash)
			continue
		}
		_, ok = validTxs[txAccount]
		if !ok {
			validTxs[txAccount] = make([]*pb.Transaction, 0)
		}
		validTxs[txAccount] = append(validTxs[txAccount], tx)
	}

	// Process all the new transaction and merge any errors into the original slice
	dirtyAccounts := mpi.txStore.InsertTxs(validTxs)

	// send tx to mempool store
	mpi.processDirtyAccount(dirtyAccounts)

	if mpi.isLeader() {
		// start batch timer when this node receives the first transaction set of a batch
		if !mpi.isBatchTimerActive() {
			mpi.startBatchTimer(StartReason1)
		}

		// generator batch by block size
		if mpi.txStore.priorityNonBatchSize >= mpi.batchSize {
			ready, err := mpi.generateBlock(false)
			if err != nil {
				return errors.New("generator batch fai")
			}
			// stop batch timer
			mpi.stopBatchTimer(StopReason2)
			mpi.batchC <- ready
		}
	}
	return nil
}

func (txStore *transactionStore) InsertTxs(txs map[string][]*pb.Transaction) map[string]bool {
	dirtyAccounts := make(map[string]bool)
	for account, list := range txs {
		for _, tx := range list {
			txHash := tx.TransactionHash.Hex()
			txPointer := &orderedIndexKey{
				account: account,
				nonce:   tx.Nonce,
			}
			txStore.txHashMap[txHash] = txPointer
			list, ok := txStore.allTxs[account]
			if !ok {
				// if this is new account to send tx, create a new txSortedMap
				txStore.allTxs[account] = newTxSortedMap()
			}
			list = txStore.allTxs[account]
			txItem := &txItem{
				account: account,
				tx:      tx,
			}
			list.items[tx.Nonce] = txItem
			list.index.insert(tx)
			atomic.AddInt32(&txStore.poolSize, 1)
		}
		dirtyAccounts[account] = true
	}
	return dirtyAccounts
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
func (mpi *mempoolImpl) generateBlock(isTimeout bool) (*raftproto.Ready, error) {
	result := make([]orderedIndexKey, 0, mpi.batchSize)

	// txs has lower nonce will be observed first in priority index iterator.
	mpi.logger.Infof("Length of priority index: %v", mpi.txStore.priorityIndex.data.Len())
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
			// batched by batch size or timeout
			condition1 := uint64(len(result)) == mpi.batchSize
			condition2 := isTimeout && uint64(len(result)) == mpi.txStore.priorityNonBatchSize
			if condition1 || condition2 {
				return false
			}
		}
		return true
	})

	// convert transaction pointers to real values
	hashList := make([]types.Hash, len(result))
	txList := make([]*pb.Transaction, len(result))
	for i, v := range result {
		rawTransaction := mpi.txStore.getTxByOrderKey(v.account, v.nonce)
		hashList[i] = rawTransaction.TransactionHash
		txList[i] = rawTransaction
	}
	mpi.increaseBatchSeqNo()
	batchSeqNo := mpi.getBatchSeqNo()
	ready := &raftproto.Ready{
		TxHashes: hashList,
		Height:   batchSeqNo,
	}
	// store the batch to cache
	if _, ok := mpi.txStore.batchedCache[batchSeqNo]; ok {
		mpi.logger.Errorf("Generate block with height %d, but there is already block at this height", batchSeqNo)
		return nil, errors.New("wrong block height ")
	}
	// store the batch to cache
	mpi.txStore.batchedCache[batchSeqNo] = txList
	// store the batch to db
	mpi.batchStore(txList)
	mpi.txStore.priorityNonBatchSize = mpi.txStore.priorityNonBatchSize - uint64(len(hashList))
	mpi.logger.Infof("Generated block %d with %d txs", batchSeqNo, len(txList))
	return ready, nil
}

func (mpi *mempoolImpl) getBlock(ready *raftproto.Ready) *mempoolBatch {
	res := &mempoolBatch{}
	// leader get the block directly from batchedCache
	if mpi.isLeader() {
		if txList, ok := mpi.txStore.batchedCache[ready.Height]; !ok {
			mpi.logger.Warningf("Leader get block failed, can't find block %d from batchedCache", ready.Height)
			missingTxnHashList := make(map[uint64]string)
			for i, txHash := range ready.TxHashes {
				missingTxnHashList[uint64(i)] = txHash.Hex()
			}
			res.missingTxnHashList = missingTxnHashList
		} else {
			// TODO (YH): check tx hash and length
			res.txList = txList
		}
		return res
	}
	// follower construct the same batch by given ready.
	return mpi.constructSameBatch(ready)
}

func (mpi *mempoolImpl) constructSameBatch(ready *raftproto.Ready) *mempoolBatch {
	res := &mempoolBatch{}
	if txList, ok := mpi.txStore.batchedCache[ready.Height]; ok {
		mpi.logger.Warningf("Batch %d already exists in batchedCache", ready.Height)
		// TODO (YH): check tx hash and length
		res.txList = txList
		return res
	}
	missingTxList := make(map[uint64]string)
	txList := make([]*pb.Transaction, 0)
	for index, txHash := range ready.TxHashes {
		var (
			txPointer *orderedIndexKey
			txMap     *txSortedMap
			txItem    *txItem
			ok        bool
		)
		if txPointer, _ = mpi.txStore.txHashMap[txHash.Hex()]; txPointer == nil {
			missingTxList[uint64(index)] = txHash.Hex()
			continue
		}
		if txMap, ok = mpi.txStore.allTxs[txPointer.account]; !ok {
			mpi.logger.Warningf("Transaction %s exist in txHashMap but not in allTxs", txHash.Hex())
			missingTxList[uint64(index)] = txHash.Hex()
			continue
		}
		if txItem, ok = txMap.items[txPointer.nonce]; !ok {
			mpi.logger.Warningf("Transaction %s exist in txHashMap but not in allTxs", txHash.Hex())
			missingTxList[uint64(index)] = txHash.Hex()
			continue
		}
		txList = append(txList, txItem.tx)
		mpi.txStore.batchedTxs[*txPointer] = true
	}
	res.missingTxnHashList = missingTxList
	res.txList = txList
	// store the batch to cache
	mpi.txStore.batchedCache[ready.Height] = txList
	// store the batch to db
	mpi.batchStore(txList)
	return res
}

func (mpi *mempoolImpl) processCommitTransactions(ready *raftproto.Ready) {
	dirtyAccounts := make(map[string]bool)
	// update current cached commit nonce for account
	for _, txHash := range ready.TxHashes {
		txHashStr := txHash.Hex()
		txPointer := mpi.txStore.txHashMap[txHashStr]
		preCommitNonce := mpi.txStore.nonceCache.getCommitNonce(txPointer.account)
		newCommitNonce := txPointer.nonce + 1
		if preCommitNonce < newCommitNonce {
			mpi.txStore.nonceCache.setCommitNonce(txPointer.account, newCommitNonce)
		}
		delete(mpi.txStore.txHashMap, txHashStr)
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
				list.index.remove(removedTxs)
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
			delta := int32(len(removedTxs))
			atomic.AddInt32(&mpi.txStore.poolSize, -delta)
		}
	}
	mpi.batchDelete(ready.TxHashes)
	delete(mpi.txStore.batchedCache, ready.Height)
	// restart batch timer for remain txs.
	if mpi.isLeader() {
		mpi.startBatchTimer(StartReason2)
	}
	mpi.logger.Debugf("Replica removes batch %d in mempool, and now there are %d batches, "+
		"priority len: %d, parkingLot len: %d", ready.Height, len(mpi.txStore.batchedCache),
		mpi.txStore.priorityIndex.size(), mpi.txStore.parkingLotIndex.size())
}

// sendFetchTxnRequest sends fetching missing transactions request to leader node.
func (mpi *mempoolImpl) sendFetchTxnRequest(height uint64, lostTxnHashList map[uint64]string) error {
	filterFetchTxHashList := &FetchTxnRequest{
		ReplicaId:       mpi.localID,
		Height:          height,
		MissingTxHashes: lostTxnHashList,
	}
	missingHashListBytes, err := filterFetchTxHashList.Marshal()
	if err != nil {
		mpi.logger.Error("Marshal MissingHashList fail")
		return err
	}
	pbMsg := mpi.msgToConsensusPbMsg(missingHashListBytes, raftproto.RaftMessage_GET_TX)
	mpi.logger.Debugf("Send fetch transactions request to replica %d", mpi.leader)
	mpi.unicast(mpi.leader, pbMsg)
	mpi.txStore.missingBatch[height] = lostTxnHashList
	return nil
}

// processFetchTxnRequest processes fetch request...
func (mpi *mempoolImpl) processFetchTxnRequest(fetchTxnRequest *FetchTxnRequest) error {
	txList := make(map[uint64]*pb.Transaction, len(fetchTxnRequest.MissingTxHashes))
	var err error
	if txList, err = mpi.loadTxnFromCache(fetchTxnRequest); err != nil {
		if txList, err = mpi.loadTxnFromStorage(fetchTxnRequest); err != nil {
			if txList, err = mpi.loadTxnFromLedger(fetchTxnRequest); err != nil {
				mpi.logger.Error("Process fetch txn request failed.")
				return err
			}
		}
	}
	fetchTxnResponse := &FetchTxnResponse{
		ReplicaId:      mpi.localID,
		Height:         fetchTxnRequest.Height,
		MissingTxnList: txList,
	}
	resBytes, err := fetchTxnResponse.Marshal()
	if err != nil {
		return err
	}
	pbMsg := mpi.msgToConsensusPbMsg(resBytes, raftproto.RaftMessage_GET_TX_ACK)
	mpi.logger.Debugf("Send fetch transactions response to replica %d", fetchTxnRequest.ReplicaId)
	mpi.unicast(fetchTxnRequest.ReplicaId, pbMsg)
	return nil
}

func (mpi *mempoolImpl) loadTxnFromCache(fetchTxnRequest *FetchTxnRequest) (map[uint64]*pb.Transaction, error) {
	missingHashList := fetchTxnRequest.MissingTxHashes
	targetHeight := fetchTxnRequest.Height
	for _, txHash := range missingHashList {
		if txPointer, _ := mpi.txStore.txHashMap[txHash]; txPointer == nil {
			return nil, fmt.Errorf("transaction %s dones't exist in txHashMap", txHash)
		}
	}
	var targetBatch []*pb.Transaction
	var ok bool
	if targetBatch, ok = mpi.txStore.batchedCache[targetHeight]; !ok {
		return nil, fmt.Errorf("batch %d dones't exist in batchedCache", targetHeight)
	}
	targetBatchLen := uint64(len(targetBatch))
	txList := make(map[uint64]*pb.Transaction, len(missingHashList))
	for index, txHash := range missingHashList {
		if index > targetBatchLen || targetBatch[index].TransactionHash.Hex() != txHash {
			return nil, fmt.Errorf("find invaild transaction, index: %d, targetHash: %s", index, txHash)
		}
		txList[index] = targetBatch[index]
	}
	return txList, nil
}

// TODO (YH): restore txn from wal
func (mpi *mempoolImpl) loadTxnFromStorage(fetchTxnRequest *FetchTxnRequest) (map[uint64]*pb.Transaction, error) {
	missingHashList := fetchTxnRequest.MissingTxHashes
	txList := make(map[uint64]*pb.Transaction)
	for index, txHash := range missingHashList {
		var (
			rawHash types.Hash
			err     error
		)
		if rawHash, err = hex2Hash(txHash); err != nil {
			return nil, err
		}
		if tx, ok := mpi.load(rawHash); !ok {
			return nil, errors.New("can't load tx from storage")
		} else {
			txList[index] = tx
		}
	}
	return txList, nil
}

func (mpi *mempoolImpl) loadTxnFromLedger(fetchTxnRequest *FetchTxnRequest) (map[uint64]*pb.Transaction, error) {
	missingHashList := fetchTxnRequest.MissingTxHashes
	txList := make(map[uint64]*pb.Transaction)
	for index, txHash := range missingHashList {
		var (
			tx      *pb.Transaction
			rawHash types.Hash
			err     error
		)
		if rawHash, err = hex2Hash(txHash); err != nil {
			return nil, err
		}
		if tx, err = mpi.ledgerHelper(rawHash); err != nil {
			return nil, err
		}
		txList[index] = tx
	}
	return txList, nil
}

func (mpi *mempoolImpl) processFetchTxnResponse(fetchTxnResponse *FetchTxnResponse) error {
	mpi.logger.Debugf("Receive fetch transactions response from replica %d", fetchTxnResponse.ReplicaId)
	if _, ok := mpi.txStore.missingBatch[fetchTxnResponse.Height]; !ok {
		return errors.New("can't find batch %d from missingBatch")
	}
	expectLen := len(mpi.txStore.missingBatch[fetchTxnResponse.Height])
	recvLen := len(fetchTxnResponse.MissingTxnList)
	if recvLen != expectLen {
		return fmt.Errorf("receive unmatched fetching txn response, expect length: %d, received length: %d", expectLen, recvLen)
	}
	validTxn := make([]*pb.Transaction, 0)
	targetBatch := mpi.txStore.missingBatch[fetchTxnResponse.Height]
	for index, tx := range fetchTxnResponse.MissingTxnList {
		if tx.Hash().Hex() != targetBatch[index] {
			return errors.New("find a hash mismatch tx")
		}
		validTxn = append(validTxn, tx)
	}
	if err := mpi.processTransactions(validTxn); err != nil {
		return err
	}
	delete(mpi.txStore.missingBatch, fetchTxnResponse.Height)
	return nil
}
