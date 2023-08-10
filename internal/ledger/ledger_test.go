package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	etherTypes "github.com/ethereum/go-ethereum/core/types"
	crypto1 "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/storage/pebble"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/pkg/repo"
	"github.com/axiomesh/eth-kit/ledger"
)

func TestNew001(t *testing.T) {
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
			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			l, err := New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, nil, log.NewWithModule("executor"))
			require.Nil(t, err)
			require.NotNil(t, l)
		})
	}
}

func TestNew002(t *testing.T) {
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
			tc.blockStorage.Put([]byte(chainMetaKey), []byte{1})
			accountCache, err := NewAccountCache()
			assert.Nil(t, err)
			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			l, err := New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, accountCache, log.NewWithModule("executor"))
			require.NotNil(t, err)
			require.Nil(t, l)
		})
	}
}

func TestNew003(t *testing.T) {
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
			tc.stateStorage.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(1))
			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			l, err := New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, nil, log.NewWithModule("executor"))
			require.NotNil(t, err)
			require.Nil(t, l)
		})
	}
}

func TestNew004(t *testing.T) {
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
			kvdb := tc.stateStorage
			kvdb.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(1))

			journal := &BlockJournal{}
			data, err := json.Marshal(journal)
			assert.Nil(t, err)

			kvdb.Put(compositeKey(journalKey, 1), data)

			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			l, err := New(createMockRepo(t), tc.blockStorage, kvdb, blockFile, nil, log.NewWithModule("executor"))
			require.Nil(t, err)
			require.NotNil(t, l)
		})
	}
}

func TestNew005(t *testing.T) {
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
			kvdb := tc.stateStorage
			kvdb.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(5))
			kvdb.Put(compositeKey(journalKey, minHeightStr), marshalHeight(3))

			journal := &BlockJournal{}
			data, err := json.Marshal(journal)
			assert.Nil(t, err)

			kvdb.Put(compositeKey(journalKey, 5), data)

			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			l, err := New(createMockRepo(t), tc.blockStorage, kvdb, blockFile, nil, log.NewWithModule("executor"))
			require.NotNil(t, err)
			require.Nil(t, l)
		})
	}
}

func Test_KV_Compatibility(t *testing.T) {
	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			testChainLedger_PersistBlockData(t, tc.kvType)
			testChainLedger_Commit(t, tc.kvType)
			testChainLedger_EVMAccessor(t, tc.kvType)
			testChainLedger_Rollback(t, tc.kvType)
			testChainLedger_QueryByPrefix(t, tc.kvType)
			testChainLedger_GetAccount(t, tc.kvType)
			testChainLedger_GetCode(t, tc.kvType)
			testChainLedger_AddAccountsToCache(t, tc.kvType)
			testChainLedger_AddState(t, tc.kvType)
			testChainLedger_AddEvent(t, tc.kvType)
			testGetBlockSign(t, tc.kvType)
			testGetBlockByHash(t, tc.kvType)
			testGetTransaction(t, tc.kvType)
			testGetTransaction1(t, tc.kvType)
			testGetTransactionMeta(t, tc.kvType)
			testGetReceipt(t, tc.kvType)
			testGetReceipt1(t, tc.kvType)
			testPrepare(t, tc.kvType)
		})
	}
}

func testChainLedger_PersistBlockData(t *testing.T, kv string) {
	ledger, _ := initLedger(t, "", kv)

	// create an account
	account := types.NewAddress(LeftPadBytes([]byte{100}, 20))

	ledger.SetState(account, []byte("a"), []byte("b"))
	accounts, journal := ledger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(1, accounts, journal))
}

