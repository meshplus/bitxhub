package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	etherTypes "github.com/ethereum/go-ethereum/core/types"
	crypto1 "github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/eth-kit/ledger"
	ledger1 "github.com/meshplus/eth-kit/ledger"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew001(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestChainLedger")
	require.Nil(t, err)

	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, nil, log.NewWithModule("executor"))
	require.Nil(t, err)
	require.NotNil(t, ledger)
}

func TestNew002(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestChainLedger")
	require.Nil(t, err)

	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	blockStorage.Put([]byte(chainMetaKey), []byte{1})

	accountCache, err := NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, accountCache, log.NewWithModule("executor"))
	require.NotNil(t, err)
	require.Nil(t, ledger)
}

func TestNew003(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestChainLedger")
	require.Nil(t, err)

	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	ldb.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(1))

	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, nil, log.NewWithModule("executor"))
	require.NotNil(t, err)
	require.Nil(t, ledger)
}

func TestNew004(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestChainLedger")
	require.Nil(t, err)

	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	ldb.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(1))

	journal := &BlockJournal{}
	data, err := json.Marshal(journal)
	assert.Nil(t, err)

	ldb.Put(compositeKey(journalKey, 1), data)

	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, nil, log.NewWithModule("executor"))
	require.Nil(t, err)
	require.NotNil(t, ledger)
}

func TestNew005(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestChainLedger")
	require.Nil(t, err)

	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	ldb.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(5))
	ldb.Put(compositeKey(journalKey, minHeightStr), marshalHeight(3))

	journal := &BlockJournal{}
	data, err := json.Marshal(journal)
	assert.Nil(t, err)

	ldb.Put(compositeKey(journalKey, 5), data)

	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, nil, log.NewWithModule("executor"))
	require.NotNil(t, err)
	require.Nil(t, ledger)
}

func TestChainLedger_PersistBlockData(t *testing.T) {
	ledger, _ := initLedger(t, "")

	// create an account
	account := types.NewAddress(bytesutil.LeftPadBytes([]byte{100}, 20))

	ledger.SetState(account, []byte("a"), []byte("b"))
	accounts, journal := ledger.FlushDirtyData()
	ledger.PersistBlockData(genBlockData(1, accounts, journal))
}

