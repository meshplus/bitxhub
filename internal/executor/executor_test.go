package executor

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/hexutil"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	vm1 "github.com/meshplus/eth-kit/evm"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	types2 "github.com/meshplus/eth-kit/types"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyPassword = "bitxhub"
	srcMethod   = "did:bitxhub:appchain1:."
	dstMethod   = "did:bitxhub:appchain2:."
	from        = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

	fromServiceID  = "1:chain0:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	toServiceID    = "1:chain1:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	PREFIX         = "tx"
	TIMEOUT_PREFIX = "timeout"
	HappyRuleAddr  = "0x00000000000000000000000000000000000000a2"
	bxhID          = "1356"
)

//
//func TestSign(t *testing.T) {
//	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
//	assert.Nil(t, err)
//	pubKey := privKey.PublicKey()
//	from,err := pubKey.Address()
//	pubKeyBytes, err := ecdsa.Ecrecover(digest, sig)
//	if err != nil {
//		return false, err
//	}
//	pubkey, err := ecdsa.UnmarshalPublicKey(pubKeyBytes, opt)
//	if err != nil {
//		return false, err
//	}
//
//	expected, err := pubkey.Address()
//	if err != nil {
//		return false, err
//	}
//}

func TestNew(t *testing.T) {
	config := generateMockConfig(t)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()

	logger := log.NewWithModule("executor")
	executor, err := New(mockLedger, logger, config, big.NewInt(5000000))
	assert.Nil(t, err)
	assert.NotNil(t, executor)

	assert.Equal(t, mockLedger, executor.ledger)
	assert.Equal(t, logger, executor.logger)
	assert.NotNil(t, executor.preBlockC)
	assert.NotNil(t, executor.blockC)
	assert.NotNil(t, executor.persistC)
	assert.Equal(t, chainMeta.BlockHash, executor.currentBlockHash)
	assert.Equal(t, chainMeta.Height, executor.currentHeight)
}

func TestGetEvm(t *testing.T) {
	config := generateMockConfig(t)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}
	// mock block for ledger
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(mockBlock(1, nil), nil).Times(1)

	logger := log.NewWithModule("executor")
	executor, err := New(mockLedger, logger, config, big.NewInt(5000000))
	assert.Nil(t, err)
	assert.NotNil(t, executor)

	txCtx := vm1.TxContext{}
	evm := executor.GetEvm(txCtx, vm1.Config{NoBaseFee: true})
	assert.NotNil(t, evm)

	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(nil, fmt.Errorf("get block error")).Times(1)
	evmErr := executor.GetEvm(txCtx, vm1.Config{NoBaseFee: true})
	assert.Nil(t, evmErr)

}

func TestSubscribeLogsEvent(t *testing.T) {
	executor := executor_start(t)
	ch := make(chan []*pb.EvmLog, 10)
	subscription := executor.SubscribeLogsEvent(ch)
	assert.NotNil(t, subscription)
}

