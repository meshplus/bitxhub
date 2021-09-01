package mempool

import (
	"io/ioutil"
	"math"
	"os"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

func TestGetBlock(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	privKey1 := genPrivKey()
	privKey2 := genPrivKey()
	tx1 := constructTx(uint64(0), &privKey1)
	tx2 := constructTx(uint64(1), &privKey1)
	tx3 := constructTx(uint64(1), &privKey2)
	tx4 := constructTx(uint64(3), &privKey2)
	var txList []pb.Transaction
	txList = append(txList, tx1, tx2, tx3, tx4)
	// mock follower
	batch := mpi.ProcessTransactions(txList, false, true)
	ast.Nil(batch)
	// mock leader
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.Nil(batch)

	// mock leader to getBlock
	txList = make([]pb.Transaction, 0)
	tx5 := constructTx(uint64(0), &privKey2)
	txList = append(txList, tx5)
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.Equal(4, len(batch.TxList.Transactions))
}

func TestGetPendingNonceByAccount(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	nonce := mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(0), nonce)

	privKey2 := genPrivKey()
	account2, _ := privKey2.PublicKey().Address()
	tx1 := constructTx(uint64(0), &privKey1)
	tx2 := constructTx(uint64(1), &privKey1)
	tx3 := constructTx(uint64(0), &privKey2)
	tx4 := constructTx(uint64(1), &privKey2)
	tx5 := constructTx(uint64(3), &privKey2)
	var txList []pb.Transaction
	txList = append(txList, tx1, tx2, tx3, tx4, tx5)
	batch := mpi.ProcessTransactions(txList, false, true)
	ast.Nil(batch)
	nonce = mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(2), nonce)
	nonce = mpi.GetPendingNonceByAccount(account2.String())
	ast.Equal(uint64(2), nonce, "not 4")
	ifFull := mpi.IsPoolFull()
	ast.Equal(false, ifFull)
}

func TestCommitTransactions(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, batchC := mockMempoolImpl(storePath)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	nonce := mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(0), nonce)
	privKey2 := genPrivKey()
	tx1 := constructTx(uint64(0), &privKey1)
	tx2 := constructTx(uint64(1), &privKey1)
	tx3 := constructTx(uint64(0), &privKey2)
	tx4 := constructTx(uint64(3), &privKey2)
	var txList []pb.Transaction
	txList = append(txList, tx1, tx2, tx3, tx4)
	batch := mpi.ProcessTransactions(txList, true, true)
	ast.Nil(batch)
	ast.Equal(3, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())

	go func() {
		<-batchC
	}()
	tx5 := constructTx(uint64(1), &privKey2)
	txList = []pb.Transaction{}
	txList = append(txList, tx5)
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.Equal(4, len(batch.TxList.Transactions))
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	ast.Equal(uint64(2), mpi.batchSeqNo)

	var txHashList []*types.Hash
	txHashList = append(txHashList, tx1.GetHash(), tx2.GetHash(), tx3.GetHash(), tx5.GetHash())
	state := &ChainState{
		TxHashList: txHashList,
		Height:     uint64(2),
	}
	mpi.CommitTransactions(state)
	time.Sleep(100 * time.Millisecond)
	ast.Equal(0, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
}

func TestIncreaseChainHeight(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	ast.Equal(uint64(1), mpi.batchSeqNo)
	mpi.batchSeqNo++
	mpi.SetBatchSeqNo(mpi.batchSeqNo)
	ast.Equal(uint64(2), mpi.batchSeqNo)
}

func TestProcessTransactions(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	txList := make([]pb.Transaction, 0)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	privKey2 := genPrivKey()
	account2, _ := privKey2.PublicKey().Address()
	tx1 := constructTx(uint64(0), &privKey1)
	tx2 := constructTx(uint64(1), &privKey1)
	tx3 := constructTx(uint64(0), &privKey2)
	tx4 := constructTx(uint64(1), &privKey2)
	tx5 := constructTx(uint64(3), &privKey2)
	txList = append(txList, tx1, tx2, tx3, tx4, tx5)
	batch := mpi.ProcessTransactions(txList, false, true)
	ast.Nil(batch)
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	ast.Equal(5, len(mpi.txStore.txHashMap))
	ast.Equal(2, mpi.txStore.allTxs[account1.String()].index.size())
	ast.Equal(3, mpi.txStore.allTxs[account2.String()].index.size())
	ast.Equal(uint64(0), mpi.txStore.nonceCache.getCommitNonce(account1.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account1.String()))
	ast.Equal(uint64(0), mpi.txStore.nonceCache.getCommitNonce(account2.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account2.String()))

	mpi.batchSize = 4
	tx6 := constructTx(uint64(2), &privKey1)
	tx7 := constructTx(uint64(4), &privKey2)
	txList = make([]pb.Transaction, 0)
	txList = append(txList, tx6, tx7)
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.Equal(4, len(batch.TxList.Transactions))
	ast.Equal(uint64(2), batch.Height)
	ast.Equal(uint64(1), mpi.txStore.priorityNonBatchSize)
	ast.Equal(5, mpi.txStore.priorityIndex.size())
	ast.Equal(2, mpi.txStore.parkingLotIndex.size())
	ast.Equal(7, len(mpi.txStore.txHashMap))
	ast.Equal(3, mpi.txStore.allTxs[account1.String()].index.size())
	ast.Equal(4, mpi.txStore.allTxs[account2.String()].index.size())
	ast.Equal(uint64(3), mpi.txStore.nonceCache.getPendingNonce(account1.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account2.String()))
}