func testChainLedger_Commit(t *testing.T, kv string) {
	lg, repoRoot := initLedger(t, "", kv)

	// create an account
	account := types.NewAddress(LeftPadBytes([]byte{100}, 20))

	lg.SetState(account, []byte("a"), []byte("b"))
	accounts, stateRoot := lg.FlushDirtyData()
	err := lg.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)
	lg.StateLedger.(*StateLedger).GetCommittedState(account, []byte("a"))
	isSuicide := lg.StateLedger.(*StateLedger).HasSuiside(account)
	assert.Equal(t, isSuicide, false)
	assert.Equal(t, uint64(1), lg.Version())
	assert.Equal(t, "0xa1a6d35708fa6cf804b6cf9479f3a55d9a87fbfb83c55a64685aeabdba6116b1", stateRoot.String())

	accounts, stateRoot = lg.FlushDirtyData()
	err = lg.Commit(2, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), lg.Version())
	assert.Equal(t, "0xf09f0198c06d549316d4ee7c497c9eaef9d24f5b1075e7bcef3d0a82dfa742cf", stateRoot.String())

	lg.SetState(account, []byte("a"), []byte("3"))
	lg.SetState(account, []byte("a"), []byte("2"))
	accounts, stateRoot = lg.FlushDirtyData()
	err = lg.Commit(3, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(3), lg.Version())
	assert.Equal(t, "0xe9fc370dd36c9bd5f67ccfbc031c909f53a3d8bc7084c01362c55f2d42ba841c", stateRoot.String())

	lg.SetBalance(account, new(big.Int).SetInt64(100))
	accounts, stateRoot = lg.FlushDirtyData()
	err = lg.Commit(4, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), lg.Version())
	assert.Equal(t, "0xc179056204ba33ed6cfc0bfe94ca03319beb522fd7b0773a589899817b49ec08", stateRoot.String())

	code := RightPadBytes([]byte{100}, 100)
	lg.SetCode(account, code)
	lg.SetState(account, []byte("b"), []byte("3"))
	lg.SetState(account, []byte("c"), []byte("2"))
	accounts, stateRoot = lg.FlushDirtyData()
	err = lg.Commit(5, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), lg.Version())
	// assert.Equal(t, uint64(5), ledger.maxJnlHeight)

	stateLedger := lg.StateLedger.(*StateLedger)
	minHeight, maxHeight := getJournalRange(stateLedger.ldb)
	journal5 := getBlockJournal(maxHeight, stateLedger.ldb)
	assert.Equal(t, uint64(1), minHeight)
	assert.Equal(t, uint64(5), maxHeight)
	assert.Equal(t, stateRoot.String(), journal5.ChangedHash.String())
	assert.Equal(t, 1, len(journal5.Journals))
	entry := journal5.Journals[0]
	assert.Equal(t, account.String(), entry.Address.String())
	assert.True(t, entry.AccountChanged)
	assert.Equal(t, uint64(100), entry.PrevAccount.Balance.Uint64())
	assert.Equal(t, uint64(0), entry.PrevAccount.Nonce)
	assert.Nil(t, entry.PrevAccount.CodeHash)
	assert.Equal(t, 2, len(entry.PrevStates))
	assert.Nil(t, entry.PrevStates[hex.EncodeToString([]byte("b"))])
	assert.Nil(t, entry.PrevStates[hex.EncodeToString([]byte("c"))])
	assert.True(t, entry.CodeChanged)
	assert.Nil(t, entry.PrevCode)
	isExist := lg.StateLedger.(*StateLedger).Exist(account)
	assert.True(t, isExist)
	isEmpty := lg.StateLedger.(*StateLedger).Empty(account)
	assert.False(t, isEmpty)
	lg.StateLedger.(*StateLedger).AccountCache()
	err = lg.StateLedger.(*StateLedger).removeJournalsBeforeBlock(10)
	assert.NotNil(t, err)
	err = lg.StateLedger.(*StateLedger).removeJournalsBeforeBlock(0)
	assert.Nil(t, err)

	// Extra Test
	hash := types.NewHashByStr("0xe9FC370DD36C9BD5f67cCfbc031C909F53A3d8bC7084C01362c55f2D42bA841c")
	revid := lg.StateLedger.(*StateLedger).Snapshot()
	lg.StateLedger.(*StateLedger).logs.thash = hash
	lg.StateLedger.(*StateLedger).AddLog(&types.EvmLog{
		TransactionHash: lg.StateLedger.(*StateLedger).logs.thash,
	})
	lg.StateLedger.(*StateLedger).GetLogs(*lg.StateLedger.(*StateLedger).logs.thash, 1, hash)
	lg.StateLedger.(*StateLedger).GetCodeHash(account)
	lg.StateLedger.(*StateLedger).GetCodeSize(account)
	currentAccount := lg.StateLedger.(*StateLedger).GetAccount(account)
	lg.StateLedger.(*StateLedger).setAccount(currentAccount)
	lg.StateLedger.(*StateLedger).AddBalance(account, big.NewInt(1))
	lg.StateLedger.(*StateLedger).SubBalance(account, big.NewInt(1))
	lg.StateLedger.(*StateLedger).AddRefund(1)
	refund := lg.StateLedger.(*StateLedger).GetRefund()
	assert.Equal(t, refund, uint64(1))
	lg.StateLedger.(*StateLedger).SubRefund(1)
	refund = lg.StateLedger.(*StateLedger).GetRefund()
	assert.Equal(t, refund, uint64(0))
	lg.StateLedger.(*StateLedger).AddAddressToAccessList(*account)
	isInAddressList := lg.StateLedger.(*StateLedger).AddressInAccessList(*account)
	assert.Equal(t, isInAddressList, true)
	lg.StateLedger.(*StateLedger).AddSlotToAccessList(*account, *hash)
	isInSlotAddressList, _ := lg.StateLedger.(*StateLedger).SlotInAccessList(*account, *hash)
	assert.Equal(t, isInSlotAddressList, true)
	lg.StateLedger.(*StateLedger).AddPreimage(*hash, []byte("11"))
	lg.StateLedger.(*StateLedger).PrepareAccessList(*account, account, []types.Address{}, ledger.AccessTupleList{})
	lg.StateLedger.(*StateLedger).Suiside(account)
	lg.StateLedger.(*StateLedger).RevertToSnapshot(revid)
	lg.StateLedger.(*StateLedger).ClearChangerAndRefund()

	lg.Close()

	// load ChainLedger from db, rollback to height 0 since no chain meta stored
	ldg, _ := initLedger(t, repoRoot, kv)
	stateLedger = ldg.StateLedger.(*StateLedger)
	assert.Equal(t, uint64(0), stateLedger.maxJnlHeight)
	assert.Equal(t, &types.Hash{}, stateLedger.prevJnlHash)

	ok, _ := ldg.GetState(account, []byte("a"))
	assert.False(t, ok)

	ok, _ = ldg.GetState(account, []byte("b"))
	assert.False(t, ok)

	ok, _ = ldg.GetState(account, []byte("c"))
	assert.False(t, ok)

	assert.Equal(t, uint64(0), ldg.GetBalance(account).Uint64())
	assert.Equal(t, []byte(nil), ldg.GetCode(account))

	ver := ldg.Version()
	assert.Equal(t, uint64(0), ver)
}

