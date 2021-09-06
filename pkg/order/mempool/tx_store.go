package mempool

import (
	"math"
	"sync"

	"github.com/google/btree"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

type transactionStore struct {
	// track all valid tx hashes cached in mempool
	txHashMap map[string]*orderedIndexKey
	// track all valid tx, mapping user' account to all related transactions.
	allTxs map[string]*txSortedMap
	// track the commit nonce and pending nonce of each account.
	nonceCache *nonceCache
	// keep track of the latest timestamp of ready txs in ttlIndex
	earliestTimestamp int64
	// keep track of the livetime of ready txs in priorityIndex
	ttlIndex *txLiveTimeMap
	// keeps track of "non-ready" txs (txs that can't be included in next block)
	// only used to help remove some txs if pool is full.
	parkingLotIndex *btreeIndex
	// keeps track of "ready" txs
	priorityIndex *btreeIndex
	// cache all the batched txs which haven't executed.
	batchedTxs map[orderedIndexKey]bool
	// track the non-batch priority transaction.
	priorityNonBatchSize uint64
}

func newTransactionStore(f GetAccountNonceFunc, logger logrus.FieldLogger) *transactionStore {
	return &transactionStore{
		txHashMap:       make(map[string]*orderedIndexKey, 0),
		allTxs:          make(map[string]*txSortedMap),
		batchedTxs:      make(map[orderedIndexKey]bool),
		parkingLotIndex: newBtreeIndex(),
		priorityIndex:   newBtreeIndex(),
		ttlIndex:        newTxLiveTimeMap(),
		nonceCache:      newNonceCache(f, logger),
	}
}

func (txStore *transactionStore) insertTxs(txs map[string][]*pb.Transaction, isLocal bool) map[string]bool {
	dirtyAccounts := make(map[string]bool)
	for account, list := range txs {
		for _, tx := range list {
			txHash := tx.TransactionHash.String()
			txPointer := &orderedIndexKey{
				account: account,
				nonce:   tx.Nonce,
			}
			txStore.txHashMap[txHash] = txPointer
			txList, ok := txStore.allTxs[account]
			if !ok {
				// if this is new account to send tx, create a new txSortedMap
				txStore.allTxs[account] = newTxSortedMap()
			}
			txList = txStore.allTxs[account]
			txItem := &txItem{
				account: account,
				tx:      tx,
				local:   isLocal,
			}
			txList.items[tx.Nonce] = txItem
			txList.index.insertBySortedNonceKey(tx)
			if isLocal {
				// no need to rebroadcast tx from other nodes to reduce network overhead
				txStore.ttlIndex.insertOrUpdateByTtlKey(account, tx.Nonce, tx.Timestamp)
			}
		}
		dirtyAccounts[account] = true
	}
	return dirtyAccounts
}

// Get transaction by account address + nonce
func (txStore *transactionStore) getTxByOrderKey(account string, seqNo uint64) *pb.Transaction {
	if list, ok := txStore.allTxs[account]; ok {
		res := list.items[seqNo]
		if res == nil {
			return nil
		}
		return res.tx
	}
	return nil
}

func (txStore *transactionStore) updateEarliestTimestamp() {
	// find the earliest tx in ttlIndex
	earliestTime := int64(math.MaxInt64)
	latestItem := txStore.ttlIndex.index.Min()
	if latestItem != nil {
		earliestTime = latestItem.(*orderedTimeoutKey).timestamp
	}
	txStore.earliestTimestamp = earliestTime
}

type txSortedMap struct {
	items map[uint64]*txItem // map nonce to transaction
	index *btreeIndex        // index for items
}

func newTxSortedMap() *txSortedMap {
	return &txSortedMap{
		items: make(map[uint64]*txItem),
		index: newBtreeIndex(),
	}
}

func (m *txSortedMap) filterReady(demandNonce uint64) ([]*pb.Transaction, []*pb.Transaction, uint64) {
	var readyTxs, nonReadyTxs []*pb.Transaction
	if m.index.data.Len() == 0 {
		return nil, nil, demandNonce
	}
	demandKey := makeSortedNonceKey(demandNonce)
	m.index.data.AscendGreaterOrEqual(demandKey, func(i btree.Item) bool {
		nonce := i.(*sortedNonceKey).nonce
		if nonce == demandNonce {
			readyTxs = append(readyTxs, m.items[demandNonce].tx)
			demandNonce++
		} else {
			nonReadyTxs = append(nonReadyTxs, m.items[nonce].tx)
		}
		return true
	})

	return readyTxs, nonReadyTxs, demandNonce
}

// forward removes all allTxs from the map with a nonce lower than the
// provided commitNonce.
func (m *txSortedMap) forward(commitNonce uint64) map[string][]*pb.Transaction {
	removedTxs := make(map[string][]*pb.Transaction)
	commitNonceKey := makeSortedNonceKey(commitNonce)
	m.index.data.AscendLessThan(commitNonceKey, func(i btree.Item) bool {
		// delete tx from map.
		nonce := i.(*sortedNonceKey).nonce
		txItem := m.items[nonce]
		account := txItem.account
		if _, ok := removedTxs[account]; !ok {
			removedTxs[account] = make([]*pb.Transaction, 0)
		}
		removedTxs[account] = append(removedTxs[account], txItem.tx)
		delete(m.items, nonce)
		return true
	})
	return removedTxs
}

type nonceCache struct {
	// commitNonces records each account's latest committed nonce in ledger.
	commitNonces map[string]uint64
	// pendingNonces records each account's latest nonce which has been included in
	// priority queue. Invariant: pendingNonces[account] >= commitNonces[account]
	pendingNonces   map[string]uint64
	pendingMu       sync.RWMutex
	commitMu        sync.Mutex
	getAccountNonce GetAccountNonceFunc
	logger          logrus.FieldLogger
}

func newNonceCache(f GetAccountNonceFunc, logger logrus.FieldLogger) *nonceCache {
	return &nonceCache{
		commitNonces:    make(map[string]uint64),
		pendingNonces:   make(map[string]uint64),
		getAccountNonce: f,
		logger:          logger,
	}
}

func (nc *nonceCache) getCommitNonce(account string) uint64 {
	nc.commitMu.Lock()
	defer nc.commitMu.Unlock()

	nonce, ok := nc.commitNonces[account]
	if !ok {
		cn := nc.getAccountNonce(types.NewAddressByStr(account))
		nc.commitNonces[account] = cn
		return cn
	}
	return nonce
}

func (nc *nonceCache) setCommitNonce(account string, nonce uint64) {
	nc.commitMu.Lock()
	defer nc.commitMu.Unlock()

	nc.commitNonces[account] = nonce
}

func (nc *nonceCache) getPendingNonce(account string) uint64 {
	nc.pendingMu.RLock()
	defer nc.pendingMu.RUnlock()
	nonce, ok := nc.pendingNonces[account]
	if !ok {
		// if nonce is unknown, check if there is committed nonce persisted in db
		// cause there are no pending txs in mempool now, pending nonce is equal to committed nonce
		return nc.getCommitNonce(account) + 1
	}
	return nonce
}

func (nc *nonceCache) setPendingNonce(account string, nonce uint64) {
	nc.pendingMu.Lock()
	defer nc.pendingMu.Unlock()
	nc.pendingNonces[account] = nonce
}

func (nc *nonceCache) updatePendingNonce(newPending map[string]uint64) {
	for account, pendingNonce := range newPending {
		nc.setPendingNonce(account, pendingNonce)
	}
}

func (nc *nonceCache) updateCommittedNonce(newCommitted map[string]uint64) {
	for account, committedNonce := range newCommitted {
		nc.setCommitNonce(account, committedNonce)
	}
}

// since the live time field in sortedTtlKey may vary during process
// we need to track the latest live time since its latest broadcast.
type txLiveTimeMap struct {
	items map[string]int64 // map account to its latest live time
	index *btree.BTree     // index for txs
}

func newTxLiveTimeMap() *txLiveTimeMap {
	return &txLiveTimeMap{
		index: btree.New(btreeDegree),
		items: make(map[string]int64),
	}
}

func (tlm *txLiveTimeMap) insertByTtlKey(account string, nonce uint64, liveTime int64) {
	tlm.index.ReplaceOrInsert(&orderedTimeoutKey{account, nonce, liveTime})
	tlm.items[makeAccountNonceKey(account, nonce)] = liveTime
}

func (tlm *txLiveTimeMap) removeByTtlKey(txs map[string][]*pb.Transaction) {
	for account, list := range txs {
		for _, tx := range list {
			liveTime, ok := tlm.items[makeAccountNonceKey(account, tx.Nonce)]
			if !ok {
				continue
			}
			tlm.index.Delete(&orderedTimeoutKey{account, tx.Nonce, liveTime})
			delete(tlm.items, makeAccountNonceKey(account, tx.Nonce))
		}
	}
}

func (tlm *txLiveTimeMap) updateByTtlKey(originalKey *orderedTimeoutKey, newTime int64) {
	tlm.index.Delete(originalKey)
	delete(tlm.items, makeAccountNonceKey(originalKey.account, originalKey.nonce))
	tlm.insertByTtlKey(originalKey.account, originalKey.nonce, newTime)
}

func (tlm *txLiveTimeMap) insertOrUpdateByTtlKey(account string, nonce uint64, liveTime int64) {
	accountNonceKey := makeAccountNonceKey(account, nonce)
	if oldTime, ok := tlm.items[accountNonceKey]; ok {
		tlm.index.Delete(&orderedTimeoutKey{account, nonce, oldTime})
	}
	tlm.insertByTtlKey(account, nonce, liveTime)
}
