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
	"github.com/axiomesh/axiom-ledger/pkg/repo"
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

			l.StateLedger.SetNonce(&types.Address{}, 2)
			c := l.Copy()
			l.StateLedger.Finalise()
			c.StateLedger.Finalise()

			d1, h1 := l.StateLedger.FlushDirtyData()
			assert.Equal(t, 1, len(d1))

			d2, h2 := c.StateLedger.FlushDirtyData()
			assert.Empty(t, d2)

			assert.NotEqual(t, h1, h2)
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

	ledger.StateLedger.SetState(account, []byte("a"), []byte("b"))
	accounts, journal := ledger.StateLedger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(1, accounts, journal))
}

func testChainLedger_Commit(t *testing.T, kv string) {
	lg, repoRoot := initLedger(t, "", kv)

	// create an account
	account := types.NewAddress(LeftPadBytes([]byte{100}, 20))

	lg.StateLedger.SetState(account, []byte("a"), []byte("b"))
	accounts, stateRoot := lg.StateLedger.FlushDirtyData()
	err := lg.StateLedger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)
	lg.StateLedger.(*StateLedgerImpl).GetCommittedState(account, []byte("a"))
	isSuicide := lg.StateLedger.(*StateLedgerImpl).HasSuicide(account)
	assert.Equal(t, isSuicide, false)
	assert.Equal(t, uint64(1), lg.StateLedger.Version())
	assert.Equal(t, "0xa1a6d35708fa6cf804b6cf9479f3a55d9a87fbfb83c55a64685aeabdba6116b1", stateRoot.String())

	accounts, stateRoot = lg.StateLedger.FlushDirtyData()
	err = lg.StateLedger.Commit(2, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), lg.StateLedger.Version())
	assert.Equal(t, "0xf09f0198c06d549316d4ee7c497c9eaef9d24f5b1075e7bcef3d0a82dfa742cf", stateRoot.String())

	lg.StateLedger.SetState(account, []byte("a"), []byte("3"))
	lg.StateLedger.SetState(account, []byte("a"), []byte("2"))
	accounts, stateRoot = lg.StateLedger.FlushDirtyData()
	err = lg.StateLedger.Commit(3, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(3), lg.StateLedger.Version())
	assert.Equal(t, "0xe9fc370dd36c9bd5f67ccfbc031c909f53a3d8bc7084c01362c55f2d42ba841c", stateRoot.String())

	lg.StateLedger.SetBalance(account, new(big.Int).SetInt64(100))
	accounts, stateRoot = lg.StateLedger.FlushDirtyData()
	err = lg.StateLedger.Commit(4, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), lg.StateLedger.Version())
	assert.Equal(t, "0xc179056204ba33ed6cfc0bfe94ca03319beb522fd7b0773a589899817b49ec08", stateRoot.String())

	code := RightPadBytes([]byte{100}, 100)
	lg.StateLedger.SetCode(account, code)
	lg.StateLedger.SetState(account, []byte("b"), []byte("3"))
	lg.StateLedger.SetState(account, []byte("c"), []byte("2"))
	accounts, stateRoot = lg.StateLedger.FlushDirtyData()
	err = lg.StateLedger.Commit(5, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), lg.StateLedger.Version())
	// assert.Equal(t, uint64(5), ledger.maxJnlHeight)

	stateLedger := lg.StateLedger.(*StateLedgerImpl)
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
	isExist := lg.StateLedger.(*StateLedgerImpl).Exist(account)
	assert.True(t, isExist)
	isEmpty := lg.StateLedger.(*StateLedgerImpl).Empty(account)
	assert.False(t, isEmpty)
	lg.StateLedger.(*StateLedgerImpl).AccountCache()
	err = lg.StateLedger.(*StateLedgerImpl).removeJournalsBeforeBlock(10)
	assert.NotNil(t, err)
	err = lg.StateLedger.(*StateLedgerImpl).removeJournalsBeforeBlock(0)
	assert.Nil(t, err)

	// Extra Test
	hash := types.NewHashByStr("0xe9FC370DD36C9BD5f67cCfbc031C909F53A3d8bC7084C01362c55f2D42bA841c")
	revid := lg.StateLedger.(*StateLedgerImpl).Snapshot()
	lg.StateLedger.(*StateLedgerImpl).logs.thash = hash
	lg.StateLedger.(*StateLedgerImpl).AddLog(&types.EvmLog{
		TransactionHash: lg.StateLedger.(*StateLedgerImpl).logs.thash,
	})
	lg.StateLedger.(*StateLedgerImpl).GetLogs(*lg.StateLedger.(*StateLedgerImpl).logs.thash, 1, hash)
	lg.StateLedger.(*StateLedgerImpl).GetCodeHash(account)
	lg.StateLedger.(*StateLedgerImpl).GetCodeSize(account)
	currentAccount := lg.StateLedger.(*StateLedgerImpl).GetAccount(account)
	lg.StateLedger.(*StateLedgerImpl).setAccount(currentAccount)
	lg.StateLedger.(*StateLedgerImpl).AddBalance(account, big.NewInt(1))
	lg.StateLedger.(*StateLedgerImpl).SubBalance(account, big.NewInt(1))
	lg.StateLedger.(*StateLedgerImpl).AddRefund(1)
	refund := lg.StateLedger.(*StateLedgerImpl).GetRefund()
	assert.Equal(t, refund, uint64(1))
	lg.StateLedger.(*StateLedgerImpl).SubRefund(1)
	refund = lg.StateLedger.(*StateLedgerImpl).GetRefund()
	assert.Equal(t, refund, uint64(0))
	lg.StateLedger.(*StateLedgerImpl).AddAddressToAccessList(*account)
	isInAddressList := lg.StateLedger.(*StateLedgerImpl).AddressInAccessList(*account)
	assert.Equal(t, isInAddressList, true)
	lg.StateLedger.(*StateLedgerImpl).AddSlotToAccessList(*account, *hash)
	isInSlotAddressList, _ := lg.StateLedger.(*StateLedgerImpl).SlotInAccessList(*account, *hash)
	assert.Equal(t, isInSlotAddressList, true)
	lg.StateLedger.(*StateLedgerImpl).AddPreimage(*hash, []byte("11"))
	lg.StateLedger.(*StateLedgerImpl).PrepareAccessList(*account, account, []types.Address{}, AccessTupleList{})
	lg.StateLedger.(*StateLedgerImpl).Suicide(account)
	lg.StateLedger.(*StateLedgerImpl).RevertToSnapshot(revid)
	lg.StateLedger.(*StateLedgerImpl).ClearChangerAndRefund()

	lg.Close()

	// load ChainLedgerImpl from db, rollback to height 0 since no chain meta stored
	ldg, _ := initLedger(t, repoRoot, kv)
	stateLedger = ldg.StateLedger.(*StateLedgerImpl)
	assert.Equal(t, uint64(0), stateLedger.maxJnlHeight)
	assert.Equal(t, &types.Hash{}, stateLedger.prevJnlHash)

	ok, _ := ldg.StateLedger.GetState(account, []byte("a"))
	assert.False(t, ok)

	ok, _ = ldg.StateLedger.GetState(account, []byte("b"))
	assert.False(t, ok)

	ok, _ = ldg.StateLedger.GetState(account, []byte("c"))
	assert.False(t, ok)

	assert.Equal(t, uint64(0), ldg.StateLedger.GetBalance(account).Uint64())
	assert.Equal(t, []byte(nil), ldg.StateLedger.GetCode(account))

	ver := ldg.StateLedger.Version()
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

	ledger.StateLedger.(*StateLedgerImpl).CreateEVMAccount(account)
	ledger.StateLedger.(*StateLedgerImpl).AddEVMBalance(account, big.NewInt(2))
	balance := ledger.StateLedger.(*StateLedgerImpl).GetEVMBalance(account)
	assert.Equal(t, balance, big.NewInt(2))
	ledger.StateLedger.(*StateLedgerImpl).SubEVMBalance(account, big.NewInt(1))
	balance = ledger.StateLedger.(*StateLedgerImpl).GetEVMBalance(account)
	assert.Equal(t, balance, big.NewInt(1))
	ledger.StateLedger.(*StateLedgerImpl).SetEVMNonce(account, 10)
	nonce := ledger.StateLedger.(*StateLedgerImpl).GetEVMNonce(account)
	assert.Equal(t, nonce, uint64(10))
	ledger.StateLedger.(*StateLedgerImpl).GetEVMCodeHash(account)
	ledger.StateLedger.(*StateLedgerImpl).SetEVMCode(account, []byte("111"))
	code := ledger.StateLedger.(*StateLedgerImpl).GetEVMCode(account)
	assert.Equal(t, code, []byte("111"))
	codeSize := ledger.StateLedger.(*StateLedgerImpl).GetEVMCodeSize(account)
	assert.Equal(t, codeSize, 3)
	ledger.StateLedger.(*StateLedgerImpl).AddEVMRefund(2)
	refund := ledger.StateLedger.(*StateLedgerImpl).GetEVMRefund()
	assert.Equal(t, refund, uint64(2))
	ledger.StateLedger.(*StateLedgerImpl).SubEVMRefund(1)
	refund = ledger.StateLedger.(*StateLedgerImpl).GetEVMRefund()
	assert.Equal(t, refund, uint64(1))
	ledger.StateLedger.(*StateLedgerImpl).GetEVMCommittedState(account, hash)
	ledger.StateLedger.(*StateLedgerImpl).SetEVMState(account, hash, hash)
	value := ledger.StateLedger.(*StateLedgerImpl).GetEVMState(account, hash)
	assert.Equal(t, value, hash)
	ledger.StateLedger.(*StateLedgerImpl).SuicideEVM(account)
	isSuicide := ledger.StateLedger.(*StateLedgerImpl).HasSuicideEVM(account)
	assert.Equal(t, isSuicide, false)
	isExist := ledger.StateLedger.(*StateLedgerImpl).ExistEVM(account)
	assert.Equal(t, isExist, true)
	isEmpty := ledger.StateLedger.(*StateLedgerImpl).EmptyEVM(account)
	assert.Equal(t, isEmpty, false)
	ledger.StateLedger.(*StateLedgerImpl).PrepareEVMAccessList(account, &account, []common.Address{}, etherTypes.AccessList{})
	ledger.StateLedger.(*StateLedgerImpl).AddAddressToEVMAccessList(account)
	isIn := ledger.StateLedger.(*StateLedgerImpl).AddressInEVMAccessList(account)
	assert.Equal(t, isIn, true)
	ledger.StateLedger.(*StateLedgerImpl).AddSlotToEVMAccessList(account, hash)
	isSlotIn, _ := ledger.StateLedger.(*StateLedgerImpl).SlotInEVMAceessList(account, hash)
	assert.Equal(t, isSlotIn, true)
	ledger.StateLedger.(*StateLedgerImpl).AddEVMPreimage(hash, []byte("1111"))
	// ledger.StateLedgerImpl.(*SimpleLedger).PrepareEVM(hash, 1)
	ledger.StateLedger.(*StateLedgerImpl).StateDB()
	ledger.StateLedger.SetTxContext(types.NewHash(hash.Bytes()), 1)
	ledger.StateLedger.(*StateLedgerImpl).AddEVMLog(&etherTypes.Log{})
}