func TestChainLedger_OpenStateDB(t *testing.T) {
	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}
	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			ss, err := OpenStateDB(filepath.Join(repoRoot, tc.kvType), tc.kvType)
			assert.Nil(t, err)
			assert.NotNil(t, ss)
			ss, err = OpenStateDB(filepath.Join(repoRoot, tc.kvType), tc.kvType)
			assert.NotNil(t, err)
			assert.Nil(t, ss)
			assert.Contains(t, err.Error(), "init "+tc.kvType+" failed")
		})
	}
}

func TestChainLedger_OpenStateDB_WrongType(t *testing.T) {
	testcase := map[string]struct {
		kvType   string
		expected string
	}{
		"wrongKvType": {kvType: "none", expected: "unknow kv type"},
	}
	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			ss, err := OpenStateDB(filepath.Join(repoRoot, tc.kvType), tc.kvType)
			assert.NotNil(t, err)
			assert.Nil(t, ss)
			assert.Contains(t, err.Error(), tc.expected)
		})
	}
}

func testChainLedger_EVMAccessor(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)

	hash := common.HexToHash("0xe9FC370DD36C9BD5f67cCfbc031C909F53A3d8bC7084C01362c55f2D42bA841c")
	// create an account
	account := common.BytesToAddress(LeftPadBytes([]byte{100}, 20))

	ledger.StateLedger.(*StateLedger).CreateEVMAccount(account)
	ledger.StateLedger.(*StateLedger).AddEVMBalance(account, big.NewInt(2))
	balance := ledger.StateLedger.(*StateLedger).GetEVMBalance(account)
	assert.Equal(t, balance, big.NewInt(2))
	ledger.StateLedger.(*StateLedger).SubEVMBalance(account, big.NewInt(1))
	balance = ledger.StateLedger.(*StateLedger).GetEVMBalance(account)
	assert.Equal(t, balance, big.NewInt(1))
	ledger.StateLedger.(*StateLedger).SetEVMNonce(account, 10)
	nonce := ledger.StateLedger.(*StateLedger).GetEVMNonce(account)
	assert.Equal(t, nonce, uint64(10))
	ledger.StateLedger.(*StateLedger).GetEVMCodeHash(account)
	ledger.StateLedger.(*StateLedger).SetEVMCode(account, []byte("111"))
	code := ledger.StateLedger.(*StateLedger).GetEVMCode(account)
	assert.Equal(t, code, []byte("111"))
	codeSize := ledger.StateLedger.(*StateLedger).GetEVMCodeSize(account)
	assert.Equal(t, codeSize, 3)
	ledger.StateLedger.(*StateLedger).AddEVMRefund(2)
	refund := ledger.StateLedger.(*StateLedger).GetEVMRefund()
	assert.Equal(t, refund, uint64(2))
	ledger.StateLedger.(*StateLedger).SubEVMRefund(1)
	refund = ledger.StateLedger.(*StateLedger).GetEVMRefund()
	assert.Equal(t, refund, uint64(1))
	ledger.StateLedger.(*StateLedger).GetEVMCommittedState(account, hash)
	ledger.StateLedger.(*StateLedger).SetEVMState(account, hash, hash)
	value := ledger.StateLedger.(*StateLedger).GetEVMState(account, hash)
	assert.Equal(t, value, hash)
	ledger.StateLedger.(*StateLedger).SuisideEVM(account)
	isSuicide := ledger.StateLedger.(*StateLedger).HasSuisideEVM(account)
	assert.Equal(t, isSuicide, false)
	isExist := ledger.StateLedger.(*StateLedger).ExistEVM(account)
	assert.Equal(t, isExist, true)
	isEmpty := ledger.StateLedger.(*StateLedger).EmptyEVM(account)
	assert.Equal(t, isEmpty, false)
	ledger.StateLedger.(*StateLedger).PrepareEVMAccessList(account, &account, []common.Address{}, etherTypes.AccessList{})
	ledger.StateLedger.(*StateLedger).AddAddressToEVMAccessList(account)
	isIn := ledger.StateLedger.(*StateLedger).AddressInEVMAccessList(account)
	assert.Equal(t, isIn, true)
	ledger.StateLedger.(*StateLedger).AddSlotToEVMAccessList(account, hash)
	isSlotIn, _ := ledger.StateLedger.(*StateLedger).SlotInEVMAceessList(account, hash)
	assert.Equal(t, isSlotIn, true)
	ledger.StateLedger.(*StateLedger).AddEVMPreimage(hash, []byte("1111"))
	// ledger.StateLedger.(*SimpleLedger).PrepareEVM(hash, 1)
	ledger.StateLedger.(*StateLedger).StateDB()
	ledger.SetTxContext(types.NewHash(hash.Bytes()), 1)
	ledger.StateLedger.(*StateLedger).AddEVMLog(&etherTypes.Log{})
}

