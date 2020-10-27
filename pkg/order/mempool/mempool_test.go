package mempool

import (
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"

	"github.com/stretchr/testify/assert"
)

func TestRecvTransaction(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	defer cleanTestData()

	privKey1 := genPrivKey()
	tx1 := constructTx(uint64(1), &privKey1)
	go mpi.txCache.listenEvent()
	go func() {
		_ = mpi.RecvTransaction(tx1)
	}()
	select {
	case txSet := <-mpi.txCache.txSetC:
		ast.Equal(1, len(txSet.TxList))
	}

	err := mpi.Start()
	ast.Nil(err)
	privKey2 := genPrivKey()
	go func() {
		_ = mpi.RecvTransaction(tx1)
	}()
	time.Sleep(1 * time.Millisecond)
	ast.Equal(1, mpi.txStore.priorityIndex.size())
	ast.Equal(0, mpi.txStore.parkingLotIndex.size())

	tx2 := constructTx(uint64(2), &privKey1)
	tx3 := constructTx(uint64(1), &privKey2)
	tx4 := constructTx(uint64(2), &privKey2)
	go func() {
		_ = mpi.RecvTransaction(tx4)
	}()
	time.Sleep(1 * time.Millisecond)
	ast.Equal(1, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	go func() {
		_ = mpi.RecvTransaction(tx2)
	}()
	time.Sleep(1 * time.Millisecond)
	ast.Equal(2, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	go func() {
		_ = mpi.RecvTransaction(tx3)
	}()
	time.Sleep(1 * time.Millisecond)
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size(), "delete tx4 until finishing executor")
	mpi.Stop()
}

func TestRecvForwardTxs(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	defer cleanTestData()

	privKey1 := genPrivKey()
	tx := constructTx(uint64(1), &privKey1)
	txList := []*pb.Transaction{tx}
	txSlice := &TxSlice{TxList: txList}
	go mpi.RecvForwardTxs(txSlice)
	select {
	case txSet := <-mpi.subscribe.txForwardC:
		ast.Equal(1, len(txSet.TxList))
	}
}

func TestUpdateLeader(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	mpi.Start()
	defer cleanTestData()
	go mpi.UpdateLeader(uint64(2))
	time.Sleep(1 * time.Millisecond)
	ast.Equal(uint64(2), mpi.leader)
}

func TestGetBlock(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	err := mpi.Start()
	ast.Nil(err)
	defer cleanTestData()

	privKey1 := genPrivKey()
	privKey2 := genPrivKey()
	tx1 := constructTx(uint64(1), &privKey1)
	tx2 := constructTx(uint64(2), &privKey1)
	tx3 := constructTx(uint64(2), &privKey2)
	tx4 := constructTx(uint64(4), &privKey2)
	tx5 := constructTx(uint64(1), &privKey2)
	var txList []*pb.Transaction
	var txHashList []types.Hash
	txList = append(txList, tx1, tx2, tx3, tx4)
	txHashList = append(txHashList, *tx1.TransactionHash, *tx2.TransactionHash, *tx3.TransactionHash, *tx5.TransactionHash)
	err = mpi.processTransactions(txList)
	ast.Nil(err)
	ready := &raftproto.Ready{
		Height:   uint64(2),
		TxHashes: txHashList,
	}
	missingTxnHashList, txList := mpi.GetBlockByHashList(ready)
	ast.Equal(1, len(missingTxnHashList), "missing tx5")
	ast.Equal(3, len(txList))

	// mock leader to getBlock
	mpi.leader = uint64(1)
	missingTxnHashList, txList = mpi.GetBlockByHashList(ready)
	ast.Equal(4, len(missingTxnHashList))
	ast.Equal(0, len(txList))

	// mock follower
	mpi.leader = uint64(2)
	txList = []*pb.Transaction{}
	txList = append(txList, tx5)
	err = mpi.processTransactions(txList)
	missingTxnHashList, txList = mpi.GetBlockByHashList(ready)
	ast.Equal(0, len(missingTxnHashList))
	ast.Equal(4, len(txList))

	// mock leader to getBlock
	mpi.leader = uint64(1)
	missingTxnHashList, txList = mpi.GetBlockByHashList(ready)
	ast.Equal(0, len(missingTxnHashList))
	ast.Equal(4, len(txList))
}

func TestGetPendingNonceByAccount(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	err := mpi.Start()
	ast.Nil(err)
	defer cleanTestData()

	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	nonce := mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(1), nonce)

	privKey2 := genPrivKey()
	account2, _ := privKey2.PublicKey().Address()
	tx1 := constructTx(uint64(1), &privKey1)
	tx2 := constructTx(uint64(2), &privKey1)
	tx3 := constructTx(uint64(1), &privKey2)
	tx4 := constructTx(uint64(2), &privKey2)
	tx5 := constructTx(uint64(4), &privKey2)
	var txList []*pb.Transaction
	txList = append(txList, tx1, tx2, tx3, tx4, tx5)
	err = mpi.processTransactions(txList)
	ast.Nil(err)
	nonce = mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(3), nonce)
	nonce = mpi.GetPendingNonceByAccount(account2.String())
	ast.Equal(uint64(3), nonce, "not 4")
}

func TestCommitTransactions(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	err := mpi.Start()
	ast.Nil(err)
	defer cleanTestData()

	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	nonce := mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(1), nonce)

	privKey2 := genPrivKey()
	tx1 := constructTx(uint64(1), &privKey1)
	tx2 := constructTx(uint64(2), &privKey1)
	tx3 := constructTx(uint64(1), &privKey2)
	tx4 := constructTx(uint64(4), &privKey2)
	var txList []*pb.Transaction
	txList = append(txList, tx1, tx2, tx3, tx4)
	mpi.leader = uint64(1)
	err = mpi.processTransactions(txList)
	ast.Equal(3, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	ast.Equal(0, len(mpi.txStore.batchedCache))

	go func() {
		<-mpi.batchC
	}()
	tx5 := constructTx(uint64(2), &privKey2)
	txList = []*pb.Transaction{}
	txList = append(txList, tx5)
	err = mpi.processTransactions(txList)
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	ast.Equal(1, len(mpi.txStore.batchedCache))
	height := mpi.GetChainHeight()
	ast.Equal(uint64(2), height)

	var txHashList []types.Hash
	txHashList = append(txHashList, *tx1.TransactionHash, *tx2.TransactionHash, *tx3.TransactionHash, *tx5.TransactionHash)
	ready := &raftproto.Ready{
		Height:   uint64(2),
		TxHashes: txHashList,
	}
	mpi.CommitTransactions(ready)
	time.Sleep(100 * time.Millisecond)
	ast.Equal(0, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	ast.Equal(0, len(mpi.txStore.batchedCache))
}

func TestFetchTxn(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	err := mpi.Start()
	ast.Nil(err)
	defer cleanTestData()

	missingList := make(map[uint64]string)
	missingList[0] = "tx1"
	lostTxnEvent := &LocalMissingTxnEvent{
		Height:             uint64(2),
		MissingTxnHashList: missingList,
		WaitC:              make(chan bool),
	}
	mpi.FetchTxn(lostTxnEvent)
	time.Sleep(10 * time.Millisecond)
	ast.Equal(1, len(mpi.txStore.missingBatch))
}

func TestIncreaseChainHeight(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	defer cleanTestData()

	ast.Equal(uint64(1), mpi.GetChainHeight())
	mpi.increaseBatchSeqNo()
	ast.Equal(uint64(2), mpi.GetChainHeight())
}