func TestChainLedger_Commit(t *testing.T) {
	ledger, repoRoot := initLedger(t, "")

	// create an account
	account := types.NewAddress(bytesutil.LeftPadBytes([]byte{100}, 20))

	ledger.SetState(account, []byte("a"), []byte("b"))
	accounts, stateRoot := ledger.FlushDirtyData()
	err := ledger.Commit(1, accounts, stateRoot)
	ledger.StateLedger.(*SimpleLedger).GetCommittedState(account, []byte("a"))
	assert.Nil(t, err)
	isSuicide := ledger.StateLedger.(*SimpleLedger).HasSuiside(account)
	assert.Equal(t, isSuicide, false)
	assert.Equal(t, uint64(1), ledger.Version())
	assert.Equal(t, "0xA1a6d35708Fa6Cf804B6cF9479F3a55d9A87FbFB83c55a64685AeaBdBa6116B1", stateRoot.String())

	accounts, stateRoot = ledger.FlushDirtyData()
	err = ledger.Commit(2, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), ledger.Version())
	assert.Equal(t, "0xF09F0198C06D549316D4ee7C497C9eaeF9D24f5b1075e7bCEF3D0a82DfA742cF", stateRoot.String())

	ledger.SetState(account, []byte("a"), []byte("3"))
	ledger.SetState(account, []byte("a"), []byte("2"))
	accounts, stateRoot = ledger.FlushDirtyData()
	err = ledger.Commit(3, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(3), ledger.Version())
	assert.Equal(t, "0xe9FC370DD36C9BD5f67cCfbc031C909F53A3d8bC7084C01362c55f2D42bA841c", stateRoot.String())

	ledger.SetBalance(account, new(big.Int).SetInt64(100))
	accounts, stateRoot = ledger.FlushDirtyData()
	err = ledger.Commit(4, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), ledger.Version())
	assert.Equal(t, "0xC179056204BA33eD6CFC0bfE94ca03319BEb522fd7B0773A589899817B49ec08", stateRoot.String())

	code := bytesutil.RightPadBytes([]byte{100}, 100)
	ledger.SetCode(account, code)
	ledger.SetState(account, []byte("b"), []byte("3"))
	ledger.SetState(account, []byte("c"), []byte("2"))
	accounts, stateRoot = ledger.FlushDirtyData()
	err = ledger.Commit(5, accounts, stateRoot)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), ledger.Version())
	//assert.Equal(t, uint64(5), ledger.maxJnlHeight)

	stateLedger := ledger.StateLedger.(*SimpleLedger)
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
	isExist := ledger.StateLedger.(*SimpleLedger).Exist(account)
	assert.True(t, isExist)
	isEmpty := ledger.StateLedger.(*SimpleLedger).Empty(account)
	assert.False(t, isEmpty)
	ledger.StateLedger.(*SimpleLedger).AccountCache()
	err = ledger.StateLedger.(*SimpleLedger).removeJournalsBeforeBlock(10)
	assert.NotNil(t, err)
	err = ledger.StateLedger.(*SimpleLedger).removeJournalsBeforeBlock(0)
	assert.Nil(t, err)

	// Extra Test
	hash := types.NewHashByStr("0xe9FC370DD36C9BD5f67cCfbc031C909F53A3d8bC7084C01362c55f2D42bA841c")
	revid := ledger.StateLedger.(*SimpleLedger).Snapshot()
	ledger.StateLedger.(*SimpleLedger).logs.thash = hash
	ledger.StateLedger.(*SimpleLedger).AddLog(&pb.EvmLog{})
	ledger.StateLedger.(*SimpleLedger).GetLogs(*ledger.StateLedger.(*SimpleLedger).logs.thash)
	ledger.StateLedger.(*SimpleLedger).GetCodeHash(account)
	ledger.StateLedger.(*SimpleLedger).GetCodeSize(account)
	currentAccount := ledger.StateLedger.(*SimpleLedger).GetAccount(account)
	ledger.StateLedger.(*SimpleLedger).setAccount(currentAccount)
	ledger.StateLedger.(*SimpleLedger).AddBalance(account, big.NewInt(1))
	ledger.StateLedger.(*SimpleLedger).SubBalance(account, big.NewInt(1))
	ledger.StateLedger.(*SimpleLedger).AddRefund(1)
	refund := ledger.StateLedger.(*SimpleLedger).GetRefund()
	assert.Equal(t, refund, uint64(1))
	ledger.StateLedger.(*SimpleLedger).SubRefund(1)
	refund = ledger.StateLedger.(*SimpleLedger).GetRefund()
	assert.Equal(t, refund, uint64(0))
	ledger.StateLedger.(*SimpleLedger).AddAddressToAccessList(*account)
	isInAddressList := ledger.StateLedger.(*SimpleLedger).AddressInAccessList(*account)
	assert.Equal(t, isInAddressList, true)
	ledger.StateLedger.(*SimpleLedger).AddSlotToAccessList(*account, *hash)
	isInSlotAddressList, _ := ledger.StateLedger.(*SimpleLedger).SlotInAccessList(*account, *hash)
	assert.Equal(t, isInSlotAddressList, true)
	ledger.StateLedger.(*SimpleLedger).AddPreimage(*hash, []byte("11"))
	ledger.StateLedger.(*SimpleLedger).PrepareAccessList(*account, account, []types.Address{}, ledger1.AccessTupleList{})
	ledger.StateLedger.(*SimpleLedger).Suiside(account)
	ledger.StateLedger.(*SimpleLedger).RevertToSnapshot(revid)
	ledger.StateLedger.(*SimpleLedger).ClearChangerAndRefund()

	ledger.Close()

	// load ChainLedger from db, rollback to height 0 since no chain meta stored
	ldg, repoRoot := initLedger(t, repoRoot)
	stateLedger = ldg.StateLedger.(*SimpleLedger)
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
	repoRoot, err := ioutil.TempDir("", "simstorage")
	assert.Nil(t, err)
	repoRoot1, err := ioutil.TempDir("", "ethstorage")
	assert.Nil(t, err)
	logger := log.NewWithModule("opendb_test")
	_, err = OpenStateDB(repoRoot, "simple")
	assert.Nil(t, err)
	ldb, err := OpenStateDB(repoRoot1, "complex")
	assert.Nil(t, err)
	blockFile, err := blockfile.NewBlockFile("", logger)
	assert.NotNil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)
	accountCache, err := NewAccountCache()
	assert.Nil(t, err)
	_, err = New(createMockRepo(t), blockStorage, ldb, blockFile, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)
}