func testChainLedger_Rollback(t *testing.T, kvType string) {
	ledger, repoRoot := initLedger(t, "", kvType)
	stateLedger := ledger.StateLedger.(*StateLedger)

	// create an addr0
	addr0 := types.NewAddress(LeftPadBytes([]byte{100}, 20))
	addr1 := types.NewAddress(LeftPadBytes([]byte{101}, 20))

	hash0 := types.Hash{}
	assert.Equal(t, &hash0, stateLedger.prevJnlHash)

	ledger.PrepareBlock(nil, 1)
	ledger.SetBalance(addr0, new(big.Int).SetInt64(1))
	accounts, journal1 := ledger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(1, accounts, journal1))

	ledger.PrepareBlock(nil, 2)
	ledger.SetBalance(addr0, new(big.Int).SetInt64(2))
	ledger.SetState(addr0, []byte("a"), []byte("2"))

	code := sha256.Sum256([]byte("code"))
	ret := crypto1.Keccak256Hash(code[:])
	codeHash := ret.Bytes()
	ledger.SetCode(addr0, code[:])

	accounts, stateRoot2 := ledger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(2, accounts, stateRoot2))

	ledger.PrepareBlock(nil, 3)
	account0 := ledger.GetAccount(addr0)
	assert.Equal(t, uint64(2), account0.GetBalance().Uint64())

	ledger.SetBalance(addr1, new(big.Int).SetInt64(3))
	ledger.SetBalance(addr0, new(big.Int).SetInt64(4))
	ledger.SetState(addr0, []byte("a"), []byte("3"))
	ledger.SetState(addr0, []byte("b"), []byte("4"))

	code1 := sha256.Sum256([]byte("code1"))
	ret1 := crypto1.Keccak256Hash(code1[:])
	codeHash1 := ret1.Bytes()
	ledger.SetCode(addr0, code1[:])

	accounts, stateRoot3 := ledger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(3, accounts, stateRoot3))

	assert.Equal(t, stateRoot3, stateLedger.prevJnlHash)
	block, err := ledger.GetBlock(3)
	assert.Nil(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(3), ledger.GetChainMeta().Height)

	account0 = ledger.GetAccount(addr0)
	assert.Equal(t, uint64(4), account0.GetBalance().Uint64())

	err = ledger.Rollback(4)
	assert.Equal(t, fmt.Sprintf("rollback state to height 4 failed: %s", ErrorRollbackToHigherNumber), err.Error())

	hash := ledger.GetBlockHash(3)
	assert.NotNil(t, hash)

	hash = ledger.GetBlockHash(100)
	assert.NotNil(t, hash)

	num, err := ledger.GetTransactionCount(3)
	assert.Nil(t, err)
	assert.NotNil(t, num)

	meta, err := ledger.LoadChainMeta()
	assert.Nil(t, err)
	assert.NotNil(t, meta)
	assert.Equal(t, uint64(3), meta.Height)

	//
	// err = ledger.Rollback(0)
	// assert.Equal(t, ErrorRollbackTooMuch, err)
	//
	// err = ledger.Rollback(1)
	// assert.Equal(t, ErrorRollbackTooMuch, err)
	// assert.Equal(t, uint64(3), ledger.GetChainMeta().Height)

	err = ledger.Rollback(3)
	assert.Nil(t, err)
	assert.Equal(t, stateRoot3, stateLedger.prevJnlHash)
	block, err = ledger.GetBlock(3)
	assert.Nil(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(3), ledger.GetChainMeta().Height)
	assert.Equal(t, codeHash1, account0.CodeHash())
	assert.Equal(t, code1[:], account0.Code())

	err = ledger.Rollback(2)
	assert.Nil(t, err)
	block, err = ledger.GetBlock(3)
	assert.Equal(t, "get bodies with height 3 from blockfile failed: out of bounds", err.Error())
	assert.Nil(t, block)
	assert.Equal(t, uint64(2), ledger.GetChainMeta().Height)
	assert.Equal(t, stateRoot2.String(), stateLedger.prevJnlHash.String())
	assert.Equal(t, uint64(1), stateLedger.minJnlHeight)
	assert.Equal(t, uint64(2), stateLedger.maxJnlHeight)

	account0 = ledger.GetAccount(addr0)
	assert.Equal(t, uint64(2), account0.GetBalance().Uint64())
	assert.Equal(t, uint64(0), account0.GetNonce())
	assert.Equal(t, codeHash[:], account0.CodeHash())
	assert.Equal(t, code[:], account0.Code())
	ok, val := account0.GetState([]byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("2"), val)

	account1 := ledger.GetAccount(addr1)
	assert.Nil(t, account1)

	ledger.GetChainMeta()
	ledger.Close()

	ledger, _ = initLedger(t, repoRoot, kvType)
	assert.Equal(t, uint64(1), stateLedger.minJnlHeight)
	assert.Equal(t, uint64(2), stateLedger.maxJnlHeight)

	err = ledger.Rollback(1)
	assert.Nil(t, err)
	err = ledger.Rollback(0)
	assert.Nil(t, err)

	err = ledger.Rollback(100)
	assert.NotNil(t, err)
}

