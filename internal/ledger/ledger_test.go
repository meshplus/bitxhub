package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLedger_Commit(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	// create an account
	account := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{100}, 20))

	ledger.SetState(account, []byte("a"), []byte("b"))
	accounts, journal := ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(1, accounts, journal))
	assert.Equal(t, uint64(1), ledger.Version())
	assert.Equal(t, "0xA1a6d35708Fa6Cf804B6cF9479F3a55d9A87FbFB83c55a64685AeaBdBa6116B1", journal.ChangedHash.Hex())

	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(2, accounts, journal))
	assert.Equal(t, uint64(2), ledger.Version())
	assert.Equal(t, "0xF09F0198C06D549316D4ee7C497C9eaeF9D24f5b1075e7bCEF3D0a82DfA742cF", journal.ChangedHash.Hex())

	ledger.SetState(account, []byte("a"), []byte("3"))
	ledger.SetState(account, []byte("a"), []byte("2"))
	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(3, accounts, journal))
	assert.Equal(t, uint64(3), ledger.Version())
	assert.Equal(t, "0xe9FC370DD36C9BD5f67cCfbc031C909F53A3d8bC7084C01362c55f2D42bA841c", journal.ChangedHash.Hex())

	ledger.SetBalance(account, 100)
	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(4, accounts, journal))
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), ledger.Version())
	assert.Equal(t, "0xC179056204BA33eD6CFC0bfE94ca03319BEb522fd7B0773A589899817B49ec08", journal.ChangedHash.Hex())

	code := bytesutil.RightPadBytes([]byte{100}, 100)
	ledger.SetCode(account, code)
	ledger.SetState(account, []byte("b"), []byte("3"))
	ledger.SetState(account, []byte("c"), []byte("2"))
	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(5, accounts, journal))
	assert.Equal(t, uint64(5), ledger.Version())
	assert.Equal(t, uint64(5), ledger.maxJnlHeight)

	minHeight, maxHeight := getJournalRange(ledger.ldb)
	journal5 := getBlockJournal(maxHeight, ledger.ldb)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), minHeight)
	assert.Equal(t, uint64(5), maxHeight)
	assert.Equal(t, journal.ChangedHash, journal5.ChangedHash)
	assert.Equal(t, 1, len(journal5.Journals))
	entry := journal5.Journals[0]
	assert.Equal(t, account, entry.Address)
	assert.True(t, entry.AccountChanged)
	assert.Equal(t, uint64(100), entry.PrevAccount.Balance)
	assert.Equal(t, uint64(0), entry.PrevAccount.Nonce)
	assert.Nil(t, entry.PrevAccount.CodeHash)
	assert.Equal(t, 2, len(entry.PrevStates))
	assert.Nil(t, entry.PrevStates[hex.EncodeToString([]byte("b"))])
	assert.Nil(t, entry.PrevStates[hex.EncodeToString([]byte("c"))])
	assert.True(t, entry.CodeChanged)
	assert.Nil(t, entry.PrevCode)

	ledger.Close()

	// load ChainLedger from db
	ldb, err = leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	ldg, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), ldg.maxJnlHeight)
	assert.Equal(t, journal.ChangedHash, ldg.prevJnlHash)

	ok, value := ldg.GetState(account, []byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("2"), value)

	ok, value = ldg.GetState(account, []byte("b"))
	assert.True(t, ok)
	assert.Equal(t, []byte("3"), value)

	ok, value = ldg.GetState(account, []byte("c"))
	assert.True(t, ok)
	assert.Equal(t, []byte("2"), value)

	assert.Equal(t, uint64(100), ldg.GetBalance(account))
	assert.Equal(t, code, ldg.GetCode(account))

	ver := ldg.Version()
	assert.Equal(t, uint64(5), ver)
}

