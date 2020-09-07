package mempool

import (
	"sync"

	"github.com/google/btree"
	"github.com/meshplus/bitxhub-model/pb"
)

type transactionStore struct {
	txHashMap       map[string]bool
	allTxs          map[string]*txSortedMap
	nonceCache      *sync.Map
	parkingLotIndex *btreeIndex
	priorityIndex   *btreeIndex
}

func newTransactionStore() *transactionStore {
	return &transactionStore{
		txHashMap:       make(map[string]bool, 0),
		allTxs:          make(map[string]*txSortedMap),
		nonceCache:      &sync.Map{},
		parkingLotIndex: newBtreeIndex(),
		priorityIndex:   newBtreeIndex(),
	}
}

func (txStore *transactionStore) findLatestNonce(tx *pb.Transaction) uint64 {
	raw, ok := txStore.nonceCache.Load(tx.From.Hex())
	if !ok {
		return 0
	}
	return raw.(uint64)
}

func (txStore *transactionStore) hashOccurred(tx *pb.Transaction) bool {
	return txStore.txHashMap[tx.Hash().Hex()]
}

type txSortedMap struct {
	items map[uint64]*pb.Transaction // map nonce to transaction
	index *btreeIndex                // index for tx
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
