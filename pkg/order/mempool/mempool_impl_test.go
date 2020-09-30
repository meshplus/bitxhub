package mempool

import (
	"testing"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/stretchr/testify/assert"
)

func TestProcessTransactions(t *testing.T) {
	ast := assert.New(t)
	mpi, batchC := mockMempoolImpl()
	defer cleanTestData()

	txList := make([]*pb.Transaction, 0)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	privKey2 := genPrivKey()
	account2, _ := privKey2.PublicKey().Address()
	tx1 := constructTx(uint64(1), &privKey1)
	tx2 := constructTx(uint64(2), &privKey1)
	tx3 := constructTx(uint64(1), &privKey2)
	tx4 := constructTx(uint64(2), &privKey2)
	tx5 := constructTx(uint64(4), &privKey2)
	txList = append(txList, tx1, tx2, tx3, tx4, tx5)
	err := mpi.processTransactions(txList)
	ast.Nil(err)
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	ast.Equal(5, len(mpi.txStore.txHashMap))
	ast.Equal(0, len(mpi.txStore.batchedCache))
	ast.Equal(2, mpi.txStore.allTxs[account1.Hex()].index.size())
	ast.Equal(3, mpi.txStore.allTxs[account2.Hex()].index.size())
	ast.Equal(uint64(1), mpi.txStore.nonceCache.getCommitNonce(account1.Hex()))
	ast.Equal(uint64(3), mpi.txStore.nonceCache.getPendingNonce(account1.Hex()))
	ast.Equal(uint64(1), mpi.txStore.nonceCache.getCommitNonce(account2.Hex()))
	ast.Equal(uint64(3), mpi.txStore.nonceCache.getPendingNonce(account2.Hex()))

	go func() {
		mpi.batchSize = 4
		mpi.leader = mpi.localID
		tx6 := constructTx(uint64(3), &privKey1)
		tx7 := constructTx(uint64(5), &privKey2)
		txList = make([]*pb.Transaction, 0)
		txList = append(txList, tx6, tx7)
		err = mpi.processTransactions(txList)
		ast.Nil(err)
	}()
	select {
	case batch := <-batchC:
		ast.Equal(4, len(batch.TxHashes))
		ast.Equal(uint64(2), batch.Height)
		ast.Equal(uint64(1), mpi.txStore.priorityNonBatchSize)
		ast.Equal(5, mpi.txStore.priorityIndex.size())
		ast.Equal(2, mpi.txStore.parkingLotIndex.size())
		ast.Equal(7, len(mpi.txStore.txHashMap))
		ast.Equal(1, len(mpi.txStore.batchedCache))
		ast.Equal(4, len(mpi.txStore.batchedCache[uint64(2)]))
		ast.Equal(3, mpi.txStore.allTxs[account1.Hex()].index.size())
		ast.Equal(4, mpi.txStore.allTxs[account2.Hex()].index.size())
		ast.Equal(uint64(4), mpi.txStore.nonceCache.getPendingNonce(account1.Hex()))
		ast.Equal(uint64(3), mpi.txStore.nonceCache.getPendingNonce(account2.Hex()))
	}
}

func TestProcessFetchTxnRequest(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	defer cleanTestData()

	privKey1 := genPrivKey()
	tx1 := constructTx(uint64(1), &privKey1)
	var txList []*pb.Transaction
	txList = append(txList, tx1)

	missingList := make(map[uint64]string)
	missingList[0] = tx1.TransactionHash.Hex()
	fetchTxnRequest := &FetchTxnRequest{
		Height:          uint64(2),
		MissingTxHashes: missingList,
	}

	err := mpi.processFetchTxnRequest(fetchTxnRequest)
	ast.NotNil(err, " can't find the missing tx from local")

	// load tx from cache
	err = mpi.processTransactions(txList)
	mpi.txStore.batchedCache[uint64(2)] = txList
	err = mpi.processFetchTxnRequest(fetchTxnRequest)
	ast.Nil(err)
}

func TestProcessFetchTxnResponse(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	defer cleanTestData()

	fetchTxnResponse := &FetchTxnResponse{
		Height: uint64(2),
	}
	err := mpi.processFetchTxnResponse(fetchTxnResponse)
	ast.NotNil(err, "can't find batch 2 from missingBatch")

	privKey1 := genPrivKey()
	tx1 := constructTx(uint64(1), &privKey1)
	missingList := make(map[uint64]string)
	missingList[0] = tx1.TransactionHash.Hex()
	mpi.txStore.missingBatch[uint64(2)] = missingList

	fetchTxnResponse.MissingTxnList = make(map[uint64]*pb.Transaction)
	fetchTxnResponse.MissingTxnList[0] = tx1
	fetchTxnResponse.MissingTxnList[1] = tx1
	err = mpi.processFetchTxnResponse(fetchTxnResponse)
	ast.NotNil(err, "length mismatch")

	delete(fetchTxnResponse.MissingTxnList, 1)
	err = mpi.processFetchTxnResponse(fetchTxnResponse)
	ast.Nil(err)
	ast.Equal(0, len(mpi.txStore.missingBatch))
}
