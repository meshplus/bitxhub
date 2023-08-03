package ledger

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/storage/pebble"
	"github.com/axiomesh/axiom-kit/types"
)

func TestAccount_GetState(t *testing.T) {
	repoRoot, err := os.MkdirTemp("", "ledger_commit")
	assert.Nil(t, err)

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
			accountCache, err := NewAccountCache()
			assert.Nil(t, err)
			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			ledger, err := New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, accountCache, log.NewWithModule("ChainLedger"))
			assert.Nil(t, err)

			addr := types.NewAddressByStr("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
			stateLedger := ledger.StateLedger.(*StateLedger)
			account := newAccount(stateLedger.ldb, stateLedger.accountCache, addr, newChanger())

			addr1 := account.GetAddress()
			assert.Equal(t, addr, addr1)

			account.SetState([]byte("a"), []byte("b"))
			ok, v := account.GetState([]byte("a"))
			assert.True(t, ok)
			assert.Equal(t, []byte("b"), v)

			ok, v = account.GetState([]byte("a"))
			assert.True(t, ok)
			assert.Equal(t, []byte("b"), v)

			account.SetState([]byte("a"), nil)
			ok, v = account.GetState([]byte("a"))
			assert.False(t, ok)
			assert.Nil(t, v)
			account.GetCommittedState([]byte("a"))
		})
	}
}

func TestAccount_AddState(t *testing.T) {}

func TestAccount_AccountBalance(t *testing.T) {
	repoRoot, err := os.MkdirTemp("", "ledger_commit")
	assert.Nil(t, err)

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
			accountCache, err := NewAccountCache()
			assert.Nil(t, err)
			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			ledger, err := New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, accountCache, log.NewWithModule("ChainLedger"))
			assert.Nil(t, err)

			addr := types.NewAddressByStr("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
			stateLedger := ledger.StateLedger.(*StateLedger)
			account := newAccount(stateLedger.ldb, stateLedger.accountCache, addr, newChanger())

			account.AddBalance(big.NewInt(1))
			account.SubBalance(big.NewInt(1))

			account.SubBalance(big.NewInt(0))

			account.setCodeAndHash([]byte{'1'})
			account.dirtyAccount = nil
			account.setBalance(big.NewInt(1))
		})
	}
}
