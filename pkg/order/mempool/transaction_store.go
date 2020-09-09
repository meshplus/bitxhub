package mempool

import (
	"github.com/google/btree"
	"github.com/meshplus/bitxhub-model/pb"
)

type transactionStore struct {
	txHashMap       map[string]bool
	allTxs          map[string]*txSortedMap
	nonceCache      map[string]uint64
	parkingLotIndex *btreeIndex
	priorityIndex   *btreeIndex
}

func newTransactionStore() *transactionStore {
	return &transactionStore{
		txHashMap:       make(map[string]bool, 0),
		allTxs:          make(map[string]*txSortedMap),
		nonceCache:      make(map[string]uint64),
		parkingLotIndex: newBtreeIndex(),
		priorityIndex:   newBtreeIndex(),
	}
}

func (txStore *transactionStore) getPendingNonce(account string) uint64 {
	nonce, ok := txStore.nonceCache[account]
	if !ok {
		return 0
	}
	return nonce
}

func (txStore *transactionStore) setPendingNonce(account string, nonce uint64) {
	txStore.nonceCache[account] = nonce
}

type txSortedMap struct {
	items map[uint64]*pb.Transaction // map nonce to transaction
	index *btreeIndex                // index for tx
}

func newTxSortedMap() *txSortedMap {
	return &txSortedMap{
		items: make(map[uint64]*pb.Transaction),
		index: newBtreeIndex(),
	}
}

func (m *txSortedMap) filterReady(demandKey *orderedQueueKey) ([]*pb.Transaction, []*pb.Transaction, uint64) {
	demandNonce := demandKey.seqNo
	var readyTxs, nonReadyTxs []*pb.Transaction
	if m.index.data.Len() == 0 {
		return nil, nil, demandNonce
	}

	m.index.data.AscendGreaterOrEqual(demandKey, func(i btree.Item) bool {
		orderedKey := i.(*orderedQueueKey)
		if orderedKey.seqNo == demandNonce {
			readyTxs = append(readyTxs, m.items[demandNonce])
			demandNonce++
		} else {
			nonReadyTxs = append(nonReadyTxs, m.items[orderedKey.seqNo])
		}
		return true
	})

	return readyTxs, nonReadyTxs, demandNonce
}
