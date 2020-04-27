package ledger

import (
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
	assert.Equal(t, uint64(4), ledger.height)
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
	assert.Equal(t, uint64(5), ledger.height)

	height, journal, err := getLatestJournal(ledger.ldb)
	assert.Nil(t, err)
	assert.Equal(t, uint64(5), height)
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
	assert.Equal(t, uint64(5), ldg.height)
	assert.Equal(t, hash, ldg.prevJournalHash)

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