func testChainLedger_QueryByPrefix(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)

	addr := types.NewAddress(LeftPadBytes([]byte{1}, 20))
	key0 := []byte{100, 100}
	key1 := []byte{100, 101}
	key2 := []byte{100, 102}
	key3 := []byte{10, 102}

	ledger.SetState(addr, key0, []byte("0"))
	ledger.SetState(addr, key1, []byte("1"))
	ledger.SetState(addr, key2, []byte("2"))
	ledger.SetState(addr, key3, []byte("2"))

	accounts, stateRoot := ledger.FlushDirtyData()

	ok, vals := ledger.QueryByPrefix(addr, string([]byte{100}))
	assert.True(t, ok)
	assert.Equal(t, 3, len(vals))
	assert.Equal(t, []byte("0"), vals[0])
	assert.Equal(t, []byte("1"), vals[1])
	assert.Equal(t, []byte("2"), vals[2])

	err := ledger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	ok, vals = ledger.QueryByPrefix(addr, string([]byte{100}))
	assert.True(t, ok)
	assert.Equal(t, 3, len(vals))
	assert.Equal(t, []byte("0"), vals[0])
	assert.Equal(t, []byte("1"), vals[1])
	assert.Equal(t, []byte("2"), vals[2])
}

func testChainLedger_GetAccount(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)

	addr := types.NewAddress(LeftPadBytes([]byte{1}, 20))
	code := LeftPadBytes([]byte{1}, 120)
	key0 := []byte{100, 100}
	key1 := []byte{100, 101}

	account := ledger.GetOrCreateAccount(addr)
	account.SetBalance(new(big.Int).SetInt64(1))
	account.SetNonce(2)
	account.SetCodeAndHash(code)

	account.SetState(key0, key1)
	account.SetState(key1, key0)

	accounts, stateRoot := ledger.FlushDirtyData()
	err := ledger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	account1 := ledger.GetAccount(addr)

	assert.Equal(t, account.GetBalance(), ledger.GetBalance(addr))
	assert.Equal(t, account.GetBalance(), account1.GetBalance())
	assert.Equal(t, account.GetNonce(), account1.GetNonce())
	assert.Equal(t, account.CodeHash(), account1.CodeHash())
	assert.Equal(t, account.Code(), account1.Code())
	ok0, val0 := account.GetState(key0)
	ok1, val1 := account.GetState(key1)
	assert.Equal(t, ok0, ok1)
	assert.Equal(t, val0, key1)
	assert.Equal(t, val1, key0)

	key2 := []byte{100, 102}
	val2 := []byte{111}
	ledger.SetState(addr, key0, val0)
	ledger.SetState(addr, key2, val2)
	ledger.SetState(addr, key0, val1)
	accounts, stateRoot = ledger.FlushDirtyData()
	err = ledger.Commit(2, accounts, stateRoot)
	assert.Nil(t, err)

	ledger.SetState(addr, key0, val0)
	ledger.SetState(addr, key0, val1)
	ledger.SetState(addr, key2, nil)
	accounts, stateRoot = ledger.FlushDirtyData()
	err = ledger.Commit(3, accounts, stateRoot)
	assert.Nil(t, err)

	ok, val := ledger.GetState(addr, key0)
	assert.True(t, ok)
	assert.Equal(t, val1, val)

	ok, val2 = ledger.GetState(addr, key2)
	assert.False(t, ok)
	assert.Nil(t, val2)
}

