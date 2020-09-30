package mempool

import (
	"testing"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/stretchr/testify/assert"
)

func TestForward(t *testing.T) {
	ast := assert.New(t)
	mpi,_ := mockMempoolImpl()
	defer cleanTestData()

	txList := make([]*pb.Transaction,0)
	privKey1 := genPrivKey()
	account1,_ := privKey1.PublicKey().Address()
	tx1 := constructTx(uint64(1),&privKey1)
	tx2 := constructTx(uint64(2),&privKey1)
	tx3 := constructTx(uint64(3),&privKey1)
	tx4 := constructTx(uint64(4),&privKey1)
	tx5 := constructTx(uint64(6),&privKey1)
	txList = append(txList, tx1, tx2, tx3, tx4, tx5)
	err := mpi.processTransactions(txList)
	ast.Nil(err)
	list := mpi.txStore.allTxs[account1.Hex()]
	ast.Equal(5, list.index.size())
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())

	removeList := list.forward(uint64(3))
	ast.Equal(1, len(removeList))
	ast.Equal(2, len(removeList[account1.Hex()]))
	ast.Equal(uint64(1), removeList[account1.Hex()][0].Nonce)
	ast.Equal(uint64(2), removeList[account1.Hex()][1].Nonce)
}