func testChainLedger_Rollback(t *testing.T, kvType string) {
	ledger, repoRoot := initLedger(t, "", kvType)
	stateLedger := ledger.StateLedger.(*StateLedgerImpl)

	// create an addr0
	addr0 := types.NewAddress(LeftPadBytes([]byte{100}, 20))
	addr1 := types.NewAddress(LeftPadBytes([]byte{101}, 20))

	hash0 := types.Hash{}
	assert.Equal(t, &hash0, stateLedger.prevJnlHash)

	ledger.StateLedger.PrepareBlock(nil, 1)
	ledger.StateLedger.SetBalance(addr0, new(big.Int).SetInt64(1))
	accounts, journal1 := ledger.StateLedger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(1, accounts, journal1))

	ledger.StateLedger.PrepareBlock(nil, 2)
	ledger.StateLedger.SetBalance(addr0, new(big.Int).SetInt64(2))
	ledger.StateLedger.SetState(addr0, []byte("a"), []byte("2"))

	code := sha256.Sum256([]byte("code"))
	ret := crypto1.Keccak256Hash(code[:])
	codeHash := ret.Bytes()
	ledger.StateLedger.SetCode(addr0, code[:])

	accounts, stateRoot2 := ledger.StateLedger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(2, accounts, stateRoot2))

	ledger.StateLedger.PrepareBlock(nil, 3)
	account0 := ledger.StateLedger.GetAccount(addr0)
	assert.Equal(t, uint64(2), account0.GetBalance().Uint64())

	ledger.StateLedger.SetBalance(addr1, new(big.Int).SetInt64(3))
	ledger.StateLedger.SetBalance(addr0, new(big.Int).SetInt64(4))
	ledger.StateLedger.SetState(addr0, []byte("a"), []byte("3"))
	ledger.StateLedger.SetState(addr0, []byte("b"), []byte("4"))

	code1 := sha256.Sum256([]byte("code1"))
	ret1 := crypto1.Keccak256Hash(code1[:])
	codeHash1 := ret1.Bytes()
	ledger.StateLedger.SetCode(addr0, code1[:])

	accounts, stateRoot3 := ledger.StateLedger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(3, accounts, stateRoot3))

	assert.Equal(t, stateRoot3, stateLedger.prevJnlHash)
	block, err := ledger.ChainLedger.GetBlock(3)
	assert.Nil(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(3), ledger.ChainLedger.GetChainMeta().Height)

	account0 = ledger.StateLedger.GetAccount(addr0)
	assert.Equal(t, uint64(4), account0.GetBalance().Uint64())

	err = ledger.Rollback(4)
	assert.Equal(t, fmt.Sprintf("rollback state to height 4 failed: %s", ErrorRollbackToHigherNumber), err.Error())

	hash := ledger.ChainLedger.GetBlockHash(3)
	assert.NotNil(t, hash)

	hash = ledger.ChainLedger.GetBlockHash(100)
	assert.NotNil(t, hash)

	num, err := ledger.ChainLedger.GetTransactionCount(3)
	assert.Nil(t, err)
	assert.NotNil(t, num)

	meta, err := ledger.ChainLedger.LoadChainMeta()
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
	block, err = ledger.ChainLedger.GetBlock(3)
	assert.Nil(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(3), ledger.ChainLedger.GetChainMeta().Height)
	assert.Equal(t, codeHash1, account0.CodeHash())
	assert.Equal(t, code1[:], account0.Code())

	err = ledger.Rollback(2)
	assert.Nil(t, err)
	block, err = ledger.ChainLedger.GetBlock(3)
	assert.Equal(t, "get bodies with height 3 from blockfile failed: out of bounds", err.Error())
	assert.Nil(t, block)
	assert.Equal(t, uint64(2), ledger.ChainLedger.GetChainMeta().Height)
	assert.Equal(t, stateRoot2.String(), stateLedger.prevJnlHash.String())
	assert.Equal(t, uint64(1), stateLedger.minJnlHeight)
	assert.Equal(t, uint64(2), stateLedger.maxJnlHeight)

	account0 = ledger.StateLedger.GetAccount(addr0)
	assert.Equal(t, uint64(2), account0.GetBalance().Uint64())
	assert.Equal(t, uint64(0), account0.GetNonce())
	assert.Equal(t, codeHash[:], account0.CodeHash())
	assert.Equal(t, code[:], account0.Code())
	ok, val := account0.GetState([]byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("2"), val)

	account1 := ledger.StateLedger.GetAccount(addr1)
	assert.Nil(t, account1)

	ledger.ChainLedger.GetChainMeta()
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

	ledger.StateLedger.SetState(addr, key0, []byte("0"))
	ledger.StateLedger.SetState(addr, key1, []byte("1"))
	ledger.StateLedger.SetState(addr, key2, []byte("2"))
	ledger.StateLedger.SetState(addr, key3, []byte("2"))

	ok, vals := ledger.StateLedger.QueryByPrefix(addr, string([]byte{100}))
	assert.True(t, ok)
	assert.Equal(t, 3, len(vals))
	assert.Equal(t, []byte("0"), vals[0])
	assert.Equal(t, []byte("1"), vals[1])
	assert.Equal(t, []byte("2"), vals[2])

	accounts, stateRoot := ledger.StateLedger.FlushDirtyData()
	err := ledger.StateLedger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	ok, vals = ledger.StateLedger.QueryByPrefix(addr, string([]byte{100}))
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

	account := ledger.StateLedger.GetOrCreateAccount(addr)
	account.SetBalance(new(big.Int).SetInt64(1))
	account.SetNonce(2)
	account.SetCodeAndHash(code)

	account.SetState(key0, key1)
	account.SetState(key1, key0)

	accounts, stateRoot := ledger.StateLedger.FlushDirtyData()
	err := ledger.StateLedger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	account1 := ledger.StateLedger.GetAccount(addr)

	assert.Equal(t, account.GetBalance(), ledger.StateLedger.GetBalance(addr))
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
	ledger.StateLedger.SetState(addr, key0, val0)
	ledger.StateLedger.SetState(addr, key2, val2)
	ledger.StateLedger.SetState(addr, key0, val1)
	accounts, stateRoot = ledger.StateLedger.FlushDirtyData()
	err = ledger.StateLedger.Commit(2, accounts, stateRoot)
	assert.Nil(t, err)

	ledger.StateLedger.SetState(addr, key0, val0)
	ledger.StateLedger.SetState(addr, key0, val1)
	ledger.StateLedger.SetState(addr, key2, nil)
	accounts, stateRoot = ledger.StateLedger.FlushDirtyData()
	err = ledger.StateLedger.Commit(3, accounts, stateRoot)
	assert.Nil(t, err)

	ok, val := ledger.StateLedger.GetState(addr, key0)
	assert.True(t, ok)
	assert.Equal(t, val1, val)

	ok, val2 = ledger.StateLedger.GetState(addr, key2)
	assert.False(t, ok)
	assert.Nil(t, val2)
}