func TestForward(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	txList := make([]pb.Transaction, 0)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	tx1 := constructTx(uint64(0), &privKey1)
	tx2 := constructTx(uint64(1), &privKey1)
	tx3 := constructTx(uint64(2), &privKey1)
	tx4 := constructTx(uint64(3), &privKey1)
	tx5 := constructTx(uint64(5), &privKey1)
	txList = append(txList, tx1, tx2, tx3, tx4, tx5)
	batch := mpi.ProcessTransactions(txList, false, true)
	ast.Nil(batch)
	list := mpi.txStore.allTxs[account1.String()]
	ast.Equal(5, list.index.size())
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())

	removeList := list.forward(uint64(2))
	ast.Equal(1, len(removeList))
	ast.Equal(2, len(removeList[account1.String()]))
	ast.Equal(uint64(0), removeList[account1.String()][0].GetNonce())
	ast.Equal(uint64(1), removeList[account1.String()][1].GetNonce())
}

func TestUnorderedIncomingTxs(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	mpi.batchSize = 5

	txList := make([]pb.Transaction, 0)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	privKey2 := genPrivKey()
	account2, _ := privKey2.PublicKey().Address()
	nonceArr := []uint64{3, 1, 0}
	for _, i := range nonceArr {
		tx1 := constructTx(i, &privKey1)
		tx2 := constructTx(i, &privKey2)
		txList = append(txList, tx1, tx2)
	}
	batch := mpi.ProcessTransactions(txList, true, true)
	ast.Nil(batch) // not enough for 5 txs to generate batch
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(2, mpi.txStore.parkingLotIndex.size())
	ast.Equal(6, len(mpi.txStore.txHashMap))
	ast.Equal(3, mpi.txStore.allTxs[account1.String()].index.size())
	ast.Equal(3, mpi.txStore.allTxs[account2.String()].index.size())
	ast.Equal(uint64(0), mpi.txStore.nonceCache.getCommitNonce(account1.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account1.String()))
	ast.Equal(uint64(0), mpi.txStore.nonceCache.getCommitNonce(account2.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account2.String()))

	tx7 := constructTx(uint64(2), &privKey1)
	tx8 := constructTx(uint64(4), &privKey2)
	txList = make([]pb.Transaction, 0)
	txList = append(txList, tx7, tx8)
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.NotNil(batch)

	var hashes []types.Hash
	// this batch will contain tx{1,2,3} for privKey1 and tx{1,2} for privKey2
	ast.Equal(5, len(batch.TxList.Transactions))
	ast.Equal(uint64(2), batch.Height)
	ast.Equal(uint64(1), mpi.txStore.priorityNonBatchSize)
	ast.Equal(6, mpi.txStore.priorityIndex.size())
	ast.Equal(3, mpi.txStore.parkingLotIndex.size(), "delete parkingLot until finishing executor")
	ast.Equal(8, len(mpi.txStore.txHashMap))
	ast.Equal(4, mpi.txStore.allTxs[account1.String()].index.size())
	ast.Equal(4, mpi.txStore.allTxs[account2.String()].index.size())
	ast.Equal(uint64(4), mpi.txStore.nonceCache.getPendingNonce(account1.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account2.String()))

	// process committed txs
	hashList := make([]*types.Hash, 0, len(hashes))
	for _, tx := range batch.TxList.Transactions {
		hashList = append(hashList, tx.GetHash())
	}
	ready := &ChainState{
		TxHashList: hashList,
		Height:     2,
	}
	mpi.processCommitTransactions(ready)
	time.Sleep(100 * time.Millisecond)

	ast.Equal(uint64(1), mpi.txStore.priorityNonBatchSize)
	ast.Equal(1, mpi.txStore.priorityIndex.size())
	ast.Equal(3, mpi.txStore.parkingLotIndex.size(), "delete parkingLot until finishing executor")
	ast.Equal(3, len(mpi.txStore.txHashMap))
	ast.Equal(1, mpi.txStore.allTxs[account1.String()].index.size())
	ast.Equal(2, mpi.txStore.allTxs[account2.String()].index.size())
	ast.Equal(uint64(4), mpi.txStore.nonceCache.getPendingNonce(account1.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getCommitNonce(account2.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account2.String()))

	// generate block3
	txList = make([]pb.Transaction, 0)
	account1NonceArr2 := []uint64{4, 5}
	for _, i := range account1NonceArr2 {
		tx1 := constructTx(i, &privKey1)
		txList = append(txList, tx1)
	}
	account2NonceArr2 := []uint64{2, 5}
	for _, i := range account2NonceArr2 {
		tx1 := constructTx(i, &privKey2)
		txList = append(txList, tx1)
	}
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.NotNil(batch)
	ast.Equal(uint64(2), mpi.txStore.priorityNonBatchSize)
	ast.Equal(7, mpi.txStore.priorityIndex.size())
	ast.Equal(3, mpi.txStore.parkingLotIndex.size(), "delete parkingLot until finishing executor")
	ast.Equal(7, len(mpi.txStore.txHashMap))
	ast.Equal(3, mpi.txStore.allTxs[account1.String()].index.size())
	ast.Equal(4, mpi.txStore.allTxs[account2.String()].index.size())
	ast.Equal(uint64(3), mpi.txStore.nonceCache.getCommitNonce(account1.String()))
	ast.Equal(uint64(6), mpi.txStore.nonceCache.getPendingNonce(account1.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getCommitNonce(account2.String()))
	ast.Equal(uint64(6), mpi.txStore.nonceCache.getPendingNonce(account2.String()))

}

func TestGetTimeoutTransaction(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	mpi.txSliceSize = 3
	allTxHashes := make([]*types.Hash, 0)

	txList := make([]pb.Transaction, 0)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	privKey2 := genPrivKey()
	account2, _ := privKey2.PublicKey().Address()
	nonceArr := []uint64{3, 4}
	for _, i := range nonceArr {
		tx1 := constructTx(i, &privKey1)
		tx2 := constructTx(i, &privKey2)
		txList = append(txList, tx1, tx2)
		allTxHashes = append(allTxHashes, tx1.GetHash(), tx2.GetHash())
	}
	batch := mpi.ProcessTransactions(txList, true, true)
	ast.Nil(batch)

	// set another incoming list for timeout
	time.Sleep(150 * time.Millisecond)
	nonceArr = []uint64{0, 1}
	for _, i := range nonceArr {
		tx1 := constructTx(i, &privKey1)
		tx2 := constructTx(i, &privKey2)
		txList = append(txList, tx1, tx2)
		allTxHashes = append(allTxHashes, tx1.GetHash(), tx2.GetHash())
	}
	batch = mpi.ProcessTransactions(txList, true, false)
	ast.NotNil(batch)
	// tx1,tx2 for account1 and account2 will be batched.
	// all txs for account1 and account2 will be in priorityIndex.
	// And tx4,tx5 for account1 and account2 will be timeout while tx3 will not
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(4, mpi.txStore.parkingLotIndex.size())
	ast.Equal(8, len(mpi.txStore.txHashMap))
	ast.Equal(4, mpi.txStore.allTxs[account1.String()].index.size())
	ast.Equal(4, mpi.txStore.allTxs[account2.String()].index.size())
	ast.Equal(uint64(0), mpi.txStore.nonceCache.getCommitNonce(account1.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account1.String()))
	ast.Equal(uint64(0), mpi.txStore.nonceCache.getCommitNonce(account2.String()))
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(account2.String()))

	tx1 := constructTx(2, &privKey1)
	tx2 := constructTx(2, &privKey2)
	txList = append(txList, tx1, tx2)
	allTxHashes = append(allTxHashes, tx1.GetHash(), tx2.GetHash())
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.NotNil(batch)

	// tx4,tx5 should be timeout
	timeoutList := mpi.GetTimeoutTransactions(100 * time.Millisecond)
	ast.NotNil(timeoutList)
	ast.Equal(2, len(timeoutList))
	ast.Equal(3, len(timeoutList[0]))
	ast.Equal(1, len(timeoutList[1]))
	ast.Equal(6, mpi.txStore.ttlIndex.index.Len())

	// wait another 150 millisecond, tx3 be timeout too.
	// though tx1,tx2 has wait a long time, they are not local and they won't be broadcast
	time.Sleep(150 * time.Millisecond)
	timeoutList = mpi.GetTimeoutTransactions(100 * time.Millisecond)
	ast.NotNil(timeoutList)
	ast.Equal(2, len(timeoutList))
	ast.Equal(3, len(timeoutList[0]))
	ast.Equal(3, len(timeoutList[1]))
	ast.Equal(6, mpi.txStore.ttlIndex.index.Len())

	// check if all indices are normally cleaned after commit
	ast.Equal(10, len(allTxHashes))
	state := &ChainState{
		TxHashList: allTxHashes,
		Height:     uint64(2),
	}
	mpi.CommitTransactions(state)
	time.Sleep(100 * time.Millisecond)
	ast.Equal(0, mpi.txStore.ttlIndex.index.Len())
	ast.Equal(0, len(mpi.txStore.ttlIndex.items))
	ast.Equal(int64(math.MaxInt64), mpi.txStore.earliestTimestamp)
	ast.Equal(0, mpi.txStore.priorityIndex.size())
	ast.Equal(0, mpi.txStore.parkingLotIndex.size())
	ast.Equal(0, len(mpi.txStore.batchedTxs))
	ast.Equal(0, len(mpi.txStore.txHashMap))
	ast.Equal(uint64(0), mpi.txStore.priorityNonBatchSize)
}

func TestRestore(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, batchC := mockMempoolImpl(storePath)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	nonce := mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(0), nonce)
	privKey2 := genPrivKey()
	//account2, _ := privKey1.PublicKey().Address()
	tx1 := constructTx(uint64(0), &privKey1)
	tx2 := constructTx(uint64(1), &privKey1)
	tx3 := constructTx(uint64(0), &privKey2)
	tx4 := constructTx(uint64(3), &privKey2)
	var txList []pb.Transaction
	txList = append(txList, tx1, tx2, tx3, tx4)
	batch := mpi.ProcessTransactions(txList, true, true)
	ast.Nil(batch)
	ast.Equal(3, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())

	go func() {
		<-batchC
	}()
	tx5 := constructTx(uint64(1), &privKey2)
	txList = []pb.Transaction{}
	txList = append(txList, tx5)
	batch = mpi.ProcessTransactions(txList, true, true)
	ast.Equal(4, len(batch.TxList.Transactions))
	ast.Equal(4, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())
	ast.Equal(uint64(2), mpi.batchSeqNo)

	var txHashList []*types.Hash
	txHashList = append(txHashList, tx1.GetHash(), tx2.GetHash(), tx3.GetHash(), tx5.GetHash())
	state := &ChainState{
		TxHashList: txHashList,
		Height:     uint64(2),
	}
	mpi.CommitTransactions(state)
	time.Sleep(100 * time.Millisecond)
	ast.Equal(0, mpi.txStore.priorityIndex.size())
	ast.Equal(1, mpi.txStore.parkingLotIndex.size())

	// stop and restore
	//ast.Nil(mpi.txStore.nonceCache.fallback.Close())
	//newMpi, _ := mockMempoolImpl(storePath)
	//ast.Equal(uint64(3), newMpi.txStore.nonceCache.getCommitNonce(account1.String()))
	//ast.Equal(uint64(3), newMpi.txStore.nonceCache.getCommitNonce(account2.String()))
	//ast.Equal(uint64(3), newMpi.txStore.nonceCache.getPendingNonce(account1.String()))
	//ast.Equal(uint64(3), newMpi.txStore.nonceCache.getPendingNonce(account2.String()))
}