func TestChainLedger_EVMAccessor(t *testing.T) {
	ledger, _ := initLedger(t, "")

	hash := common.HexToHash("0xe9FC370DD36C9BD5f67cCfbc031C909F53A3d8bC7084C01362c55f2D42bA841c")
	// create an account
	account := common.BytesToAddress(bytesutil.LeftPadBytes([]byte{100}, 20))

	ledger.StateLedger.(*SimpleLedger).CreateEVMAccount(account)
	ledger.StateLedger.(*SimpleLedger).AddEVMBalance(account, big.NewInt(2))
	balance := ledger.StateLedger.(*SimpleLedger).GetEVMBalance(account)
	assert.Equal(t, balance, big.NewInt(2))
	ledger.StateLedger.(*SimpleLedger).SubEVMBalance(account, big.NewInt(1))
	balance = ledger.StateLedger.(*SimpleLedger).GetEVMBalance(account)
	assert.Equal(t, balance, big.NewInt(1))
	ledger.StateLedger.(*SimpleLedger).SetEVMNonce(account, 10)
	nonce := ledger.StateLedger.(*SimpleLedger).GetEVMNonce(account)
	assert.Equal(t, nonce, uint64(10))
	ledger.StateLedger.(*SimpleLedger).GetEVMCodeHash(account)
	ledger.StateLedger.(*SimpleLedger).SetEVMCode(account, []byte("111"))
	code := ledger.StateLedger.(*SimpleLedger).GetEVMCode(account)
	assert.Equal(t, code, []byte("111"))
	codeSize := ledger.StateLedger.(*SimpleLedger).GetEVMCodeSize(account)
	assert.Equal(t, codeSize, 3)
	ledger.StateLedger.(*SimpleLedger).AddEVMRefund(2)
	refund := ledger.StateLedger.(*SimpleLedger).GetEVMRefund()
	assert.Equal(t, refund, uint64(2))
	ledger.StateLedger.(*SimpleLedger).SubEVMRefund(1)
	refund = ledger.StateLedger.(*SimpleLedger).GetEVMRefund()
	assert.Equal(t, refund, uint64(1))
	ledger.StateLedger.(*SimpleLedger).GetEVMCommittedState(account, hash)
	ledger.StateLedger.(*SimpleLedger).SetEVMState(account, hash, hash)
	value := ledger.StateLedger.(*SimpleLedger).GetEVMState(account, hash)
	assert.Equal(t, value, hash)
	ledger.StateLedger.(*SimpleLedger).SuisideEVM(account)
	isSuicide := ledger.StateLedger.(*SimpleLedger).HasSuisideEVM(account)
	assert.Equal(t, isSuicide, false)
	isExist := ledger.StateLedger.(*SimpleLedger).ExistEVM(account)
	assert.Equal(t, isExist, true)
	isEmpty := ledger.StateLedger.(*SimpleLedger).EmptyEVM(account)
	assert.Equal(t, isEmpty, false)
	ledger.StateLedger.(*SimpleLedger).PrepareEVMAccessList(account, &account, []common.Address{}, etherTypes.AccessList{})
	ledger.StateLedger.(*SimpleLedger).AddAddressToEVMAccessList(account)
	isIn := ledger.StateLedger.(*SimpleLedger).AddressInEVMAccessList(account)
	assert.Equal(t, isIn, true)
	ledger.StateLedger.(*SimpleLedger).AddSlotToEVMAccessList(account, hash)
	isSlotIn, _ := ledger.StateLedger.(*SimpleLedger).SlotInEVMAceessList(account, hash)
	assert.Equal(t, isSlotIn, true)
	ledger.StateLedger.(*SimpleLedger).AddEVMPreimage(hash, []byte("1111"))
	ledger.StateLedger.(*SimpleLedger).PrepareEVM(hash, 1)
	ledger.StateLedger.(*SimpleLedger).StateDB()
	ledger.StateLedger.(*SimpleLedger).AddEVMLog(&etherTypes.Log{})
}