func TestChainLedger_Rollback(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_rollback")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	// create an addr0
	addr0 := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{100}, 20))
	addr1 := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{101}, 20))

	hash0 := types.Hash{}
	assert.Equal(t, hash0, ledger.prevJnlHash)

	ledger.SetBalance(addr0, 1)
	accounts, journal1 := ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(1, accounts, journal1))

	ledger.SetBalance(addr0, 2)
	ledger.SetState(addr0, []byte("a"), []byte("2"))

	code := sha256.Sum256([]byte("code"))
	codeHash := sha256.Sum256(code[:])
	ledger.SetCode(addr0, code[:])

	accounts, journal2 := ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(2, accounts, journal2))

	account0 := ledger.GetAccount(addr0)
	assert.Equal(t, uint64(2), account0.GetBalance())

	ledger.SetBalance(addr1, 3)
	ledger.SetBalance(addr0, 4)
	ledger.SetState(addr0, []byte("a"), []byte("3"))
	ledger.SetState(addr0, []byte("b"), []byte("4"))

	code1 := sha256.Sum256([]byte("code1"))
	codeHash1 := sha256.Sum256(code1[:])
	ledger.SetCode(addr0, code1[:])

	accounts, journal3 := ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(3, accounts, journal3))

	assert.Equal(t, journal3.ChangedHash, ledger.prevJnlHash)
	block, err := ledger.GetBlock(3)
	assert.Nil(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(3), ledger.chainMeta.Height)

	account0 = ledger.GetAccount(addr0)
	assert.Equal(t, uint64(4), account0.GetBalance())

	err = ledger.Rollback(4)
	assert.Equal(t, ErrorRollbackToHigherNumber, err)

	err = ledger.RemoveJournalsBeforeBlock(2)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), ledger.minJnlHeight)

	err = ledger.Rollback(0)
	assert.Equal(t, ErrorRollbackTooMuch, err)

	err = ledger.Rollback(1)
	assert.Equal(t, ErrorRollbackTooMuch, err)
	assert.Equal(t, uint64(3), ledger.chainMeta.Height)

	err = ledger.Rollback(3)
	assert.Nil(t, err)
	assert.Equal(t, journal3.ChangedHash, ledger.prevJnlHash)
	block, err = ledger.GetBlock(3)
	assert.Nil(t, err)
	assert.NotNil(t, block)
	assert.Equal(t, uint64(3), ledger.chainMeta.Height)
	assert.Equal(t, codeHash1[:], account0.CodeHash())
	assert.Equal(t, code1[:], account0.Code())

	err = ledger.Rollback(2)
	assert.Nil(t, err)
	block, err = ledger.GetBlock(3)
	assert.Equal(t, storage.ErrorNotFound, err)
	assert.Nil(t, block)
	assert.Equal(t, uint64(2), ledger.chainMeta.Height)
	assert.Equal(t, journal2.ChangedHash, ledger.prevJnlHash)
	assert.Equal(t, uint64(2), ledger.minJnlHeight)
	assert.Equal(t, uint64(2), ledger.maxJnlHeight)

	account0 = ledger.GetAccount(addr0)
	assert.Equal(t, uint64(2), account0.GetBalance())
	assert.Equal(t, uint64(0), account0.GetNonce())
	assert.Equal(t, codeHash[:], account0.CodeHash())
	assert.Equal(t, code[:], account0.Code())
	ok, val := account0.GetState([]byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("2"), val)

	account1 := ledger.GetAccount(addr1)
	assert.Equal(t, uint64(0), account1.GetBalance())
	assert.Equal(t, uint64(0), account1.GetNonce())
	assert.Nil(t, account1.CodeHash())
	assert.Nil(t, account1.Code())

	ledger.Close()

	ldb, err = leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	ledger, err = New(blockStorage, ldb, NewAccountCache(), log.NewWithModule("executor"))
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), ledger.minJnlHeight)
	assert.Equal(t, uint64(2), ledger.maxJnlHeight)

	err = ledger.Rollback(1)
	assert.Equal(t, ErrorRollbackTooMuch, err)
}

func TestChainLedger_RemoveJournalsBeforeBlock(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_removeJournal")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	assert.Equal(t, uint64(0), ledger.minJnlHeight)
	assert.Equal(t, uint64(0), ledger.maxJnlHeight)

	accounts, journal := ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(1, accounts, journal))
	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(2, accounts, journal))
	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(3, accounts, journal))
	accounts, journal4 := ledger.FlushDirtyDataAndComputeJournal()
	ledger.PersistBlockData(genBlockData(4, accounts, journal4))

	assert.Equal(t, uint64(1), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)

	minHeight, maxHeight := getJournalRange(ledger.ldb)
	journal = getBlockJournal(maxHeight, ledger.ldb)
	assert.Equal(t, uint64(1), minHeight)
	assert.Equal(t, uint64(4), maxHeight)
	assert.Equal(t, journal4.ChangedHash, journal.ChangedHash)

	err = ledger.RemoveJournalsBeforeBlock(5)
	assert.Equal(t, ErrorRemoveJournalOutOfRange, err)

	err = ledger.RemoveJournalsBeforeBlock(2)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)

	minHeight, maxHeight = getJournalRange(ledger.ldb)
	journal = getBlockJournal(maxHeight, ledger.ldb)
	assert.Equal(t, uint64(2), minHeight)
	assert.Equal(t, uint64(4), maxHeight)
	assert.Equal(t, journal4.ChangedHash, journal.ChangedHash)

	err = ledger.RemoveJournalsBeforeBlock(2)
	assert.Nil(t, err)

	assert.Equal(t, uint64(2), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)

	err = ledger.RemoveJournalsBeforeBlock(1)
	assert.Nil(t, err)

	assert.Equal(t, uint64(2), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)

	err = ledger.RemoveJournalsBeforeBlock(4)
	assert.Nil(t, err)

	assert.Equal(t, uint64(4), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)
	assert.Equal(t, journal4.ChangedHash, ledger.prevJnlHash)

	minHeight, maxHeight = getJournalRange(ledger.ldb)
	journal = getBlockJournal(maxHeight, ledger.ldb)
	assert.Equal(t, uint64(4), minHeight)
	assert.Equal(t, uint64(4), maxHeight)
	assert.Equal(t, journal4.ChangedHash, journal.ChangedHash)

	ledger.Close()
	ldb, err = leveldb.New(filepath.Join(repoRoot, "ledger"))
	ledger, err = New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)
	assert.Equal(t, journal4.ChangedHash, ledger.prevJnlHash)
}