func testChainLedger_GetCode(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)

	addr := types.NewAddress(LeftPadBytes([]byte{1}, 20))
	code := LeftPadBytes([]byte{10}, 120)

	code0 := ledger.GetCode(addr)
	assert.Nil(t, code0)

	ledger.SetCode(addr, code)

	accounts, stateRoot := ledger.FlushDirtyData()
	err := ledger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	vals := ledger.GetCode(addr)
	assert.Equal(t, code, vals)

	accounts, stateRoot = ledger.FlushDirtyData()
	err = ledger.Commit(2, accounts, stateRoot)
	assert.Nil(t, err)

	vals = ledger.GetCode(addr)
	assert.Equal(t, code, vals)
}

func testChainLedger_AddAccountsToCache(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	stateLedger := ledger.StateLedger.(*StateLedger)

	addr := types.NewAddress(LeftPadBytes([]byte{1}, 20))
	key := []byte{1}
	val := []byte{2}
	code := RightPadBytes([]byte{1, 2, 3, 4}, 100)

	ledger.SetBalance(addr, new(big.Int).SetInt64(100))
	ledger.SetNonce(addr, 1)
	ledger.SetState(addr, key, val)
	ledger.SetCode(addr, code)

	accounts, stateRoot := ledger.FlushDirtyData()
	ledger.Clear()

	mycode := ledger.GetCode(addr)
	assert.Equal(t, code, mycode)
	innerAccount, ok := stateLedger.accountCache.getInnerAccount(addr)
	assert.True(t, ok)
	assert.Equal(t, uint64(100), innerAccount.Balance.Uint64())
	assert.Equal(t, uint64(1), innerAccount.Nonce)
	ret := crypto1.Keccak256Hash(code)
	codeHash := ret.Bytes()
	assert.Equal(t, types.NewHash(codeHash).Bytes(), innerAccount.CodeHash)

	val1, ok := stateLedger.accountCache.getState(addr, string(key))
	assert.True(t, ok)
	assert.Equal(t, val, val1)

	code1, ok := stateLedger.accountCache.getCode(addr)
	assert.True(t, ok)
	assert.Equal(t, code, code1)

	assert.Equal(t, uint64(100), ledger.GetBalance(addr).Uint64())
	assert.Equal(t, uint64(1), ledger.GetNonce(addr))

	ok, val1 = ledger.GetState(addr, key)
	assert.Equal(t, true, ok)
	assert.Equal(t, val, val1)
	assert.Equal(t, code, ledger.GetCode(addr))

	err := ledger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	assert.Equal(t, uint64(100), ledger.GetBalance(addr).Uint64())
	assert.Equal(t, uint64(1), ledger.GetNonce(addr))

	ok, val1 = ledger.GetState(addr, key)
	assert.Equal(t, true, ok)
	assert.Equal(t, val, val1)
	assert.Equal(t, code, ledger.GetCode(addr))

	_, ok = stateLedger.accountCache.getInnerAccount(addr)
	assert.True(t, ok)

	_, ok = stateLedger.accountCache.getState(addr, string(key))
	assert.True(t, ok)

	_, ok = stateLedger.accountCache.getCode(addr)
	assert.True(t, ok)
}