func TestBlockExecutor_ExecuteBlock(t *testing.T) {
	config := generateMockConfig(t)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

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
		TxHash:    types.NewHash([]byte(from)),
		Data:      data,
		EventType: pb.Event_INTERCHAIN,
	}
	ev2 := &pb.Event{
		TxHash:    types.NewHash([]byte(from)),
		Data:      data,
		EventType: pb.Event_NODEMGR,
	}
	ev3 := &pb.Event{
		TxHash:    types.NewHash([]byte(from)),
		Data:      data,
		EventType: pb.Event_AUDIT_APPCHAIN,
	}

	evs = append(evs, ev, ev2, ev3)
	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
	stateLedger.EXPECT().QueryByPrefix(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	stateLedger.EXPECT().Events(gomock.Any()).Return(evs).AnyTimes()
	stateLedger.EXPECT().Commit(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().Clear().AnyTimes()
	stateLedger.EXPECT().GetBalance(gomock.Any()).Return(new(big.Int).SetUint64(1050000000000)).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateLedger.EXPECT().SetCode(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetCode(gomock.Any()).Return([]byte("10")).AnyTimes()
	stateLedger.EXPECT().GetLogs(gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	chainLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().FlushDirtyData().Return(make(map[string]ledger2.IAccount), &types.Hash{}).AnyTimes()
	stateLedger.EXPECT().PrepareBlock(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Finalise(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Snapshot().Return(1).AnyTimes()
	stateLedger.EXPECT().RevertToSnapshot(1).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Close().AnyTimes()
	chainLedger.EXPECT().Close().AnyTimes()

	timeListLedger := make(map[string][]byte)
	recordLedger := make(map[string][]byte)
	timeListLedger["timeout-1"] = []byte{'1', '0'}
	timeListLedger["timeout-2"] = []byte{'t', '-', '2'}
	stateLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr *types.Address, key []byte, value []byte) {

			if addr.String() == constant.TransactionMgrContractAddr.Address().String() {
				// manager timeout list
				if strings.HasPrefix(string(key), TIMEOUT_PREFIX) {
					timeListLedger[string(key)] = value
				}
				if strings.HasPrefix(string(key), PREFIX) {
					recordLedger[string(key)] = value
				}
			}
		}).AnyTimes()
	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr *types.Address, key []byte) (bool, []byte) {
			if addr.String() == constant.TransactionMgrContractAddr.Address().String() {
				if strings.HasPrefix(string(key), TIMEOUT_PREFIX) {
					return true, timeListLedger[string(key)]
				}
				if strings.HasPrefix(string(key), PREFIX) {
					return true, recordLedger[string(key)]
				}
				return false, nil

			} else if addr.String() == constant.InterchainContractAddr.Address().String() {
				if string(key) == "bitxhub-id" {
					return true, []byte("1")
				}
				return false, nil
			}

			return true, []byte("10")
		}).AnyTimes()
	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, config, big.NewInt(5000000))
	assert.Nil(t, err)

	// mock data for block
	var txs []*pb.BxhTransaction
	var invalidTxs []*pb.BxhTransaction
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)
	pubKey := privKey.PublicKey()

	// set tx with empty transaction data
	emptyDataTx := mockTx(t, nil)
	txs = append(txs, emptyDataTx)
	invalidTxs = append(invalidTxs, emptyDataTx)

	// set signature for txs
	for _, tx := range txs {
		tx.From, err = pubKey.Address()
		assert.Nil(t, err)
		body, err := tx.Marshal()
		assert.Nil(t, err)
		ret := sha256.Sum256(body)

		sig, err := asym.SignWithType(privKey, types.NewHash(ret[:]).Bytes())
		assert.Nil(t, err)
		tx.Signature = sig
		tx.TransactionHash = tx.Hash()
	}
	// set invalid signature tx
	invalidTx := mockTx(t, nil)
	invalidTx.From = types.NewAddressByStr(from)
	invalidTx.Signature = []byte("invalid")
	invalidTx.TransactionHash = invalidTx.Hash()
	txs = append(txs, invalidTx)

	assert.Nil(t, exec.Start())

	done := make(chan bool)
	ch := make(chan events.ExecutedEvent)
	blockSub := exec.SubscribeBlockEvent(ch)
	defer blockSub.Unsubscribe()

	// count received block to end test
	var wg sync.WaitGroup
	wg.Add(2)
	go listenBlock(&wg, done, ch)

	// send blocks to executor
	commitEvent1 := mockCommitEvent(uint64(2), nil)

	transactions := make([]pb.Transaction, 0)
	for _, tx := range txs {
		transactions = append(transactions, tx)
	}

	commitEvent2 := mockCommitEvent(uint64(3), transactions)
	exec.ExecuteBlock(commitEvent1)
	exec.ExecuteBlock(commitEvent2)

	wg.Wait()
	done <- true
	assert.Nil(t, exec.Stop())

}

