package executor

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
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
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/executor/oracle/appchain"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	types2 "github.com/meshplus/eth-kit/types"
	types3 "github.com/meshplus/eth-kit/types"
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

const wasmGasLimit = 5000000000000000

//
// func TestSign(t *testing.T) {
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
// }

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
	executor, err := New(mockLedger, logger, &appchain.Client{}, config, big.NewInt(5000000))
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
}

func TestGetBoltContracts(t *testing.T) {
	executor := executor_start(t)
	contracts := executor.GetBoltContracts()
	assert.NotNil(t, contracts)
}

func TestSubscribeLogsEvent(t *testing.T) {
	executor := executor_start(t)
	ch := make(chan []*pb.EvmLog, 10)
	subscription := executor.SubscribeLogsEvent(ch)
	assert.NotNil(t, subscription)
}

func TestSubscribeNodeEvent(t *testing.T) {
	executor := executor_start(t)
	nodeCh := make(chan events.NodeEvent)
	subscription := executor.SubscribeNodeEvent(nodeCh)
	assert.NotNil(t, subscription)
}

func TestSubscribeAuditEvent(t *testing.T) {
	executor := executor_start(t)
	auditCh := make(chan *pb.AuditTxInfo, 1024)
	subscription := executor.SubscribeAuditEvent(auditCh)
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
	m := make(map[string]*pb.EventWrapper)
	m[from] = &pb.EventWrapper{
		IsBatch: false,
		Index:   3,
	}
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
	chainLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().FlushDirtyData().Return(make(map[string]ledger2.IAccount), &types.Hash{}).AnyTimes()
	stateLedger.EXPECT().PrepareBlock(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Finalise(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Snapshot().Return(1).AnyTimes()
	stateLedger.EXPECT().RevertToSnapshot(1).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Close().AnyTimes()
	chainLedger.EXPECT().Close().AnyTimes()

	timeListLedger := make(map[string][]byte)
	recordLedger := make(map[string][]byte)
	timeListLedger["timeout-1"] = []byte{'1', '0'}
	timeListLedger["timeout-2"] = []byte{'t', '-', '2'}
	stateLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr *types.Address, key []byte, value []byte, _ interface{}) {
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

	stateLedger.EXPECT().AddState(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr *types.Address, key []byte, value []byte) {
			if addr.String() == constant.TransactionMgrContractAddr.Address().String() {
				// manager timeout list
				if strings.HasPrefix(string(key), TIMEOUT_PREFIX) {
					timeListLedger[string(key)] = value
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

	exec, err := New(mockLedger, logger, &appchain.Client{}, config, big.NewInt(5000000))
	assert.Nil(t, err)

	// mock data for block
	var txs []*pb.BxhTransaction
	var invalidTxs []*pb.BxhTransaction
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)
	pubKey := privKey.PublicKey()

	// set tx of illegal TransactionData_BVM type
	ibtp1 := mockIBTP(t, 1, pb.IBTP_INTERCHAIN)
	BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp1)
	BVMTx := mockTx(t, BVMData)
	txs = append(txs, BVMTx)
	invalidTxs = append(invalidTxs, BVMTx)
	// set tx of illegal TransactionData_XVM type
	ibtp2 := mockIBTP(t, 2, pb.IBTP_INTERCHAIN)
	XVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_XVM, ibtp2)
	XVMTx := mockTx(t, XVMData)
	txs = append(txs, XVMTx)
	invalidTxs = append(invalidTxs, XVMTx)
	// set tx of TransactionData_NORMAL type
	ibtp3 := mockIBTP(t, 3, pb.IBTP_INTERCHAIN)
	NormalData := mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_XVM, ibtp3)
	NormalTx := mockTx(t, NormalData)
	txs = append(txs, NormalTx)
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

	// set tx of TimeoutHeight 1 for block 2
	var timeoutTxs1 []pb.Transaction
	var timeoutTxs2 []pb.Transaction
	invalidTxHashMap := make(map[string]bool)
	timeoutIbtp1 := mockIBTP1(t, 1, pb.IBTP_INTERCHAIN)
	timeoutIbtp1.TimeoutHeight = 1
	NormalData = mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_BVM, timeoutIbtp1)
	tx1 := mockTx1(t, NormalData, timeoutIbtp1)
	timeoutTxs1 = append(timeoutTxs1, tx1)

	// set tx of TimeoutHeight is max for block 2
	timeoutIbtp2 := mockIBTP1(t, 2, pb.IBTP_INTERCHAIN)
	NormalData = mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_BVM, timeoutIbtp2)
	tx2 := mockTx1(t, NormalData, timeoutIbtp2)
	timeoutTxs1 = append(timeoutTxs1, tx2)

	// set invalidTx for block 2
	invalidIbtp3 := mockIBTP1(t, 3, pb.IBTP_INTERCHAIN)
	NormalData = mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_BVM, invalidIbtp3)
	tx3 := mockTx1(t, NormalData, invalidIbtp3)
	invalidTxHashMap[tx3.GetHash().String()] = true
	timeoutTxs1 = append(timeoutTxs1, tx3)

	recordLedger = mockRecordLedger(recordLedger, timeoutTxs1, 2)

	err = exec.setTimeoutList(2, timeoutTxs1, invalidTxHashMap, nil, bxhID)
	assert.Nil(t, err)
	txId1 := "1:chain0:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997-1:chain1:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997-1"
	txId2 := "1:chain0:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997-1:chain1:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997-2"
	txId3 := "1:chain0:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997-1:chain1:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997-3"
	val1 := recordLedger[contracts.TxInfoKey(txId1)]
	val2 := recordLedger[contracts.TxInfoKey(txId2)]
	val3 := recordLedger[contracts.TxInfoKey(txId3)]
	var record1 pb.TransactionRecord
	var record2 pb.TransactionRecord
	var record3 pb.TransactionRecord
	err = record1.Unmarshal(val1)
	assert.Nil(t, err)
	err = record2.Unmarshal(val2)
	assert.Nil(t, err)
	err = record3.Unmarshal(val3)
	assert.Nil(t, err)
	assert.Equal(t, record1.Height, uint64(3))
	assert.Equal(t, record1.Status, pb.TransactionStatus_BEGIN)
	val, ok := timeListLedger["timeout-3"]
	assert.True(t, ok)
	assert.NotNil(t, val)

	// the receipt of tx1
	receipt1 := mockIBTP1(t, 1, pb.IBTP_RECEIPT_SUCCESS)
	NormalData = mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_BVM, receipt1)
	tx4 := mockTx1(t, NormalData, receipt1)
	timeoutTxs2 = append(timeoutTxs2, tx4)
	recordLedger = mockRecordLedger(recordLedger, timeoutTxs2, 3)
	err = exec.setTimeoutList(3, timeoutTxs2, invalidTxHashMap, nil, bxhID)
	assert.Nil(t, err)
	val, ok = timeListLedger["timeout-3"]
	assert.True(t, ok)
	assert.Equal(t, []byte{}, val)

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

	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)

	assert.Nil(t, err)

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
	chainLedger.EXPECT().LoadChainMeta().Return(chainMeta).AnyTimes()
	stateLedger.EXPECT().GetLogs(gomock.Any()).Return(nil).AnyTimes()
	chainLedger.EXPECT().GetBlock(gomock.Any(), gomock.Any()).Return(mockBlock(10, nil), nil).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().PrepareBlock(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(10000000000000)).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()
	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, &appchain.Client{}, config, big.NewInt(5000000))
	assert.Nil(t, err)

	// mock data for block
	var txs []pb.Transaction
	tx, err := genBVMContractTransaction(privKey, 1, contractAddr, "GetIBTPByID", pb.String(id), pb.Bool(true))
	assert.Nil(t, err)

	invalidIbtp := mockIBTP1(t, 0, pb.IBTP_INTERCHAIN)
	NormalData := mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_BVM, invalidIbtp)
	tx2 := mockTx1(t, NormalData, invalidIbtp)
	txs = append(txs, tx, tx2)
	receipts := exec.ApplyReadonlyTransactions(txs)
	assert.Equal(t, 2, len(receipts))
	assert.Equal(t, hash.Bytes(), receipts[0].Ret)
	assert.Equal(t, pb.Receipt_SUCCESS, receipts[0].Status)

	var txs2 []pb.Transaction
	privKey2, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	tx3, _ := genBVMContractTransaction(privKey2, 1, contractAddr, "GetIBTPByID", pb.String(id), pb.Bool(true))
	txs2 = append(txs2, tx3)
	exec.bxhGasPrice = big.NewInt(100000000000000)
	receipts = exec.ApplyReadonlyTransactions(txs2)
	assert.Equal(t, 1, len(receipts))
	assert.Equal(t, pb.Receipt_FAILED, receipts[0].Status)

	/*var data []byte
	addr := "0x72c445Bc9285ff4275Eda950Fb2e17389935F3D9"
	ethAddr := common.BytesToAddress([]byte(addr))*/

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
	var txs3 []pb.Transaction
	txs3 = append(txs3, tx4)
	types2.InitEIP155Signer(big.NewInt(1))
	exec.ApplyReadonlyTransactions(txs3)

}