func testChainLedger_AddState(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)

	account := types.NewAddress(LeftPadBytes([]byte{100}, 20))
	key0 := "100"
	value0 := []byte{100}
	ledger.AddState(account, []byte(key0), value0)
	accounts, journal := ledger.FlushDirtyData()

	ledger.PersistBlockData(genBlockData(1, accounts, journal))
	require.Equal(t, uint64(1), ledger.Version())

	ok, val := ledger.GetState(account, []byte(key0))
	assert.True(t, ok)
	assert.Equal(t, value0, val)

	key1 := "101"
	value0 = []byte{99}
	value1 := []byte{101}
	ledger.SetState(account, []byte(key0), value0)
	ledger.AddState(account, []byte(key1), value1)
	accounts, journal = ledger.FlushDirtyData()

	ledger.PersistBlockData(genBlockData(2, accounts, journal))
	require.Equal(t, uint64(2), ledger.Version())

	ok, val = ledger.GetState(account, []byte(key0))
	assert.True(t, ok)
	assert.Equal(t, value0, val)

	ok, val = ledger.GetState(account, []byte(key1))
	assert.True(t, ok)
	assert.Equal(t, value1, val)
}

func testChainLedger_AddEvent(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)

	hash0 := types.NewHash([]byte{1})
	hash1 := types.NewHash([]byte{2})
	event00 := &types.Event{
		TxHash:    hash0,
		Data:      nil,
		EventType: types.EventOTHER,
	}
	event01 := &types.Event{
		TxHash:    hash0,
		Data:      []byte{1},
		EventType: types.EventOTHER,
	}
	event10 := &types.Event{
		TxHash:    hash1,
		Data:      []byte{1},
		EventType: types.EventOTHER,
	}

	ledger.AddEvent(event00)
	ledger.AddEvent(event01)
	ledger.AddEvent(event10)

	events := ledger.Events(hash0.String())
	assert.Equal(t, 2, len(events))
	assert.Equal(t, event00, events[0])
	assert.Equal(t, event01, events[1])

	events = ledger.Events("123")
	assert.Equal(t, 0, len(events))
}

