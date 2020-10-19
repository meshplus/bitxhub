package executor

import (
	"encoding/json"
	"fmt"
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
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/internal/storages/blockfile"
	"github.com/meshplus/bitxhub/pkg/cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyPassword  = "bitxhub"
	from         = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	to           = "0x000018f7c3a6e5e4c0b877fe3e688ab08840b997"
	executorType = "serial"
)

func TestNew(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}
	mockLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()

	logger := log.NewWithModule("executor")
	executor, err := New(mockLedger, logger, executorType)
	assert.Nil(t, err)
	assert.NotNil(t, executor)

	assert.Equal(t, mockLedger, executor.ledger)
	assert.Equal(t, logger, executor.logger)
	assert.NotNil(t, executor.preBlockC)
	assert.NotNil(t, executor.blockC)
	assert.NotNil(t, executor.persistC)
	assert.NotNil(t, executor.ibtpVerify)
	assert.NotNil(t, executor.validationEngine)
	assert.Equal(t, chainMeta.BlockHash, executor.currentBlockHash)
	assert.Equal(t, chainMeta.Height, executor.currentHeight)
	assert.NotNil(t, executor.wasmInstances)
	assert.Equal(t, 0, len(executor.wasmInstances))
}

func TestBlockExecutor_ExecuteBlock(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.NewHash([]byte(from)),
	}

	evs := make([]*pb.Event, 0)
	m := make(map[string]uint64)
	m[from] = 3
	data, err := json.Marshal(m)
	assert.Nil(t, err)
	ev := &pb.Event{
		TxHash:     types.NewHash([]byte(from)),
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
	mockLedger.EXPECT().FlushDirtyDataAndComputeJournal().Return(make(map[string]*ledger.Account), &ledger.BlockJournal{ChangedHash: &types.Hash{}}).AnyTimes()
	mockLedger.EXPECT().PersistBlockData(gomock.Any()).AnyTimes()
	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, executorType)
	assert.Nil(t, err)

	// mock data for block
	var txs []*pb.Transaction
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)
	pubKey := privKey.PublicKey()

	// set tx of TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(t, BVMData)
	txs = append(txs, BVMTx)
	// set tx of TransactionData_XVM type
	ibtp2 := mockIBTP(t, 2, pb.IBTP_INTERCHAIN)
	XVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_XVM, ibtp2)
	XVMTx := mockTx(t, XVMData)
	txs = append(txs, XVMTx)
	// set tx of TransactionData_NORMAL type
	ibtp3 := mockIBTP(t, 3, pb.IBTP_INTERCHAIN)
	NormalData := mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_XVM, ibtp3)
	NormalTx := mockTx(t, NormalData)
	txs = append(txs, NormalTx)
	// set tx with empty transaction data
	emptyDataTx := mockTx(t, nil)
	txs = append(txs, emptyDataTx)

	// set signature for txs
	for _, tx := range txs {
		tx.From, err = pubKey.Address()
		assert.Nil(t, err)
		sig, err := privKey.Sign(tx.SignHash().Bytes())
		assert.Nil(t, err)
		tx.Signature = sig
		tx.TransactionHash = tx.Hash()
	}
	// set invalid signature tx
	invalidTx := mockTx(t, nil)
	invalidTx.From = types.NewAddressByStr(from)
	invalidTx.Signature = []byte("invalid")
	txs = append(txs, invalidTx)

	assert.Nil(t, exec.Start())

	done := make(chan bool)
	ch := make(chan events.NewBlockEvent)
	blockSub := exec.SubscribeBlockEvent(ch)
	defer blockSub.Unsubscribe()

	// count received block to end test
	var wg sync.WaitGroup
	wg.Add(2)
	go listenBlock(&wg, done, ch)

	// send blocks to executor
	block1 := mockBlock(uint64(1), nil)
	block2 := mockBlock(uint64(2), txs)
	exec.ExecuteBlock(block1)
	exec.ExecuteBlock(block2)

	wg.Wait()
	done <- true
	assert.Nil(t, exec.Stop())
}