func TestChainLedger_Rollback(t *testing.T) {
	ledger, repoRoot := initLedger(t, "")
	stateLedger := ledger.StateLedger.(*SimpleLedger)

	// create an addr0
	addr0 := types.NewAddress(bytesutil.LeftPadBytes([]byte{100}, 20))
	addr1 := types.NewAddress(bytesutil.LeftPadBytes([]byte{101}, 20))

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
	//
	//err = ledger.Rollback(0)
	//assert.Equal(t, ErrorRollbackTooMuch, err)
	//
	//err = ledger.Rollback(1)
	//assert.Equal(t, ErrorRollbackTooMuch, err)
	//assert.Equal(t, uint64(3), ledger.GetChainMeta().Height)

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
	assert.Equal(t, fmt.Sprintf("get bodies with height 3 from blockfile failed: out of bounds"), err.Error())
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

	ledger.Close()

	ledger, _ = initLedger(t, repoRoot)
	assert.Equal(t, uint64(1), stateLedger.minJnlHeight)
	assert.Equal(t, uint64(2), stateLedger.maxJnlHeight)

	err = ledger.Rollback(1)
	assert.Nil(t, err)
}

func TestChainLedger_QueryByPrefix(t *testing.T) {
	ledger, _ := initLedger(t, "")

	addr := types.NewAddress(bytesutil.LeftPadBytes([]byte{1}, 20))
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

func TestChainLedger_GetAccount(t *testing.T) {
	ledger, _ := initLedger(t, "")

	addr := types.NewAddress(bytesutil.LeftPadBytes([]byte{1}, 20))
	code := bytesutil.LeftPadBytes([]byte{1}, 120)
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

func TestChainLedger_GetCode(t *testing.T) {
	ledger, _ := initLedger(t, "")

	addr := types.NewAddress(bytesutil.LeftPadBytes([]byte{1}, 20))
	code := bytesutil.LeftPadBytes([]byte{10}, 120)

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

func TestChainLedger_AddAccountsToCache(t *testing.T) {
	ledger, _ := initLedger(t, "")
	stateLedger := ledger.StateLedger.(*SimpleLedger)

	addr := types.NewAddress(bytesutil.LeftPadBytes([]byte{1}, 20))
	key := []byte{1}
	val := []byte{2}
	code := bytesutil.RightPadBytes([]byte{1, 2, 3, 4}, 100)

	ledger.SetBalance(addr, new(big.Int).SetInt64(100))
	ledger.SetNonce(addr, 1)
	ledger.SetState(addr, key, val)
	ledger.SetCode(addr, code)

	accounts, stateRoot := ledger.FlushDirtyData()
	ledger.Clear()

	mycode := ledger.GetCode(addr)
	fmt.Println(mycode)

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

func TestChainLedger_GetInterchainMeta(t *testing.T) {
	ledger, _ := initLedger(t, "")

	// create an account
	account := types.NewAddress(bytesutil.LeftPadBytes([]byte{100}, 20))
	ledger.SetState(account, []byte("a"), []byte("b"))
	accounts, journal := ledger.FlushDirtyData()

	meta, err := ledger.GetInterchainMeta(1)
	require.Equal(t, "get interchain info with height 1 from blockfile failed: out of bounds", err.Error())
	require.Nil(t, meta)

	ledger.PersistBlockData(genBlockData(1, accounts, journal))
	require.Equal(t, uint64(1), ledger.Version())

	meta, err = ledger.GetInterchainMeta(1)
	require.Nil(t, err)
	require.Equal(t, 0, len(meta.Counter))

	// deprecated for blockfile
	// meta = &pb.InterchainMeta{
	// 	Counter: make(map[string]*pb.Uint64Slice),
	// 	L2Roots: make([]types.Hash, 0),
	// }
	// meta.Counter["a"] = &pb.Uint64Slice{}
	// meta.L2Roots = append(meta.L2Roots, *types.NewHash([]byte{}))
	// batch := ledger.blockchainStore.NewBatch()
	// err = ledger.persistInterChainMeta(batch, meta, 2)
	// require.Nil(t, err)
	// batch.Commit()

	// meta2, err := ledger.GetInterchainMeta(2)
	// require.Nil(t, err)
	// require.Equal(t, len(meta.Counter), len(meta2.Counter))
	// require.Equal(t, meta.Counter["a"], meta2.Counter["a"])
	// require.Equal(t, len(meta.L2Roots), len(meta2.Counter))
}

func TestChainLedger_AddState(t *testing.T) {
	ledger, _ := initLedger(t, "")

	account := types.NewAddress(bytesutil.LeftPadBytes([]byte{100}, 20))
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

func TestChainLedger_AddEvent(t *testing.T) {
	ledger, _ := initLedger(t, "")

	hash0 := types.NewHash([]byte{1})
	hash1 := types.NewHash([]byte{2})
	event00 := &pb.Event{
		TxHash:    hash0,
		Data:      nil,
		EventType: pb.Event_OTHER,
	}
	event01 := &pb.Event{
		TxHash:    hash0,
		Data:      []byte{1},
		EventType: pb.Event_INTERCHAIN,
	}
	event10 := &pb.Event{
		TxHash:    hash1,
		Data:      []byte{1},
		EventType: pb.Event_INTERCHAIN,
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

func TestPutBlock(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestPutBlock")
	require.Nil(t, err)

	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, nil, log.NewWithModule("executor"))
	block := &pb.Block{}
	ledger.PutBlock(uint64(0), block)
	require.Nil(t, err)
}

func TestGetBlockSign(t *testing.T) {
	ledger, _ := initLedger(t, "")
	_, err := ledger.GetBlockSign(uint64(0))
	assert.NotNil(t, err)
}

func TestGetBlockByHash(t *testing.T) {
	ledger, _ := initLedger(t, "")
	_, err := ledger.GetBlockByHash(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(blockHashKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.GetBlockByHash(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func TestGetTransaction(t *testing.T) {
	ledger, _ := initLedger(t, "")
	_, err := ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func TestGetTransaction1(t *testing.T) {
	ledger, _ := initLedger(t, "")
	_, err := ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	meta := pb.TransactionMeta{
		BlockHeight: 0,
	}
	metaBytes, err := meta.Marshal()
	require.Nil(t, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), metaBytes)
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func TestGetTransactionMeta(t *testing.T) {
	ledger, _ := initLedger(t, "")
	_, err := ledger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("1"))
	_, err = ledger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetTransactionMeta(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func TestGetReceipt(t *testing.T) {
	ledger, _ := initLedger(t, "")
	_, err := ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), []byte("0"))
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func TestGetReceipt1(t *testing.T) {
	ledger, _ := initLedger(t, "")
	_, err := ledger.GetTransaction(types.NewHash([]byte("1")))
	assert.Equal(t, storage.ErrorNotFound, err)
	meta := pb.TransactionMeta{
		BlockHeight: 0,
	}
	metaBytes, err := meta.Marshal()
	require.Nil(t, err)
	ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.Put(compositeKey(transactionMetaKey, types.NewHash([]byte("1")).String()), metaBytes)
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
	err = ledger.ChainLedger.(*ChainLedgerImpl).bf.AppendBlock(0, []byte("1"), []byte("1"), []byte("1"), []byte("1"), []byte("1"))
	require.Nil(t, err)
	_, err = ledger.GetReceipt(types.NewHash([]byte("1")))
	assert.NotNil(t, err)
}

func TestPrepare(t *testing.T) {
	ledger, _ := initLedger(t, "")
	batch := ledger.ChainLedger.(*ChainLedgerImpl).blockchainStore.NewBatch()
	transactions := []pb.Transaction{}
	transaction := &pb.BxhTransaction{
		TransactionHash: types.NewHash([]byte("1")),
	}
	transactions = append(transactions, transaction)
	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number: uint64(0),
		},
		BlockHash:    types.NewHash([]byte{1}),
		Transactions: &pb.Transactions{Transactions: transactions},
	}
	_, err := ledger.ChainLedger.(*ChainLedgerImpl).prepareBlock(batch, block)
	require.Nil(t, err)
	receipts := []*pb.Receipt{}
	receipt := &pb.Receipt{
		TxHash: types.NewHash([]byte("1")),
	}
	receipts = append(receipts, receipt)
	_, err = ledger.ChainLedger.(*ChainLedgerImpl).prepareReceipts(batch, block, receipts)
	require.Nil(t, err)
	_, err = ledger.ChainLedger.(*ChainLedgerImpl).prepareTransactions(batch, block)
	require.Nil(t, err)
}

func genBlockData(height uint64, accounts map[string]ledger.IAccount, stateRoot *types.Hash) *BlockData {
	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number:    height,
			StateRoot: stateRoot,
		},
		Transactions: &pb.Transactions{},
	}

	block.BlockHash = block.Hash()
	return &BlockData{
		Block: &pb.Block{
			BlockHeader: &pb.BlockHeader{
				Number:    height,
				StateRoot: stateRoot,
			},
			BlockHash:    types.NewHash([]byte{1}),
			Transactions: &pb.Transactions{},
		},
		Receipts:       nil,
		Accounts:       accounts,
		InterchainMeta: &pb.InterchainMeta{},
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

	return &repo.Repo{
		Key: &repo.Key{
			PrivKey: privKey,
			Address: address.String(),
		},
		Config: &repo.Config{
			RepoRoot: "",
			Title:    "",
			Solo:     false,
			Port:     repo.Port{},
			PProf:    repo.PProf{},
			Monitor:  repo.Monitor{},
			Limiter:  repo.Limiter{},
			Gateway:  repo.Gateway{},
			Ping:     repo.Ping{},
			Log:      repo.Log{},
			Cert:     repo.Cert{},
			Txpool:   repo.Txpool{},
			Order:    repo.Order{},
			Executor: repo.Executor{},
			Ledger: repo.Ledger{
				Type: "simple",
			},
			Genesis:  repo.Genesis{},
			Security: repo.Security{},
		},
	}
}

func initLedger(t *testing.T, repoRoot string) (*Ledger, string) {
	if repoRoot == "" {
		root, err := ioutil.TempDir("", "TestChainLedger")
		require.Nil(t, err)
		repoRoot = root
	}

	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache, err := NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, accountCache, log.NewWithModule("executor"))
	require.Nil(t, err)

	return ledger, repoRoot
}
