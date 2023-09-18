package ledger

import (
	"math/big"
	"path/filepath"
	"sync"
	"testing"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/stretchr/testify/assert"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/storage/pebble"
	"github.com/axiomesh/axiom-kit/types"
)

func TestAccountCache_clear(t *testing.T) {
	accountCache, err := NewAccountCache()
	assert.Nil(t, err)

	code := []byte{1}
	addr := &types.Address{}
	err = accountCache.add(map[string]IAccount{
		addr.String(): &SimpleAccount{
			Addr:           &types.Address{},
			originAccount:  &InnerAccount{},
			dirtyAccount:   &InnerAccount{},
			originState:    cmap.New[[]byte](),
			dirtyState:     cmap.New[[]byte](),
			originCode:     nil,
			dirtyCode:      code,
			dirtyStateHash: &types.Hash{},
			ldb:            nil,
			cache:          nil,
			lock:           sync.RWMutex{},
			changer:        &stateChanger{},
			suicided:       false,
		},
	})
	assert.Nil(t, err)

	c, ok := accountCache.getCode(addr)
	assert.True(t, ok)
	assert.EqualValues(t, code, c)

	accountCache.clear()
	_, ok = accountCache.getCode(addr)
	assert.False(t, ok)
}

func TestAccount_GetState(t *testing.T) {
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
			accountCache, err := NewAccountCache()
			assert.Nil(t, err)
			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			ledger, err := New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, accountCache, log.NewWithModule("ChainLedgerImpl"))
			assert.Nil(t, err)

			addr := types.NewAddressByStr("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
			stateLedger := ledger.StateLedger.(*StateLedgerImpl)
			account := NewAccount(stateLedger.ldb, stateLedger.accountCache, addr, NewChanger())

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

func TestAccount_AccountBalance(t *testing.T) {
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
			accountCache, err := NewAccountCache()
			assert.Nil(t, err)
			logger := log.NewWithModule("account_test")
			blockFile, err := blockfile.NewBlockFile(filepath.Join(repoRoot, name), logger)
			assert.Nil(t, err)
			ledger, err := New(createMockRepo(t), tc.blockStorage, tc.stateStorage, blockFile, accountCache, log.NewWithModule("ChainLedgerImpl"))
			assert.Nil(t, err)

			addr := types.NewAddressByStr("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
			stateLedger := ledger.StateLedger.(*StateLedgerImpl)
			account := NewAccount(stateLedger.ldb, stateLedger.accountCache, addr, NewChanger())

			account.AddBalance(big.NewInt(1))
			account.SubBalance(big.NewInt(1))

			account.SubBalance(big.NewInt(0))

			account.setCodeAndHash([]byte{'1'})
			account.dirtyAccount = nil
			account.setBalance(big.NewInt(1))
		})
	}
}