func TestChainLedger_QueryByPrefix(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_queryByPrefix")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	addr := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{1}, 20))
	key0 := []byte{100, 100}
	key1 := []byte{100, 101}
	key2 := []byte{100, 102}
	key3 := []byte{10, 102}

	ledger.SetState(addr, key0, []byte("0"))
	ledger.SetState(addr, key1, []byte("1"))
	ledger.SetState(addr, key2, []byte("2"))
	ledger.SetState(addr, key3, []byte("2"))

	accounts, journal := ledger.FlushDirtyDataAndComputeJournal()

	ok, vals := ledger.QueryByPrefix(addr, string([]byte{100}))
	assert.True(t, ok)
	assert.Equal(t, 3, len(vals))
	assert.Equal(t, []byte("0"), vals[0])
	assert.Equal(t, []byte("1"), vals[1])
	assert.Equal(t, []byte("2"), vals[2])

	err = ledger.Commit(1, accounts, journal)
	assert.Nil(t, err)

	ok, vals = ledger.QueryByPrefix(addr, string([]byte{100}))
	assert.True(t, ok)
	assert.Equal(t, 3, len(vals))
	assert.Equal(t, []byte("0"), vals[0])
	assert.Equal(t, []byte("1"), vals[1])
	assert.Equal(t, []byte("2"), vals[2])

}

func TestChainLedger_GetAccount(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_getAccount")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	addr := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{1}, 20))
	code := bytesutil.LeftPadBytes([]byte{1}, 120)
	key0 := []byte{100, 100}
	key1 := []byte{100, 101}

	account := ledger.GetOrCreateAccount(addr)
	account.SetBalance(1)
	account.SetNonce(2)
	account.SetCodeAndHash(code)

	account.SetState(key0, key1)
	account.SetState(key1, key0)

	accounts, journal := ledger.FlushDirtyDataAndComputeJournal()
	err = ledger.Commit(1, accounts, journal)
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
	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	err = ledger.Commit(2, accounts, journal)
	assert.Nil(t, err)

	ledger.SetState(addr, key0, val0)
	ledger.SetState(addr, key0, val1)
	ledger.SetState(addr, key2, nil)
	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	err = ledger.Commit(3, accounts, journal)
	assert.Nil(t, err)

	ok, val := ledger.GetState(addr, key0)
	assert.True(t, ok)
	assert.Equal(t, val1, val)

	ok, val2 = ledger.GetState(addr, key2)
	assert.False(t, ok)
	assert.Nil(t, val2)
}

func TestChainLedger_GetCode(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_getCode")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	addr := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{1}, 20))
	code := bytesutil.LeftPadBytes([]byte{10}, 120)

	code0 := ledger.GetCode(addr)
	assert.Nil(t, code0)

	ledger.SetCode(addr, code)

	accounts, journal := ledger.FlushDirtyDataAndComputeJournal()
	err = ledger.Commit(1, accounts, journal)
	assert.Nil(t, err)

	vals := ledger.GetCode(addr)
	assert.Equal(t, code, vals)

	accounts, journal = ledger.FlushDirtyDataAndComputeJournal()
	err = ledger.Commit(2, accounts, journal)
	assert.Nil(t, err)

	vals = ledger.GetCode(addr)
	assert.Equal(t, code, vals)

	vals = ledger.GetCode(addr)
	assert.Equal(t, code, vals)
}