func testGetBlockSign(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.GetBlockSign(uint64(0))
	assert.NotNil(t, err)
}

func testGetBlockByHash(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.GetBlockByHash(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedger).blockchainStore.Put(compositeKey(blockHashKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.GetBlockByHash(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetTransaction(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedger).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedger).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetTransaction1(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	meta := types.TransactionMeta{
		BlockHeight: 0,
	}
	metaBytes, err := meta.Marshal()
	require.Nil(t, err)
	ledger.ChainLedger.(*ChainLedger).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), metaBytes)
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedger).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetTransactionMeta(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedger).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedger).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetReceipt(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedger).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("0"))
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedger).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetReceipt1(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	meta := types.TransactionMeta{
		BlockHeight: 0,
	}
	metaBytes, err := meta.Marshal()
	require.Nil(t, err)
	ledger.ChainLedger.(*ChainLedger).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), metaBytes)
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedger).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testPrepare(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	batch := ledger.ChainLedger.(*ChainLedger).blockchainStore.NewBatch()
	transactions := []*types.Transaction{}
	transaction, err := types.GenerateEmptyTransactionAndSigner()
	require.Nil(t, err)
	transactions = append(transactions, transaction)
	block := &types.Block{
		BlockHeader: &types.BlockHeader{
			Number: uint64(0),
		},
		BlockHash:    types.NewHash([]byte{1}),
		Transactions: transactions,
	}
	_, err = ledger.ChainLedger.(*ChainLedger).prepareBlock(batch, block)
	require.Nil(t, err)
	receipts := []*types.Receipt{}
	receipt := &types.Receipt{
		TxHash: types.NewHash([]byte("1")),
	}
	receipts = append(receipts, receipt)
	_, err = ledger.ChainLedger.(*ChainLedger).prepareReceipts(batch, block, receipts)
	require.Nil(t, err)
	_, err = ledger.ChainLedger.(*ChainLedger).prepareTransactions(batch, block)
	require.Nil(t, err)

	bloomRes := CreateBloom(receipts)
	require.NotNil(t, bloomRes)
}

func genBlockData(height uint64, accounts map[string]ledger.IAccount, stateRoot *types.Hash) *BlockData {
	block := &types.Block{
		BlockHeader: &types.BlockHeader{
			Number:    height,
			StateRoot: stateRoot,
		},
		Transactions: []*types.Transaction{},
	}

	block.BlockHash = block.Hash()
	return &BlockData{
		Block: &types.Block{
			BlockHeader: &types.BlockHeader{
				Number:    height,
				StateRoot: stateRoot,
			},
			BlockHash:    types.NewHash([]byte{1}),
			Transactions: []*types.Transaction{},
		},
		Receipts: nil,
		Accounts: accounts,
	}
}

func createMockRepo(t *testing.T) *repo.Repo {
	r, err := repo.Default(t.TempDir())
	require.Nil(t, err)
	return r
}

func initLedger(t *testing.T, repoRoot string, kv string) (*Ledger, string) {
	if repoRoot == "" {
		repoRoot = t.TempDir()
	}

	var blockStorage storage.Storage
	var stateStorage storage.Storage
	var err error
	if kv == "leveldb" {
		blockStorage, err = leveldb.New(filepath.Join(repoRoot, "storage"))
		assert.Nil(t, err)
		stateStorage, err = leveldb.New(filepath.Join(repoRoot, "ledger"))
		assert.Nil(t, err)
	} else if kv == "pebble" {
		blockStorage, err = pebble.New(filepath.Join(repoRoot, "storage"))
		assert.Nil(t, err)
		stateStorage, err = pebble.New(filepath.Join(repoRoot, "ledger"))
		assert.Nil(t, err)
	}

	accountCache, err := NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	l, err := New(createMockRepo(t), blockStorage, stateStorage, blockFile, accountCache, log.NewWithModule("executor"))
	require.Nil(t, err)

	return l, repoRoot
}

// LeftPadBytes zero-pads slice to the left up to length l.
func LeftPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}

func RightPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded, slice)

	return padded
}
