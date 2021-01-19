package mempool

import (
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/google/btree"
)

// the key of priorityIndex and parkingLotIndex.
type orderedIndexKey struct {
	account string
	nonce   uint64
}

// TODO (YH): add expiration time order
// Less should guarantee item can be cast into orderedIndexKey.
func (oik *orderedIndexKey) Less(than btree.Item) bool {
	other := than.(*orderedIndexKey)
	if oik.account != other.account {
		return oik.account < other.account
	}
	return oik.nonce < other.nonce
}

type sortedNonceKey struct {
	nonce uint64
}

// Less should guarantee item can be cast into sortedNonceKey.
func (snk *sortedNonceKey) Less(item btree.Item) bool {
	dst, _ := item.(*sortedNonceKey)
	return snk.nonce < dst.nonce
}

type orderedTimeoutKey struct {
	account   string
	nonce     uint64
	timestamp int64 // the timestamp of index key created
}

func (otk *orderedTimeoutKey) Less(than btree.Item) bool {
	other := than.(*orderedTimeoutKey)
	if otk.timestamp != other.timestamp {
		return otk.timestamp < other.timestamp
	}
	if otk.account != other.account {
		return otk.account < other.account
	}
	return otk.nonce < other.nonce
}

func makeOrderedIndexKey(account string, tx *pb.Transaction) *orderedIndexKey {
	return &orderedIndexKey{
		account: account,
		nonce:   tx.Nonce,
	}
}

func makeTimeoutKey(account string, tx *pb.Transaction) *orderedTimeoutKey {
	return &orderedTimeoutKey{
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

func makeAccountNonceKey(account string, nonce uint64) string {
	return fmt.Sprintf("%s-%d", account, nonce)
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

func (idx *btreeIndex) insertByTimeoutKey(account string, tx *pb.Transaction) {
	idx.data.ReplaceOrInsert(makeTimeoutKey(account, tx))
}

func (idx *btreeIndex) removeByTimeoutKey(txs map[string][]*pb.Transaction) {
	for account, list := range txs {
		for _, tx := range list {
			idx.data.Delete(makeTimeoutKey(account, tx))
		}
	}
}

// Size returns the size of the index
func (idx *btreeIndex) size() int {
	return idx.data.Len()
}