func TestBlockExecutor_ApplyReadonlyTransactions(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockLedger := mock_ledger.NewMockLedger(mockCtl)

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}

	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)

	addr, err := privKey.PublicKey().Address()
	assert.Nil(t, err)
	id := fmt.Sprintf("%s-%s-%d", addr.String(), to, 1)

	hash := types.NewHash([]byte{1})
	val, err := json.Marshal(hash)
	assert.Nil(t, err)

	contractAddr := constant.InterchainContractAddr.Address()

	mockLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	mockLedger.EXPECT().Events(gomock.Any()).Return(nil).AnyTimes()
	mockLedger.EXPECT().Commit(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockLedger.EXPECT().Clear().AnyTimes()
	mockLedger.EXPECT().GetState(contractAddr, []byte(fmt.Sprintf("index-tx-%s", id))).Return(true, val).AnyTimes()
	mockLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockLedger.EXPECT().FlushDirtyDataAndComputeJournal().Return(make(map[string]*ledger.Account), &ledger.BlockJournal{}).AnyTimes()
	mockLedger.EXPECT().PersistBlockData(gomock.Any()).AnyTimes()
	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, executorType)
	assert.Nil(t, err)

	// mock data for block
	var txs []*pb.Transaction
	tx, err := genBVMContractTransaction(privKey, 1, contractAddr, "GetIBTPByID", pb.String(id))
	assert.Nil(t, err)

	txs = append(txs, tx)
	receipts := exec.ApplyReadonlyTransactions(txs)

	assert.Equal(t, 1, len(receipts))
	assert.Equal(t, hash.Bytes(), receipts[0].Ret)
	assert.Equal(t, pb.Receipt_SUCCESS, receipts[0].Status)
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
	block := &pb.Block{
		BlockHeader:  header,
		Transactions: txs,
	}
	block.BlockHash = block.Hash()

	return block
}

func mockTx(t *testing.T, data *pb.TransactionData) *pb.Transaction {
	var content []byte
	if data != nil {
		content, _ = data.Marshal()
	}
	return &pb.Transaction{
		To:      randAddress(t),
		Payload: content,
		Nonce:   uint64(rand.Int63()),
	}
}

func TestBlockExecutor_ExecuteBlock_Transfer(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "executor")
	require.Nil(t, err)

	blockchainStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	require.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	require.Nil(t, err)

	repo.DefaultConfig()
	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("executor_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), blockchainStorage, ldb, blockFile, accountCache, log.NewWithModule("ledger"))
	require.Nil(t, err)

	_, from := loadAdminKey(t)

	ldg.SetBalance(from, 100000000)
	account, journal := ldg.FlushDirtyDataAndComputeJournal()
	err = ldg.Commit(1, account, journal)
	require.Nil(t, err)
	err = ldg.PersistExecutionResult(mockBlock(1, nil), nil, &pb.InterchainMeta{})
	require.Nil(t, err)

	executor, err := New(ldg, log.NewWithModule("executor"), executorType)
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
	viewLedger, err := ledger.New(createMockRepo(t), blockchainStorage, ldb, blockFile, accountCache, log.NewWithModule("ledger"))
	require.Nil(t, err)

	exec, err := New(viewLedger, log.NewWithModule("executor"), executorType)
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

	transactionData := &pb.TransactionData{
		Type:   pb.TransactionData_NORMAL,
		Amount: 1,
	}

	data, err := transactionData.Marshal()
	require.Nil(t, err)

	tx := &pb.Transaction{
		From:      from,
		To:        to,
		Timestamp: time.Now().UnixNano(),
		Payload:   data,
		Amount:    1,
	}

	err = tx.Sign(privKey)
	require.Nil(t, err)
	tx.TransactionHash = tx.Hash()

	return tx
}

func loadAdminKey(t *testing.T) (crypto.PrivateKey, *types.Address) {
	privKey, err := asym.RestorePrivateKey(filepath.Join("testdata", "key.json"), keyPassword)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return privKey, from
}

func randAddress(t *testing.T) *types.Address {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)
	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return address
}

func genBVMContractTransaction(privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	return genContractTransaction(pb.TransactionData_BVM, privateKey, nonce, address, method, args...)
}

func genXVMContractTransaction(privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	return genContractTransaction(pb.TransactionData_XVM, privateKey, nonce, address, method, args...)
}

func genContractTransaction(vmType pb.TransactionData_VMType, privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (*pb.Transaction, error) {
	from, err := privateKey.PublicKey().Address()
	if err != nil {
		return nil, err
	}

	pl := &pb.InvokePayload{
		Method: method,
		Args:   args[:],
	}

	data, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	td := &pb.TransactionData{
		Type:    pb.TransactionData_INVOKE,
		VmType:  vmType,
		Payload: data,
	}

	pld, err := td.Marshal()
	if err != nil {
		return nil, err
	}

	tx := &pb.Transaction{
		From:      from,
		To:        address,
		Payload:   pld,
		Timestamp: time.Now().UnixNano(),
		Nonce:     nonce,
	}

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("tx sign: %w", err)
	}

	tx.TransactionHash = tx.Hash()

	return tx, nil
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

func createMockRepo(t *testing.T) *repo.Repo {
	key := `-----BEGIN EC PRIVATE KEY-----
BcNwjTDCxyxLNjFKQfMAc6sY6iJs+Ma59WZyC/4uhjE=
-----END EC PRIVATE KEY-----`

	privKey, err := cert.ParsePrivateKey([]byte(key), crypto.Secp256k1)
	require.Nil(t, err)

	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return &repo.Repo{
		Key: &repo.Key{
			PrivKey: privKey,
			Address: address.String(),
		},
	}
}