func TestGenerateBlock(t *testing.T) {
	ast := assert.New(t)
	storePath, err := ioutil.TempDir("", "mempool")
	ast.Nil(err)
	defer os.RemoveAll(storePath)
	mpi, _ := mockMempoolImpl(storePath)
	privKey1 := genPrivKey()
	account1, _ := privKey1.PublicKey().Address()
	nonce := mpi.GetPendingNonceByAccount(account1.String())
	ast.Equal(uint64(0), nonce)
	privKey2 := genPrivKey()
	//account2, _ := privKey1.PublicKey().Address()
	tx1 := constructTx(uint64(0), &privKey1)
	tx2 := constructTx(uint64(1), &privKey1)
	tx3 := constructTx(uint64(0), &privKey2)
	//tx4 := constructTx(uint64(1), &privKey2)
	tx5 := constructTx(uint64(2), &privKey2)
	time.Sleep(10 * time.Millisecond)

	txList := make([]pb.Transaction, 0)
	txList = append(txList, tx1, tx2, tx3, tx5)
	batches := mpi.ProcessTransactions(txList, true, true)
	ast.Nil(batches)
	ast.Equal(true, mpi.HasPendingRequest())
	ast.Equal(uint64(3), mpi.txStore.priorityNonBatchSize)
	ast.Equal(uint64(2), mpi.txStore.nonceCache.getPendingNonce(tx1.GetFrom().String()))

	blockBatches := mpi.GenerateBlock()
	ast.Equal(uint64(2), blockBatches.Height)
	ast.Equal(false, mpi.HasPendingRequest())
	ast.Equal(uint64(0), mpi.txStore.priorityNonBatchSize)
	ast.Equal(3, len(blockBatches.TxList.Transactions))
}
