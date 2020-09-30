package mempool

import (
	"testing"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	ast := assert.New(t)
	mempool, _ := mockMempoolImpl()
	defer cleanTestData()

	txList := make([]*pb.Transaction, 0)
	txHashList := make([]types.Hash,0)
	txHash1, _ := hex2Hash("txHash1")
	tx1 := &pb.Transaction{Nonce: uint64(1), TransactionHash: txHash1}
	txHash2, _ := hex2Hash("txHash2")
	tx2 := &pb.Transaction{Nonce: uint64(1), TransactionHash: txHash2}
	txList = append(txList, tx1, tx2)
	txHashList = append(txHashList, txHash1, txHash2)
	mempool.batchStore(txList)
	tx, ok := mempool.load(txHash1)
	ast.Equal(true,ok)
	ast.Equal(uint64(1),tx.Nonce)

	mempool.batchDelete(txHashList)
	tx, ok = mempool.load(txHash1)
	ast.Equal(false,ok)
	ast.Nil(tx)
}