func TestCalcTimeoutL2Root(t *testing.T) {
	exec := executor_start(t)
	hashres, err := exec.calcTimeoutL2Root([]string{"1", "2", "3"})
	require.Nil(t, err)
	require.NotNil(t, hashres)
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

func mockTx1(t *testing.T, data *pb.TransactionData, ibtp *pb.IBTP) *pb.BxhTransaction {
	var content []byte
	if data != nil {
		content, _ = data.Marshal()
	}
	to := &types.Address{
		Address: "0x000000000000000000000000000000000000000f",
	}
	return &pb.BxhTransaction{
		To:      to,
		Payload: content,
		Nonce:   uint64(rand.Int63()),
		IBTP:    ibtp,
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

	ldg.SetState(constant.InterchainContractAddr.Address(), []byte(contracts.BitXHubID), []byte("1"), nil)

	executor, err := New(ldg, log.NewWithModule("executor"), &appchain.Client{}, config, big.NewInt(5000000))
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
	require.EqualValues(t, 1, ldg.GetBalance(from).Uint64())

	// test executor with readonly ledger
	viewLedger, err := ledger.New(createMockRepo(t), blockchainStorage, ldb, blockFile, accountCache, log.NewWithModule("ledger"))
	require.Nil(t, err)

	exec, err := New(viewLedger, log.NewWithModule("executor"), &appchain.Client{}, config, big.NewInt(0))
	require.Nil(t, err)

	tx := mockTransferTx(t)
	receipts := exec.ApplyReadonlyTransactions([]pb.Transaction{tx})
	require.NotNil(t, receipts)
	require.Equal(t, pb.Receipt_SUCCESS, receipts[0].Status)
	require.Nil(t, receipts[0].Ret)
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
		Timestamp: time.Now().UnixNano(),
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

func genBVMContractTransaction(privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (pb.Transaction, error) {
	return genContractTransaction(pb.TransactionData_BVM, privateKey, nonce, address, method, args...)
}

func genXVMContractTransaction(privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (pb.Transaction, error) {
	return genContractTransaction(pb.TransactionData_XVM, privateKey, nonce, address, method, args...)
}

func genEthTransaction(nonce uint64, address *common.Address, value *big.Int, data []byte) (pb.Transaction, error) {
	t := time.Now()
	txData := &types3.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(10000),
		Gas:      uint64(1000000),
		To:       address,
		Value:    value,
		Data:     data,
	}
	hash := types2.RlpHash([]interface{}{
		txData.GetNonce(),
		txData.GetGasPrice(),
		txData.GetGas(),
		txData.GetTo(),
		txData.GetValue(),
		txData.GetData(),
	})
	privkey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	signature, _ := asym.SignWithType(privkey, hash.Bytes())

	r := big.NewInt(int64(binary.LittleEndian.Uint32(signature[0:32])))
	s := big.NewInt(int64(binary.LittleEndian.Uint32(signature[32:64])))
	v := big.NewInt(int64(signature[64]))
	txData.R = r
	txData.S = s
	txData.V = v
	tx := &types2.EthTransaction{
		Inner: txData,
		Time:  t,
	}

	return tx, nil
}

func genContractTransaction(vmType pb.TransactionData_VMType, privateKey crypto.PrivateKey, nonce uint64, address *types.Address, method string, args ...*pb.Arg) (pb.Transaction, error) {
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

	tx := &pb.BxhTransaction{
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
		Amount:  "10",
		Payload: pd,
	}
}

func mockIBTP(t *testing.T, index uint64, typ pb.IBTP_Type) *pb.IBTP {
	content := pb.Content{
		Func: "set",
	}

	bytes, err := content.Marshal()
	assert.Nil(t, err)

	payload := pb.Payload{
		Encrypted: false,
		Content:   bytes,
	}
	ibtppd, err := payload.Marshal()
	assert.Nil(t, err)

	return &pb.IBTP{
		From:    from,
		To:      from,
		Payload: ibtppd,
		Index:   index,
		Type:    typ,
	}
}

func mockIBTP1(t *testing.T, index uint64, typ pb.IBTP_Type) *pb.IBTP {
	content := pb.Content{
		Func: "set",
	}

	bytes, err := content.Marshal()
	assert.Nil(t, err)

	payload := pb.Payload{
		Encrypted: false,
		Content:   bytes,
	}
	ibtppd, err := payload.Marshal()
	assert.Nil(t, err)

	proof := []byte("1")
	proofHash := sha256.Sum256(proof)
	return &pb.IBTP{
		From:    fromServiceID,
		To:      toServiceID,
		Payload: ibtppd,
		Index:   index,
		Type:    typ,
		Proof:   proofHash[:],
	}
}

func createMockRepo(t *testing.T) *repo.Repo {
	key := `-----BEGIN EC PRIVATE KEY-----
BcNwjTDCxyxLNjFKQfMAc6sY6iJs+Ma59WZyC/4uhjE=
-----END EC PRIVATE KEY-----`

	privKey, err := libp2pcert.ParsePrivateKey([]byte(key), crypto.Secp256k1)
	require.Nil(t, err)

	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	rep := &repo.Repo{
		Key: &repo.Key{
			PrivKey: privKey,
			Address: address.String(),
		},
		Config: &repo.Config{},
	}
	rep.Config.Executor.Type = "serial"

	return rep
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

func mockRecordLedger(ledger map[string][]byte, txList []pb.Transaction, height uint64) map[string][]byte {
	var timeoutHeight uint64
	for _, tx := range txList {
		switch tx.(type) {
		case *pb.BxhTransaction:
			if !tx.IsIBTP() {
				continue
			}
			ibtp := tx.GetIBTP()
			if ibtp.Category() == pb.IBTP_RESPONSE {
				continue
			}
			if ibtp.TimeoutHeight == 0 {
				timeoutHeight = math.MaxUint64 - height
			} else {
				timeoutHeight = uint64(ibtp.GetTimeoutHeight())
			}

			txId := fmt.Sprintf("%s-%s-%d", ibtp.From, ibtp.To, ibtp.Index)
			var record = pb.TransactionRecord{
				Height: height + timeoutHeight,
				Status: pb.TransactionStatus_BEGIN,
			}
			status, _ := record.Marshal()
			ledger[contracts.TxInfoKey(txId)] = status
		}
	}
	return ledger
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
	executor, _ := New(mockLedger, logger, &appchain.Client{}, config, big.NewInt(5000000))
	return executor
}

func TestRollback(t *testing.T) {
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
	m := make(map[string]*pb.EventWrapper)
	m[from] = &pb.EventWrapper{Index: 3}
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
	chainLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().FlushDirtyData().Return(make(map[string]ledger2.IAccount), &types.Hash{}).AnyTimes()
	stateLedger.EXPECT().PrepareBlock(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Finalise(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Snapshot().Return(1).AnyTimes()
	stateLedger.EXPECT().RevertToSnapshot(1).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Close().AnyTimes()
	chainLedger.EXPECT().Close().AnyTimes()
	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	stateLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, &appchain.Client{}, config, big.NewInt(5000000))
	assert.Nil(t, err)

	// mock data for block
	var txs1 []*pb.BxhTransaction
	var txs2 []*pb.BxhTransaction
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	assert.Nil(t, err)
	pubKey := privKey.PublicKey()

	for i := 1; i <= 10; i++ {
		// set tx of illegal TransactionData_BVM type
		ibtp := mockIBTP(t, uint64(i), pb.IBTP_INTERCHAIN)
		BVMData := mockTxData(t, pb.TransactionData_INVOKE, pb.TransactionData_BVM, ibtp)
		tx := mockTx(t, BVMData)
		txs1 = append(txs1, tx)
	}

	for i := 1; i <= 10; i++ {
		// set tx of illegal TransactionData_BVM type
		ibtp := mockIBTP(t, uint64(i), pb.IBTP_INTERCHAIN)
		BVMData := mockTxData(t, pb.TransactionData_NORMAL, pb.TransactionData_BVM, ibtp)
		tx := mockTx(t, BVMData)
		txs2 = append(txs2, tx)
	}

	// set signature for txs
	txs1, err = signForTx(txs1, privKey, pubKey)
	assert.Nil(t, err)
	txs2, err = signForTx(txs2, privKey, pubKey)
	assert.Nil(t, err)

	assert.Nil(t, exec.Start())

	done := make(chan bool)
	ch := make(chan events.ExecutedEvent)
	blockSub := exec.SubscribeBlockEvent(ch)
	defer blockSub.Unsubscribe()

	// count received block to end test
	var wg sync.WaitGroup
	wg.Add(3)
	go listenBlock(&wg, done, ch)

	// send blocks to executor
	commitEvent1 := mockCommitEvent(uint64(2), nil)

	transactions1 := make([]pb.Transaction, 0)
	transactions2 := make([]pb.Transaction, 0)
	for _, tx := range txs1 {
		transactions1 = append(transactions1, tx)
	}
	for _, tx := range txs2 {
		transactions2 = append(transactions2, tx)
	}

	commitEvent2 := mockCommitEvent(uint64(3), transactions1)
	commitEvent3 := mockCommitEvent(uint64(3), transactions2)

	chainLedger.EXPECT().GetBlock(uint64(3), gomock.Any()).Return(commitEvent2.Block, nil).Times(1)
	chainLedger.EXPECT().GetBlock(uint64(2), gomock.Any()).Return(commitEvent1.Block, nil).Times(1)
	stateLedger.EXPECT().RollbackState(uint64(2)).Return(nil).Times(1)
	chainLedger.EXPECT().RollbackBlockChain(uint64(2)).Return(nil).Times(1)

	exec.ExecuteBlock(commitEvent1)
	exec.ExecuteBlock(commitEvent2)
	exec.ExecuteBlock(commitEvent3)

	wg.Wait()
	done <- true
	assert.Nil(t, exec.Stop())
	assert.Equal(t, exec.currentHeight, uint64(3))
}

func signForTx(txs []*pb.BxhTransaction, privKey crypto.PrivateKey, pubKey crypto.PublicKey) ([]*pb.BxhTransaction, error) {
	var err error
	for _, tx := range txs {
		tx.From, err = pubKey.Address()
		if err != nil {
			return nil, err
		}
		body, err := tx.Marshal()
		if err != nil {
			return nil, err
		}
		ret := sha256.Sum256(body)

		sig, err := asym.SignWithType(privKey, types.NewHash(ret[:]).Bytes())
		if err != nil {
			return nil, err
		}
		tx.Signature = sig
		tx.TransactionHash = tx.Hash()
	}
	return txs, nil
}