func TestChainLedger_AddAccountsToCache(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_addAccountToCache")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	assert.Nil(t, err)

	addr := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{1}, 20))
	key := []byte{1}
	val := []byte{2}
	code := bytesutil.RightPadBytes([]byte{1, 2, 3, 4}, 100)

	ledger.SetBalance(addr, 100)
	ledger.SetNonce(addr, 1)
	ledger.SetState(addr, key, val)
	ledger.SetCode(addr, code)

	accounts, journal := ledger.FlushDirtyDataAndComputeJournal()
	ledger.Clear()

	innerAccount, ok := ledger.accountCache.getInnerAccount(addr.Hex())
	assert.True(t, ok)
	assert.Equal(t, uint64(100), innerAccount.Balance)
	assert.Equal(t, uint64(1), innerAccount.Nonce)
	assert.Equal(t, types.Hash(sha256.Sum256(code)).Bytes(), innerAccount.CodeHash)

	val1, ok := ledger.accountCache.getState(addr.Hex(), string(key))
	assert.True(t, ok)
	assert.Equal(t, val, val1)

	code1, ok := ledger.accountCache.getCode(addr.Hex())
	assert.True(t, ok)
	assert.Equal(t, code, code1)

	assert.Equal(t, uint64(100), ledger.GetBalance(addr))
	assert.Equal(t, uint64(1), ledger.GetNonce(addr))

	ok, val1 = ledger.GetState(addr, key)
	assert.Equal(t, true, ok)
	assert.Equal(t, val, val1)
	assert.Equal(t, code, ledger.GetCode(addr))

	err = ledger.Commit(1, accounts, journal)
	assert.Nil(t, err)

	assert.Equal(t, uint64(100), ledger.GetBalance(addr))
	assert.Equal(t, uint64(1), ledger.GetNonce(addr))

	ok, val1 = ledger.GetState(addr, key)
	assert.Equal(t, true, ok)
	assert.Equal(t, val, val1)
	assert.Equal(t, code, ledger.GetCode(addr))

	_, ok = ledger.accountCache.getInnerAccount(addr.Hex())
	assert.False(t, ok)

	_, ok = ledger.accountCache.getState(addr.Hex(), string(key))
	assert.False(t, ok)

	_, ok = ledger.accountCache.getCode(addr.Hex())
	assert.False(t, ok)
}

func TestChainLedger_GetInterchainMeta(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_GetInterchainMeta")
	require.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache := NewAccountCache()
	ledger, err := New(blockStorage, ldb, accountCache, log.NewWithModule("executor"))
	require.Nil(t, err)

	// create an account
	account := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{100}, 20))
	ledger.SetState(account, []byte("a"), []byte("b"))
	accounts, journal := ledger.FlushDirtyDataAndComputeJournal()

	meta, err := ledger.GetInterchainMeta(1)
	require.Equal(t, storage.ErrorNotFound, err)
	require.Nil(t, meta)

	ledger.PersistBlockData(genBlockData(1, accounts, journal))
	require.Equal(t, uint64(1), ledger.Version())

	meta, err = ledger.GetInterchainMeta(1)
	require.Nil(t, err)
	require.Equal(t, 0, len(meta.Counter))

	meta = &pb.InterchainMeta{
		Counter: make(map[string]*pb.Uint64Slice),
		L2Roots: make([]types.Hash, 0),
	}
	meta.Counter["a"] = &pb.Uint64Slice{}
	meta.L2Roots = append(meta.L2Roots, [32]byte{})
	batch := ledger.blockchainStore.NewBatch()
	err = ledger.persistInterChainMeta(batch, meta, 2)
	require.Nil(t, err)
	batch.Commit()

	meta2, err := ledger.GetInterchainMeta(2)
	require.Nil(t, err)
	require.Equal(t, len(meta.Counter), len(meta2.Counter))
	require.Equal(t, meta.Counter["a"], meta2.Counter["a"])
	require.Equal(t, len(meta.L2Roots), len(meta2.Counter))
}

func genBlockData(height uint64, accounts map[string]*Account, journal *BlockJournal) *BlockData {
	return &BlockData{
		Block: &pb.Block{
			BlockHeader: &pb.BlockHeader{
				Number: height,
			},
			BlockHash:    sha256.Sum256([]byte{1}),
			Transactions: []*pb.Transaction{{}},
		},
		Receipts:       nil,
		Accounts:       accounts,
		Journal:        journal,
		InterchainMeta: &pb.InterchainMeta{},
	}
}
