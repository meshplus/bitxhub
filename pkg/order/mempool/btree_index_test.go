package mempool

import (
	"testing"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/stretchr/testify/assert"
)

func TestLess(t *testing.T) {
	ast := assert.New(t)
	tx := &pb.Transaction{
		Nonce: 1,
	}
	orderedIndexKey := makeOrderedIndexKey("account", tx)
	tx.Nonce = 2
	orderedIndexKey1 := makeOrderedIndexKey("bitxhub", tx)
	isLess := orderedIndexKey.Less(orderedIndexKey1)
	ast.Equal(true, isLess, "orderedIndexKey's account is less than orderedIndexKey1")
	tx.Nonce = 2
	orderedIndexKey2 := makeOrderedIndexKey("account", tx)
	isLess = orderedIndexKey.Less(orderedIndexKey2)
	ast.Equal(true, isLess, "orderedIndexKey's nonce is less than orderedIndexKey2")
}

func TestSortedNonceKeyLess(t *testing.T) {
	ast := assert.New(t)
	sortedNonceKey := makeSortedNonceKey(uint64(1))
	sortedNonceKey1 := makeSortedNonceKey(uint64(2))
	isLess := sortedNonceKey.Less(sortedNonceKey1)
	ast.Equal(true, isLess, "sortedNonceKey's nonce is less than sortedNonceKey1")
}

func TestSortedNonceIndex(t *testing.T) {
	ast := assert.New(t)
	tx := &pb.Transaction{
		Nonce: uint64(1),
	}
	btreeIndex :=newBtreeIndex()
	btreeIndex.insertBySortedNonceKey(tx)
	ast.Equal(1, btreeIndex.data.Len())

	txn := make(map[string][]*pb.Transaction)
	list := make([]*pb.Transaction,1)
	list[0] = tx
	txn["account"] = list
	btreeIndex.removeBySortedNonceKey(txn)
	ast.Equal(0, btreeIndex.data.Len())
}

func TestOrderedQueueIndex(t *testing.T) {
	ast := assert.New(t)
	tx := &pb.Transaction{
		Nonce: uint64(1),
	}
	btreeIndex :=newBtreeIndex()
	btreeIndex.insertByOrderedQueueKey("account",tx)
	ast.Equal(1, btreeIndex.data.Len())

	txn := make(map[string][]*pb.Transaction)
	list := make([]*pb.Transaction,1)
	list[0] = tx
	txn["account"] = list
	btreeIndex.removeByOrderedQueueKey(txn)
	ast.Equal(0, btreeIndex.data.Len())
}