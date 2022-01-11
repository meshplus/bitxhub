package ledger

import (
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/hexutil"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/stretchr/testify/assert"
)

func TestAccount_GetState(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache, err := NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, accountCache, log.NewWithModule("ChainLedger"))
	assert.Nil(t, err)

	h := hexutil.Encode(bytesutil.LeftPadBytes([]byte{11}, 20))
	addr := types.NewAddressByStr(h)
	stateLedger := ledger.StateLedger.(*SimpleLedger)
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
}

func TestAccount_AddState(t *testing.T) {

}

func TestAccount_AccountBalance(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(filepath.Join(repoRoot, "storage"))
	assert.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	assert.Nil(t, err)

	accountCache, err := NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("account_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ledger, err := New(createMockRepo(t), blockStorage, ldb, blockFile, accountCache, log.NewWithModule("ChainLedger"))
	assert.Nil(t, err)

	h := hexutil.Encode(bytesutil.LeftPadBytes([]byte{11}, 20))
	addr := types.NewAddressByStr(h)
	stateLedger := ledger.StateLedger.(*SimpleLedger)
	account := newAccount(stateLedger.ldb, stateLedger.accountCache, addr, newChanger())

	account.AddBalance(big.NewInt(1))
	account.SubBalance(big.NewInt(1))

}
