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
	ast.Equal(true, isLess)
}




