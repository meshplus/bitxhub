package executor

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axiomesh/axiom-kit/hexutil"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/storage/pebble"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/executor/system/governance"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/model/events"
	"github.com/axiomesh/axiom/pkg/repo"
	ethvm "github.com/axiomesh/eth-kit/evm"
	ethledger "github.com/axiomesh/eth-kit/ledger"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
)

const (
	keyPassword = "bitxhub"
	srcMethod   = "did:axiom:addr1:."
	dstMethod   = "did:axiom:addr2:."
	from        = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

	fromServiceID  = "1:chain0:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	toServiceID    = "1:chain1:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	PREFIX         = "tx"
	TIMEOUT_PREFIX = "timeout"
	HappyRuleAddr  = "0x00000000000000000000000000000000000000a2"
	bxhID          = "1356"
)

//	func TestSign(t *testing.T) {
//		privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
//		assert.Nil(t, err)
//		pubKey := privKey.PublicKey()
//		from,err := pubKey.Address()
//		pubKeyBytes, err := ecdsa.Ecrecover(digest, sig)
//		if err != nil {
//			return false, err
//		}
//		pubkey, err := ecdsa.UnmarshalPublicKey(pubKeyBytes, opt)
//		if err != nil {
//			return false, err
//		}
//
//		expected, err := pubkey.Address()
//		if err != nil {
//			return false, err
//		}
//	}
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
	chainMeta := &types.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()

	logger := log.NewWithModule("executor")
	executor, err := New(mockLedger, logger, config)
	assert.Nil(t, err)
	assert.NotNil(t, executor)

	assert.Equal(t, mockLedger, executor.ledger)
	assert.Equal(t, logger, executor.logger)
	assert.NotNil(t, executor.blockC)
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
	chainMeta := &types.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}
	// mock block for ledger
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(mockBlock(1, nil), nil).Times(1)

	logger := log.NewWithModule("executor")
	executor, err := New(mockLedger, logger, config)
	assert.Nil(t, err)
	assert.NotNil(t, executor)

	txCtx := ethvm.TxContext{}
	evm, err := executor.GetEvm(txCtx, ethvm.Config{NoBaseFee: true})
	assert.NotNil(t, evm)
	assert.Nil(t, err)

	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(nil, errors.New("get block error")).Times(1)
	evmErr, err := executor.GetEvm(txCtx, ethvm.Config{NoBaseFee: true})
	assert.Nil(t, evmErr)
	assert.NotNil(t, err)
}