func TestBlockExecutor_ApplyReadonlyTransactions(t *testing.T) {
	config := generateMockConfig(t)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}

	id := fmt.Sprintf("%s-%s-%d", srcMethod, dstMethod, 1)

	hash := types.NewHash([]byte{1})
	val, err := json.Marshal(hash)
	assert.Nil(t, err)

	contractAddr := constant.InterchainContractAddr.Address()
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	stateLedger.EXPECT().Events(gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().Commit(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().Clear().AnyTimes()
	stateLedger.EXPECT().GetState(constant.TransactionMgrContractAddr.Address(), gomock.Any()).Return(false, nil).AnyTimes()
	stateLedger.EXPECT().GetState(contractAddr, []byte(fmt.Sprintf("index-tx-%s", id))).Return(true, val).AnyTimes()
	chainLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().FlushDirtyData().Return(make(map[string]ledger2.IAccount), &types.Hash{}).AnyTimes()
	stateLedger.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateLedger.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Finalise(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Snapshot().Return(1).AnyTimes()
	stateLedger.EXPECT().RevertToSnapshot(1).AnyTimes()
	stateLedger.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	chainLedger.EXPECT().LoadChainMeta().Return(chainMeta).AnyTimes()
	stateLedger.EXPECT().GetLogs(gomock.Any()).Return(nil).AnyTimes()
	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(mockBlock(10, nil), nil).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().PrepareBlock(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(10000000000000)).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()
	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, config, big.NewInt(5000000))
	assert.Nil(t, err)

	rawTx := "0xf86c8085147d35700082520894f927bb571eaab8c9a361ab405c9e4891c5024380880de0b6b3a76400008025a00b8e3b66c1e7ae870802e3ef75f1ec741f19501774bd5083920ce181c2140b99a0040c122b7ebfb3d33813927246cbbad1c6bf210474f5d28053990abff0fd4f53"
	tx4 := &types2.EthTransaction{}
	tx4.Unmarshal(hexutil.Decode(rawTx))

	stateLedger.EXPECT().GetEVMNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateLedger.EXPECT().GetEVMBalance(gomock.Any()).Return(big.NewInt(1000000000000000000)).AnyTimes()
	stateLedger.EXPECT().SubEVMBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().ExistEVM(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().CreateEVMAccount(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().AddEVMBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetEVMCode(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetEVMRefund().AnyTimes()
	stateLedger.EXPECT().GetEVMCodeHash(gomock.Any()).AnyTimes()
	var txs3 []pb.Transaction
	txs3 = append(txs3, tx4)
	types2.InitEIP155Signer(big.NewInt(1))
	exec.ApplyReadonlyTransactions(txs3)

}

func listenBlock(wg *sync.WaitGroup, done chan bool, blockCh chan events.ExecutedEvent) {
	for {
		select {
		case <-blockCh:
			wg.Done()
		case <-done:
			return
		}
	}
}

func mockCommitEvent(blockNumber uint64, txs []pb.Transaction) *pb.CommitEvent {
	block := mockBlock(blockNumber, txs)
	localList := make([]bool, len(block.Transactions.Transactions))
	for i := 0; i < len(block.Transactions.Transactions); i++ {
		localList[i] = false
	}
	return &pb.CommitEvent{
		Block:     block,
		LocalList: localList,
	}
}

func mockBlock(blockNumber uint64, txs []pb.Transaction) *pb.Block {
	header := &pb.BlockHeader{
		Number:    blockNumber,
		Timestamp: time.Now().Unix(),
	}

	block := &pb.Block{
		BlockHeader:  header,
		Transactions: &pb.Transactions{Transactions: txs},
	}
	block.BlockHash = block.Hash()

	return block
}

func mockTx(t *testing.T, data *pb.TransactionData) *pb.BxhTransaction {
	var content []byte
	if data != nil {
		content, _ = data.Marshal()
	}
	return &pb.BxhTransaction{
		To:      randAddress(t),
		Payload: content,
		Nonce:   uint64(rand.Int63()),
	}
}

func TestBlockExecutor_ExecuteBlock_Transfer(t *testing.T) {
	config := generateMockConfig(t)
	repoRoot, err := ioutil.TempDir("", "executor")
	require.Nil(t, err)

	blockchainStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	require.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	require.Nil(t, err)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("executor_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), blockchainStorage, ldb, blockFile, accountCache, log.NewWithModule("ledger"))
	require.Nil(t, err)

	_, from := loadAdminKey(t)

	ldg.SetBalance(from, new(big.Int).SetInt64(21000*5000000*3+4))
	account, journal := ldg.FlushDirtyData()
	err = ldg.Commit(1, account, journal)
	require.Nil(t, err)
	err = ldg.PersistExecutionResult(mockBlock(1, nil), nil, &pb.InterchainMeta{})
	require.Nil(t, err)

	executor, err := New(ldg, log.NewWithModule("executor"), config, big.NewInt(5000000))
	require.Nil(t, err)
	err = executor.Start()
	require.Nil(t, err)

	ch := make(chan events.ExecutedEvent)
	sub := executor.SubscribeBlockEvent(ch)
	defer sub.Unsubscribe()

	var txs []pb.Transaction
	txs = append(txs, mockTransferTx(t))
	txs = append(txs, mockTransferTx(t))
	txs = append(txs, mockTransferTx(t))
	commitEvent := mockCommitEvent(2, txs)
	executor.ExecuteBlock(commitEvent)
	require.Nil(t, err)

	block := <-ch
	require.EqualValues(t, 2, block.Block.Height())
	require.EqualValues(t, 4, ldg.GetBalance(from).Uint64())

	// test executor with readonly ledger
	viewLedger, err := ledger.New(createMockRepo(t), blockchainStorage, ldb, blockFile, accountCache, log.NewWithModule("ledger"))
	require.Nil(t, err)

	exec, err := New(viewLedger, log.NewWithModule("executor"), config, big.NewInt(0))
	require.Nil(t, err)

	tx := mockTransferTx(t)
	receipts := exec.ApplyReadonlyTransactions([]pb.Transaction{tx})
	require.NotNil(t, receipts)
	require.Equal(t, pb.Receipt_FAILED, receipts[0].Status)
}

func mockTransferTx(t *testing.T) pb.Transaction {
	privKey, from := loadAdminKey(t)
	to := randAddress(t)

	transactionData := &pb.TransactionData{
		Type:   pb.TransactionData_NORMAL,
		Amount: "1",
	}

	data, err := transactionData.Marshal()
	require.Nil(t, err)

	tx := &pb.BxhTransaction{
		From:      from,
		To:        to,
		Timestamp: time.Now().Unix(),
		Payload:   data,
		Amount:    "1",
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

func createMockRepo(t *testing.T) *repo.Repo {
	key := `-----BEGIN EC PRIVATE KEY-----
BcNwjTDCxyxLNjFKQfMAc6sY6iJs+Ma59WZyC/4uhjE=
-----END EC PRIVATE KEY-----`

	privKey, err := libp2pcert.ParsePrivateKey([]byte(key), crypto.Secp256k1)
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

func generateMockConfig(t *testing.T) *repo.Config {
	config, err := repo.DefaultConfig()
	assert.Nil(t, err)

	for i := 0; i < 4; i++ {
		config.Admins = append(config.Admins, &repo.Admin{
			Address: types.NewAddress([]byte{byte(1)}).String(),
			Weight:  2,
		})
	}

	return config
}

func executor_start(t *testing.T) *BlockExecutor {
	config := generateMockConfig(t)
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	// mock data for ledger
	chainMeta := &pb.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()

	logger := log.NewWithModule("executor")
	executor, _ := New(mockLedger, logger, config, big.NewInt(5000000))
	return executor
}
