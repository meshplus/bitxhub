package mempool

import (
	"github.com/google/btree"
	"github.com/meshplus/bitxhub-model/pb"
)

// the key of priorityIndex and parkingLotIndex.
type orderedIndexKey struct {
	account   string
	nonce     uint64
	timestamp int64 // the timestamp of tx when it is received
}

// TODO (YH): add expiration time order
// Less should guarantee item can be cast into orderedIndexKey.
func (oik *orderedIndexKey) Less(than btree.Item) bool {
	other := than.(*orderedIndexKey)
	if oik.account != other.account {
		return oik.timestamp < other.timestamp
	}
	return oik.nonce < other.nonce
}

type sortedNonceKey struct {
	nonce uint64
}

type sortedTtlKey struct {
	account  string
	nonce    uint64
	liveTime int64 // the timestamp of tx when it is received or rebroadcast
}

func (stk *sortedTtlKey) Less(than btree.Item) bool {
	other := than.(*sortedTtlKey)
	return stk.liveTime < other.liveTime
}

// Less should guarantee item can be cast into sortedNonceKey.
func (snk *sortedNonceKey) Less(item btree.Item) bool {
	dst, _ := item.(*sortedNonceKey)
	return snk.nonce < dst.nonce
}

func makeOrderedIndexKey(account string, tx *pb.Transaction) *orderedIndexKey {
	return &orderedIndexKey{
		account:   account,
		nonce:     tx.Nonce,
		timestamp: tx.Timestamp,
	}
}

func makeSortedNonceKey(nonce uint64) *sortedNonceKey {
	return &sortedNonceKey{
		nonce: nonce,
	}
}

type btreeIndex struct {
	data *btree.BTree
}

func newBtreeIndex() *btreeIndex {
	return &btreeIndex{
		data: btree.New(btreeDegree),
	}
}

func (idx *btreeIndex) insertBySortedNonceKey(tx *pb.Transaction) {
	idx.data.ReplaceOrInsert(makeSortedNonceKey(tx.Nonce))
}

func (idx *btreeIndex) removeBySortedNonceKey(txs map[string][]*pb.Transaction) {
	for _, list := range txs {
		for _, tx := range list {
			idx.data.Delete(makeSortedNonceKey(tx.Nonce))
		}
	}
}

func (idx *btreeIndex) insertByOrderedQueueKey(account string, tx *pb.Transaction) {
	idx.data.ReplaceOrInsert(makeOrderedIndexKey(account, tx))
}

func (idx *btreeIndex) removeByOrderedQueueKey(txs map[string][]*pb.Transaction) {
	for account, list := range txs {
		for _, tx := range list {
			idx.data.Delete(makeOrderedIndexKey(account, tx))
		}
	}
}

// Size returns the size of the index
func (idx *btreeIndex) size() int {
	return idx.data.Len()
}
