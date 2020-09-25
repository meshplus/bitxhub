package mempool

import (
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

func makeOrderedIndexKey(account string, tx *pb.Transaction) *orderedIndexKey {
	return &orderedIndexKey{
		account: account,
		nonce:   uint64(tx.Nonce),
	}
}

func makeSortedNonceKeyKey(nonce uint64) *sortedNonceKey {
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

func (idx *btreeIndex) insert(tx *pb.Transaction) {
	idx.data.ReplaceOrInsert(makeSortedNonceKeyKey(uint64(tx.Nonce)))
}

func (idx *btreeIndex) remove(txs map[string][]*pb.Transaction) {
	for _, list := range txs {
		for _, tx := range list {
			idx.data.Delete(makeSortedNonceKeyKey(uint64(tx.Nonce)))
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
func (idx *btreeIndex) size() uint64 {
	return uint64(idx.data.Len())
}
