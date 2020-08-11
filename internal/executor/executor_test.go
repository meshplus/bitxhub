package executor

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyPassword = "bitxhub"
	from        = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
)

func TestBlockExecutor_ExecuteBlock(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.String2Hash(from),
	}

	evs := make([]*pb.Event, 0)
	m := make(map[string]uint64)
	m[from] = 3
	data, err := json.Marshal(m)
	assert.Nil(t, err)
	ev := &pb.Event{
		TxHash:     types.String2Hash(from),
		Data:       data,
		Interchain: true,
	}
	evs = append(evs, ev)
	mockLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	mockLedger.EXPECT().Events(gomock.Any()).Return(evs).AnyTimes()
	mockLedger.EXPECT().Commit(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockLedger.EXPECT().Clear().AnyTimes()
	mockLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).Return(true, []byte("10")).AnyTimes()
	mockLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLedger.EXPECT().GetBalance(gomock.Any()).Return(uint64(10)).AnyTimes()
	mockLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()
	mockLedger.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	mockLedger.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	mockLedger.EXPECT().SetCode(gomock.Any(), gomock.Any()).AnyTimes()
	mockLedger.EXPECT().GetCode(gomock.Any()).Return([]byte("10")).AnyTimes()
	mockLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockLedger.EXPECT().FlushDirtyDataAndComputeJournal().Return(make(map[string]*ledger.Account), &ledger.BlockJournal{}).AnyTimes()
	mockLedger.EXPECT().PersistBlockData(gomock.Any()).AnyTimes()
	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger)
	assert.Nil(t, err)

	// mock data for block
	var txs []*pb.Transaction
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)
	pubKey := privKey.PublicKey()

	// set tx of TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(BVMData)
	txs = append(txs, BVMTx)
	// set tx of TransactionData_XVM type
	ibtp2 := mockIBTP(t, 2, pb.IBTP_INTERCHAIN)
	XVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_XVM, ibtp2)
	XVMTx := mockTx(XVMData)
	txs = append(txs, XVMTx)
	// set tx of TransactionData_NORMAL type
	ibtp3 := mockIBTP(t, 3, pb.IBTP_INTERCHAIN)
	NormalData := mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_XVM, ibtp3)
	NormalTx := mockTx(NormalData)
	txs = append(txs, NormalTx)

	// set signature for txs
	for _, tx := range txs {
		sig, err := privKey.Sign(tx.SignHash().Bytes())
		assert.Nil(t, err)
		tx.Signature = sig
		tx.From, err = pubKey.Address()
		assert.Nil(t, err)
	}
	// set invalid signature tx
	invalidTx := mockTx(nil)
	invalidTx.Signature = []byte("invalid")
	invalidTx.From = types.String2Address(from)
	txs = append(txs, invalidTx)

	assert.Nil(t, exec.Start())

	done := make(chan bool)
	ch := make(chan events.NewBlockEvent)
	blockSub := exec.SubscribeBlockEvent(ch)
	defer blockSub.Unsubscribe()

	// count received block to end test
	var wg sync.WaitGroup
	wg.Add(3)
	go listenBlock(&wg, done, ch)

	// send blocks to executor
	block2 := mockBlock(uint64(2), txs)
	block4 := mockBlock(uint64(4), txs)
	block3 := mockBlock(uint64(3), txs)
	block6 := mockBlock(uint64(6), txs)
	exec.ExecuteBlock(block2)
	exec.ExecuteBlock(block4)
	exec.ExecuteBlock(block6)
	exec.ExecuteBlock(block3)

	wg.Wait()
	done <- true
	assert.Nil(t, exec.Stop())
	assert.Equal(t, 2, exec.pendingBlockQ.Len())
	assert.Equal(t, uint64(5), exec.getDemandNumber())
}

func listenBlock(wg *sync.WaitGroup, done chan bool, blockCh chan events.NewBlockEvent) {
	for {
		select {
		case <-blockCh:
			wg.Done()
		case <-done:
			return
		}
	}
}

func mockBlock(blockNumber uint64, txs []*pb.Transaction) *pb.Block {
	header := &pb.BlockHeader{
		Number:    blockNumber,
		Timestamp: time.Now().UnixNano(),
	}
	return &pb.Block{
		BlockHeader:  header,
		Transactions: txs,
	}
}