func TestSubscribeLogsEvent(t *testing.T) {
	executor := executor_start(t)
	ch := make(chan []*types.EvmLog, 10)
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
	chainMeta := &types.ChainMeta{
		Height:    1,
		GasPrice:  big.NewInt(5000),
		BlockHash: types.NewHash([]byte(from)),
	}
	block := &types.Block{
		BlockHeader: &types.BlockHeader{
			GasPrice: 5000,
		},
	}

	evs := make([]*types.Event, 0)
	m := make(map[string]uint64)
	m[from] = 3
	data, err := json.Marshal(m)
	assert.Nil(t, err)
	ev := &types.Event{
		TxHash:    types.NewHash([]byte(from)),
		Data:      data,
		EventType: types.EventOTHER,
	}
	ev2 := &types.Event{
		TxHash:    types.NewHash([]byte(from)),
		Data:      data,
		EventType: types.EventOTHER,
	}
	ev3 := &types.Event{
		TxHash:    types.NewHash([]byte(from)),
		Data:      data,
		EventType: types.EventOTHER,
	}

	evs = append(evs, ev, ev2, ev3)
	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
	stateLedger.EXPECT().QueryByPrefix(gomock.Any(), gomock.Any()).Return(false, nil).AnyTimes()
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(block, nil).AnyTimes()
	stateLedger.EXPECT().Events(gomock.Any()).Return(evs).AnyTimes()
	stateLedger.EXPECT().Commit(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().Clear().AnyTimes()
	stateLedger.EXPECT().GetEVMNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateLedger.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(3000000000000000000)).AnyTimes()
	stateLedger.EXPECT().GetEVMBalance(gomock.Any()).Return(big.NewInt(3000000000000000000)).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateLedger.EXPECT().SetCode(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetCode(gomock.Any()).Return([]byte("10")).AnyTimes()
	stateLedger.EXPECT().GetLogs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	chainLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().FlushDirtyData().Return(make(map[string]ethledger.IAccount), &types.Hash{}).AnyTimes()
	stateLedger.EXPECT().PrepareBlock(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Finalise(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Snapshot().Return(1).AnyTimes()
	stateLedger.EXPECT().RevertToSnapshot(1).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Close().AnyTimes()
	chainLedger.EXPECT().Close().AnyTimes()
	stateLedger.EXPECT().GetEVMCode(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetEVMRefund().AnyTimes()
	stateLedger.EXPECT().GetEVMCodeHash(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SubEVMBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SetEVMNonce(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().ExistEVM(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().CreateEVMAccount(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().AddEVMBalance(gomock.Any(), gomock.Any()).AnyTimes()

	stateLedger.EXPECT().SetState(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr *types.Address, key []byte, value []byte) {},
	).AnyTimes()
	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr *types.Address, key []byte) (bool, []byte) {
			return true, []byte("10")
		}).AnyTimes()
	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, config)
	assert.Nil(t, err)

	var txs []*types.Transaction
	emptyDataTx := mockTx(t)
	txs = append(txs, emptyDataTx)

	invalidTx := mockTx(t)
	invalidTx.Inner.(*types.LegacyTx).Nonce = 1000
	txs = append(txs, invalidTx)

	assert.Nil(t, exec.Start())

	ch := make(chan events.ExecutedEvent)
	blockSub := exec.SubscribeBlockEvent(ch)
	defer blockSub.Unsubscribe()

	// send blocks to executor
	commitEvent1 := mockCommitEvent(uint64(2), nil)

	commitEvent2 := mockCommitEvent(uint64(3), txs)
	exec.ExecuteBlock(commitEvent1)
	exec.ExecuteBlock(commitEvent2)

	blockRes1 := <-ch
	assert.EqualValues(t, 2, blockRes1.Block.BlockHeader.Number)
	assert.Equal(t, 0, len(blockRes1.Block.Transactions))
	assert.Equal(t, 0, len(blockRes1.TxHashList))
	blockRes2 := <-ch
	assert.EqualValues(t, 3, blockRes2.Block.BlockHeader.Number)
	assert.Equal(t, 2, len(blockRes2.Block.Transactions))
	assert.Equal(t, 2, len(blockRes2.TxHashList))

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
	chainMeta := &types.ChainMeta{
		Height:    1,
		GasPrice:  big.NewInt(5000),
		BlockHash: types.NewHashByStr(from),
	}

	id := fmt.Sprintf("%s-%s-%d", srcMethod, dstMethod, 1)

	hash := types.NewHash([]byte{1})
	val, err := json.Marshal(hash)
	assert.Nil(t, err)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	ld, err := leveldb.New(filepath.Join(repoRoot, "executor"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())

	contractAddr := types.NewAddressByStr("0xdac17f958d2ee523a2206206994597c13d831ec7")
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
	stateLedger.EXPECT().Events(gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().Commit(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().Clear().AnyTimes()
	stateLedger.EXPECT().GetState(contractAddr, []byte(fmt.Sprintf("index-tx-%s", id))).Return(true, val).AnyTimes()
	chainLedger.EXPECT().PersistExecutionResult(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	stateLedger.EXPECT().FlushDirtyData().Return(make(map[string]ethledger.IAccount), &types.Hash{}).AnyTimes()
	stateLedger.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateLedger.EXPECT().SetNonce(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Finalise(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().Snapshot().Return(1).AnyTimes()
	stateLedger.EXPECT().RevertToSnapshot(1).AnyTimes()
	stateLedger.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	chainLedger.EXPECT().LoadChainMeta().Return(chainMeta, nil).AnyTimes()
	stateLedger.EXPECT().GetLogs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(mockBlock(10, nil), nil).AnyTimes()
	stateLedger.EXPECT().PrepareEVM(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().PrepareBlock(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SetEVMNonce(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetEVMNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateLedger.EXPECT().GetBalance(gomock.Any()).Return(big.NewInt(3000000000000000000)).AnyTimes()
	stateLedger.EXPECT().GetEVMBalance(gomock.Any()).Return(big.NewInt(3000000000000000000)).AnyTimes()
	stateLedger.EXPECT().SubEVMBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().ExistEVM(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().CreateEVMAccount(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().AddEVMBalance(gomock.Any(), gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetEVMCode(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetEVMRefund().AnyTimes()
	stateLedger.EXPECT().GetEVMCodeHash(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()
	stateLedger.EXPECT().AddLog(gomock.Any()).AnyTimes()

	signer, err := types.GenerateSigner()
	assert.Nil(t, err)
	err = governance.InitCouncilMembers(stateLedger, []*repo.Admin{
		{
			Address: signer.Addr.String(),
			Weight:  1,
		},
		{
			Address: "0x1220000000000000000000000000000000000000",
			Weight:  1,
		},
		{
			Address: "0x1230000000000000000000000000000000000000",
			Weight:  1,
		},
		{
			Address: "0x1240000000000000000000000000000000000000",
			Weight:  1,
		},
	}, "1000000")
	assert.Nil(t, err)
	err = governance.InitNodeMembers(stateLedger, []*repo.Member{
		{
			NodeId: "16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
		},
	})
	assert.Nil(t, err)

	logger := log.NewWithModule("executor")

	exec, err := New(mockLedger, logger, config)
	assert.Nil(t, err)

	rawTx := "0xf86c8085147d35700082520894f927bb571eaab8c9a361ab405c9e4891c5024380880de0b6b3a76400008025a00b8e3b66c1e7ae870802e3ef75f1ec741f19501774bd5083920ce181c2140b99a0040c122b7ebfb3d33813927246cbbad1c6bf210474f5d28053990abff0fd4f53"
	tx4 := &types.Transaction{}
	tx4.Unmarshal(hexutil.Decode(rawTx))

	var txs3 []*types.Transaction
	tx5, _, err := types.GenerateTransactionAndSigner(uint64(0), types.NewAddressByStr("0xdAC17F958D2ee523a2206206994597C13D831ec7"), big.NewInt(1), nil)
	assert.Nil(t, err)
	// test system contract
	data := generateNodeAddProposeData(t, NodeExtraArgs{
		Nodes: []*NodeMember{
			{
				NodeId: "16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
			},
		},
	})
	assert.Nil(t, err)
	fmt.Println(data)
	tx6, err := types.GenerateTransactionWithSigner(uint64(1), types.NewAddressByStr(common.NodeManagerContractAddr), big.NewInt(0), data, signer)
	assert.Nil(t, err)

	txs3 = append(txs3, tx4, tx5, tx6)
	res := exec.ApplyReadonlyTransactions(txs3)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, types.ReceiptSUCCESS, res[0].Status)
	assert.Equal(t, types.ReceiptSUCCESS, res[1].Status)
	assert.Equal(t, types.ReceiptSUCCESS, res[2].Status)
}

func generateNodeAddProposeData(t *testing.T, extraArgs NodeExtraArgs) []byte {
	// test system contract
	gabi, err := governance.GetABI()
	assert.Nil(t, err)

	title := "title"
	desc := "desc"
	blockNumber := uint64(1000)

	extra, err := json.Marshal(extraArgs)
	assert.Nil(t, err)
	data, err := gabi.Pack(governance.ProposeMethod, uint8(governance.NodeUpdate), title, desc, blockNumber, extra)
	assert.Nil(t, err)
	return data
}

// NodeExtraArgs is Node proposal extra arguments
type NodeExtraArgs struct {
	Nodes []*NodeMember
}

type NodeMember struct {
	NodeId string
}

func mockCommitEvent(blockNumber uint64, txs []*types.Transaction) *types.CommitEvent {
	block := mockBlock(blockNumber, txs)
	localList := make([]bool, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		localList[i] = false
	}
	return &types.CommitEvent{
		Block:     block,
		LocalList: localList,
	}
}

func mockBlock(blockNumber uint64, txs []*types.Transaction) *types.Block {
	header := &types.BlockHeader{
		Number:    blockNumber,
		Timestamp: time.Now().Unix(),
	}

	block := &types.Block{
		BlockHeader:  header,
		Transactions: txs,
	}
	block.BlockHash = block.Hash()

	return block
}

func mockTx(t *testing.T) *types.Transaction {
	tx, _, err := types.GenerateTransactionAndSigner(uint64(0), types.NewAddressByStr("0xdAC17F958D2ee523a2206206994597C13D831ec7"), big.NewInt(1), nil)
	assert.Nil(t, err)
	return tx
}

func TestBlockExecutor_ExecuteBlock_Transfer(t *testing.T) {
	config := generateMockConfig(t)
	repoRoot := t.TempDir()

	lBlockStorage, err := leveldb.New(filepath.Join(repoRoot, "lStorage"))
	assert.Nil(t, err)
	lStateStorage, err := leveldb.New(filepath.Join(repoRoot, "lLedger"))
	assert.Nil(t, err)
	pBlockStorage, err := pebble.New(filepath.Join(repoRoot, "pStorage"))
	assert.Nil(t, err)
	pStateStorage, err := pebble.New(filepath.Join(repoRoot, "pLedger"))
	assert.Nil(t, err)

	testcase := map[string]struct {
		blockStorage storage.Storage
		stateStorage storage.Storage
	}{
		"leveldb": {blockStorage: lBlockStorage, stateStorage: lStateStorage},
		"pebble":  {blockStorage: pBlockStorage, stateStorage: pStateStorage},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			accountCache, err := ledger.NewAccountCache()
			assert.Nil(t, err)
			logger := log.NewWithModule("executor_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			ldg, err := ledger.New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, accountCache, log.NewWithModule("ledger"))
			require.Nil(t, err)

			from, err := types.GenerateSigner()
			require.Nil(t, err)
			to := types.NewAddressByStr("0xdAC17F958D2ee523a2206206994597C13D831ec7")

			ldg.SetBalance(from.Addr, new(big.Int).Mul(big.NewInt(5000000000000), big.NewInt(21000*10000)))
			account, journal := ldg.FlushDirtyData()
			err = ldg.Commit(1, account, journal)
			require.Nil(t, err)
			err = ldg.PersistExecutionResult(mockBlock(1, nil), nil)
			require.Nil(t, err)

			executor, err := New(ldg, log.NewWithModule("executor"), config)
			require.Nil(t, err)
			err = executor.Start()
			require.Nil(t, err)

			ch := make(chan events.ExecutedEvent)
			sub := executor.SubscribeBlockEvent(ch)
			defer sub.Unsubscribe()

			tx1 := mockTransferTx(t, from, to, 0, 1)
			tx2 := mockTransferTx(t, from, to, 1, 1)
			tx3 := mockTransferTx(t, from, to, 2, 1)
			commitEvent := mockCommitEvent(2, []*types.Transaction{tx1, tx2, tx3})
			executor.ExecuteBlock(commitEvent)

			block := <-ch
			require.EqualValues(t, 2, block.Block.Height())
			require.EqualValues(t, 3, ldg.GetBalance(to).Uint64())
		})
	}
}

func mockTransferTx(t *testing.T, s *types.Signer, to *types.Address, nonce, amount int) *types.Transaction {
	tx, err := types.GenerateTransactionWithSigner(uint64(nonce), to, big.NewInt(int64(amount)), nil, s)
	assert.Nil(t, err)
	return tx
}

func createMockRepo(t *testing.T) *repo.Repo {
	r, err := repo.Default(t.TempDir())
	assert.Nil(t, err)

	return r
}

func generateMockConfig(t *testing.T) *repo.Config {
	r, err := repo.Default(t.TempDir())
	assert.Nil(t, err)
	config := r.Config

	for i := 0; i < 4; i++ {
		config.Genesis.Admins = append(config.Genesis.Admins, &repo.Admin{
			Address: types.NewAddress([]byte{byte(1)}).String(),
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
	chainMeta := &types.ChainMeta{
		Height:    1,
		BlockHash: types.NewHashByStr(from),
	}
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()

	logger := log.NewWithModule("executor")
	executor, _ := New(mockLedger, logger, config)
	return executor
}