func testChainLedger_GetCode(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)

	addr := types.NewAddress(LeftPadBytes([]byte{1}, 20))
	code := LeftPadBytes([]byte{10}, 120)

	code0 := ledger.StateLedger.GetCode(addr)
	assert.Nil(t, code0)

	ledger.StateLedger.SetCode(addr, code)

	accounts, stateRoot := ledger.StateLedger.FlushDirtyData()
	err := ledger.StateLedger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	vals := ledger.StateLedger.GetCode(addr)
	assert.Equal(t, code, vals)

	accounts, stateRoot = ledger.StateLedger.FlushDirtyData()
	err = ledger.StateLedger.Commit(2, accounts, stateRoot)
	assert.Nil(t, err)

	vals = ledger.StateLedger.GetCode(addr)
	assert.Equal(t, code, vals)
}

func testChainLedger_AddAccountsToCache(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	stateLedger := ledger.StateLedger.(*StateLedgerImpl)

	addr := types.NewAddress(LeftPadBytes([]byte{1}, 20))
	key := []byte{1}
	val := []byte{2}
	code := RightPadBytes([]byte{1, 2, 3, 4}, 100)

	ledger.StateLedger.SetBalance(addr, new(big.Int).SetInt64(100))
	ledger.StateLedger.SetNonce(addr, 1)
	ledger.StateLedger.SetState(addr, key, val)
	ledger.StateLedger.SetCode(addr, code)

	accounts, stateRoot := ledger.StateLedger.FlushDirtyData()
	ledger.StateLedger.Clear()

	mycode := ledger.StateLedger.GetCode(addr)
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

	assert.Equal(t, uint64(100), ledger.StateLedger.GetBalance(addr).Uint64())
	assert.Equal(t, uint64(1), ledger.StateLedger.GetNonce(addr))

	ok, val1 = ledger.StateLedger.GetState(addr, key)
	assert.Equal(t, true, ok)
	assert.Equal(t, val, val1)
	assert.Equal(t, code, ledger.StateLedger.GetCode(addr))

	err := ledger.StateLedger.Commit(1, accounts, stateRoot)
	assert.Nil(t, err)

	assert.Equal(t, uint64(100), ledger.StateLedger.GetBalance(addr).Uint64())
	assert.Equal(t, uint64(1), ledger.StateLedger.GetNonce(addr))

	ok, val1 = ledger.StateLedger.GetState(addr, key)
	assert.Equal(t, true, ok)
	assert.Equal(t, val, val1)
	assert.Equal(t, code, ledger.StateLedger.GetCode(addr))

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
	ledger.StateLedger.SetState(account, []byte(key0), value0)
	accounts, journal := ledger.StateLedger.FlushDirtyData()

	ledger.PersistBlockData(genBlockData(1, accounts, journal))
	require.Equal(t, uint64(1), ledger.StateLedger.Version())

	ok, val := ledger.StateLedger.GetState(account, []byte(key0))
	assert.True(t, ok)
	assert.Equal(t, value0, val)

	key1 := "101"
	value0 = []byte{99}
	value1 := []byte{101}
	ledger.StateLedger.SetState(account, []byte(key0), value0)
	ledger.StateLedger.SetState(account, []byte(key1), value1)
	accounts, journal = ledger.StateLedger.FlushDirtyData()

	ledger.PersistBlockData(genBlockData(2, accounts, journal))
	require.Equal(t, uint64(2), ledger.StateLedger.Version())

	ok, val = ledger.StateLedger.GetState(account, []byte(key0))
	assert.True(t, ok)
	assert.Equal(t, value0, val)

	ok, val = ledger.StateLedger.GetState(account, []byte(key1))
	assert.True(t, ok)
	assert.Equal(t, value1, val)
}