func mockTx(data *pb.TransactionData) *pb.Transaction {
	return &pb.Transaction{
		Data:  data,
		Nonce: rand.Int63(),
	}
}

func TestBlockExecutor_ExecuteBlock_Transfer(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "executor")
	require.Nil(t, err)

	blockchainStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	require.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	require.Nil(t, err)

	accountCache := ledger.NewAccountCache()
	ldg, err := ledger.New(blockchainStorage, ldb, accountCache, log.NewWithModule("ledger"))
	require.Nil(t, err)

	_, from := loadAdminKey(t)

	ldg.SetBalance(from, 100000000)
	account, journal := ldg.FlushDirtyDataAndComputeJournal()
	err = ldg.Commit(1, account, journal)
	require.Nil(t, err)
	err = ldg.PersistExecutionResult(mockBlock(1, nil), nil, &pb.InterchainMeta{})
	require.Nil(t, err)

	executor, err := New(ldg, log.NewWithModule("executor"))
	require.Nil(t, err)
	err = executor.Start()
	require.Nil(t, err)

	ch := make(chan events.NewBlockEvent)
	sub := executor.SubscribeBlockEvent(ch)
	defer sub.Unsubscribe()

	var txs []*pb.Transaction
	txs = append(txs, mockTransferTx(t))
	txs = append(txs, mockTransferTx(t))
	txs = append(txs, mockTransferTx(t))
	executor.ExecuteBlock(mockBlock(2, txs))
	require.Nil(t, err)

	block := <-ch
	require.EqualValues(t, 2, block.Block.Height())
	require.EqualValues(t, uint64(99999997), ldg.GetBalance(from))

	// test executor with readonly ledger
	viewLedger, err := ledger.New(blockchainStorage, ldb, accountCache, log.NewWithModule("ledger"))
	require.Nil(t, err)

	exec, err := New(viewLedger, log.NewWithModule("executor"))
	require.Nil(t, err)

	tx := mockTransferTx(t)
	receipts := exec.ApplyReadonlyTransactions([]*pb.Transaction{tx})
	require.NotNil(t, receipts)
	require.Equal(t, pb.Receipt_SUCCESS, receipts[0].Status)
	require.Nil(t, receipts[0].Ret)
}

func mockTransferTx(t *testing.T) *pb.Transaction {
	privKey, from := loadAdminKey(t)
	to := randAddress(t)

	tx := &pb.Transaction{
		From:      from,
		To:        to,
		Timestamp: time.Now().UnixNano(),
		Data: &pb.TransactionData{
			Type:   pb.TransactionData_NORMAL,
			Amount: 1,
		},
		Nonce: rand.Int63(),
	}

	err := tx.Sign(privKey)
	require.Nil(t, err)
	tx.TransactionHash = tx.Hash()

	return tx
}

func loadAdminKey(t *testing.T) (crypto.PrivateKey, types.Address) {
	privKey, err := asym.RestorePrivateKey(filepath.Join("testdata", "key.json"), keyPassword)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return privKey, from
}

func randAddress(t *testing.T) types.Address {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)
	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return address
}

func mockTxData(t *testing.T, dataType pb.TransactionData_Type, vmType pb.TransactionData_VMType, ibtp proto.Marshaler) *pb.TransactionData {
	ib, err := ibtp.Marshal()
	assert.Nil(t, err)

	tmpIP := &pb.InvokePayload{
		Method: "set",
		Args:   []*pb.Arg{{Value: ib}},
	}
	pd, err := tmpIP.Marshal()
	assert.Nil(t, err)

	return &pb.TransactionData{
		VmType:  vmType,
		Type:    dataType,
		Amount:  10,
		Payload: pd,
	}
}

func mockIBTP(t *testing.T, index uint64, typ pb.IBTP_Type) *pb.IBTP {
	content := pb.Content{
		SrcContractId: from,
		DstContractId: from,
		Func:          "set",
	}

	bytes, err := content.Marshal()
	assert.Nil(t, err)

	ibtppd, err := json.Marshal(pb.Payload{
		Encrypted: false,
		Content:   bytes,
	})
	assert.Nil(t, err)

	return &pb.IBTP{
		From:      from,
		To:        from,
		Payload:   ibtppd,
		Index:     index,
		Type:      typ,
		Timestamp: time.Now().UnixNano(),
	}
}
