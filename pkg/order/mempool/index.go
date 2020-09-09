package mempool

import (
	"github.com/google/btree"
	"github.com/meshplus/bitxhub-model/pb"
)

type orderedQueueKey struct {
	accountAddress string
	seqNo          uint64
}

func (item *orderedQueueKey) Less(than btree.Item) bool {
	other := than.(*orderedQueueKey)
	if item.accountAddress != other.accountAddress {
		return item.accountAddress < other.accountAddress
	}
	return item.seqNo < other.seqNo
}

func makeKey(tx *pb.Transaction) *orderedQueueKey {
	return &orderedQueueKey{
		accountAddress: tx.From.Hex(),
		seqNo:          uint64(tx.Nonce),
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

func (idx *btreeIndex) insert(txs []*pb.Transaction) {
	for _, tx := range txs {
		idx.data.ReplaceOrInsert(makeKey(tx))
	}
}

func (idx *btreeIndex) remove(txs []*pb.Transaction) {
	for _, tx := range txs {
		idx.data.Delete(makeKey(tx))
	}
}