func testGetBlockSign(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.ChainLedger.GetBlockSign(uint64(0))
	assert.NotNil(t, err)
}

func testGetBlockByHash(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.ChainLedger.GetBlockByHash(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(blockHashKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.ChainLedger.GetBlockByHash(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetTransaction(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.ChainLedger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.ChainLedger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.ChainLedger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetTransaction1(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.ChainLedger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	meta := types.TransactionMeta{
		BlockHeight: 0,
	}
	metaBytes, err := meta.Marshal()
	require.Nil(t, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), metaBytes)
	_, err = ledger.ChainLedger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.ChainLedger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetTransactionMeta(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.ChainLedger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.ChainLedger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.ChainLedger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetReceipt(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.ChainLedger.GetReceipt(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("0"))
	_, err = ledger.ChainLedger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.ChainLedger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testGetReceipt1(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	_, err := ledger.ChainLedger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	meta := types.TransactionMeta{
		BlockHeight: 0,
	}
	metaBytes, err := meta.Marshal()
	require.Nil(t, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), metaBytes)
	_, err = ledger.ChainLedger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.ChainLedger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func testPrepare(t *testing.T, kvType string) {
	ledger, _ := initLedger(t, "", kvType)
	batch := ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.NewBatch()
	var transactions []*types.Transaction
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
	_, err = ledger.ChainLedger.(*ChainLedgerImpl).prepareBlock(batch, block)
	require.Nil(t, err)
	var receipts []*types.Receipt
	receipt := &types.Receipt{
		TxHash: types.NewHash([]byte("1")),
	}
	receipts = append(receipts, receipt)
	_, err = ledger.ChainLedger.(*ChainLedgerImpl).prepareReceipts(batch, block, receipts)
	require.Nil(t, err)
	_, err = ledger.ChainLedger.(*ChainLedgerImpl).prepareTransactions(batch, block)
	require.Nil(t, err)

	bloomRes := CreateBloom(receipts)
	require.NotNil(t, bloomRes)
}

func genBlockData(height uint64, accounts map[string]IAccount, stateRoot *types.Hash) *BlockData {
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
