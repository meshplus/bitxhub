package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"testing"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/stretchr/testify/assert"
)

func TestLedger_Commit(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)

	// create an account
	account := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{100}, 20))

	ledger.SetState(account, []byte("a"), []byte("b"))
	hash, err := ledger.Commit(1)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), ledger.Version())
	assert.Equal(t, "0xe5ace5cd035b4c3d9d73a3f4a4a64e6e306010c75c35558283847c7c6473d66c", hash.Hex())

	hash, err = ledger.Commit(2)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), ledger.Version())
	assert.Equal(t, "0x4204720214cb812d802b2075c5fed85cd5dfe8a6065627489b6296108f0fedc2", hash.Hex())

	ledger.SetState(account, []byte("a"), []byte("3"))
	ledger.SetState(account, []byte("a"), []byte("2"))
	hash, err = ledger.Commit(3)
	assert.Nil(t, err)
	assert.Equal(t, uint64(3), ledger.Version())
	assert.Equal(t, "0xf08cc4b2da3f277202dc50a094ff2021300375915c14894a53fe02540feb3411", hash.Hex())

	ledger.SetBalance(account, 100)
	hash, err = ledger.Commit(4)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), ledger.Version())
	assert.Equal(t, "0x8ef7f408372406532c7060045d77fb67d322cea7aa49afdc3a741f4f340dc6d5", hash.Hex())

	code := bytesutil.RightPadBytes([]byte{100}, 100)
	ledger.SetCode(account, code)
	ledger.SetState(account, []byte("b"), []byte("3"))
	ledger.SetState(account, []byte("c"), []byte("2"))
	hash, err = ledger.Commit(5)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), ledger.Version())
	assert.Equal(t, uint64(5), ledger.maxJnlHeight)

	minHeight, maxHeight := getJournalRange(ledger.ldb)
	journal := getBlockJournal(maxHeight, ledger.ldb)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), minHeight)
	assert.Equal(t, uint64(5), maxHeight)
	assert.Equal(t, hash, journal.ChangedHash)
	assert.Equal(t, 1, len(journal.Journals))
	entry := journal.Journals[0]
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
	ldg, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), ldg.maxJnlHeight)
	assert.Equal(t, hash, ldg.prevJnlHash)

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
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)

	// create an addr0
	addr0 := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{100}, 20))
	addr1 := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{101}, 20))

	hash0 := types.Hash{}
	assert.Equal(t, hash0, ledger.prevJnlHash)

	code := sha256.Sum256([]byte("code"))
	codeHash := sha256.Sum256(code[:])

	ledger.SetBalance(addr0, 1)
	ledger.SetCode(addr0, code[:])

	hash1, err := ledger.Commit(1)
	assert.Nil(t, err)

	ledger.SetBalance(addr0, 2)
	ledger.SetState(addr0, []byte("a"), []byte("2"))

	code1 := sha256.Sum256([]byte("code1"))
	codeHash1 := sha256.Sum256(code1[:])
	ledger.SetCode(addr0, code1[:])

	hash2, err := ledger.Commit(2)
	assert.Nil(t, err)

	ledger.SetBalance(addr1, 3)
	ledger.SetBalance(addr0, 4)
	ledger.SetState(addr0, []byte("a"), []byte("3"))
	ledger.SetState(addr0, []byte("b"), []byte("4"))

	hash3, err := ledger.Commit(3)
	assert.Nil(t, err)
	assert.Equal(t, hash3, ledger.prevJnlHash)

	err = ledger.Rollback(2)
	assert.Nil(t, err)
	assert.Equal(t, hash2, ledger.prevJnlHash)
	assert.Equal(t, uint64(1), ledger.minJnlHeight)
	assert.Equal(t, uint64(2), ledger.maxJnlHeight)

	account0 := ledger.GetAccount(addr0)
	assert.Equal(t, uint64(2), account0.GetBalance())
	assert.Equal(t, uint64(0), account0.GetNonce())
	assert.Equal(t, codeHash1[:], account0.CodeHash())
	assert.Equal(t, code1[:], account0.Code())
	ok, val := account0.GetState([]byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("2"), val)

	account1 := ledger.GetAccount(addr1)
	assert.Equal(t, uint64(0), account1.GetBalance())
	assert.Equal(t, uint64(0), account1.GetNonce())
	assert.Nil(t, account1.CodeHash())
	assert.Nil(t, account1.Code())

	ledger.Close()
	ledger, err = New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), ledger.minJnlHeight)
	assert.Equal(t, uint64(2), ledger.maxJnlHeight)

	err = ledger.Rollback(1)
	assert.Nil(t, err)
	assert.Equal(t, hash1, ledger.prevJnlHash)

	account0 = ledger.GetAccount(addr0)
	assert.Equal(t, uint64(1), account0.GetBalance())
	assert.Equal(t, uint64(0), account0.GetNonce())
	assert.Equal(t, codeHash[:], account0.CodeHash())
	assert.Equal(t, code[:], account0.Code())
	ok, _ = account0.GetState([]byte("a"))
	assert.False(t, ok)

	err = ledger.Rollback(0)
	assert.Equal(t, ErrorRollbackTooMuch, err)

}

func TestChainLedger_RemoveJournalsBeforeBlock(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_removeJournal")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)

	assert.Equal(t, uint64(0), ledger.minJnlHeight)
	assert.Equal(t, uint64(0), ledger.maxJnlHeight)

	_, _ = ledger.Commit(1)
	_, _ = ledger.Commit(2)
	_, _ = ledger.Commit(3)
	hash, _ := ledger.Commit(4)

	assert.Equal(t, uint64(1), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)

	minHeight, maxHeight := getJournalRange(ledger.ldb)
	journal := getBlockJournal(maxHeight, ledger.ldb)
	assert.Equal(t, uint64(1), minHeight)
	assert.Equal(t, uint64(4), maxHeight)
	assert.Equal(t, hash, journal.ChangedHash)

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
	assert.Equal(t, hash, journal.ChangedHash)

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
	assert.Equal(t, hash, ledger.prevJnlHash)

	minHeight, maxHeight = getJournalRange(ledger.ldb)
	journal = getBlockJournal(maxHeight, ledger.ldb)
	assert.Equal(t, uint64(4), minHeight)
	assert.Equal(t, uint64(4), maxHeight)
	assert.Equal(t, hash, journal.ChangedHash)

	ledger.Close()
	ledger, err = New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)
	assert.Equal(t, uint64(4), ledger.minJnlHeight)
	assert.Equal(t, uint64(4), ledger.maxJnlHeight)
	assert.Equal(t, hash, ledger.prevJnlHash)
}
