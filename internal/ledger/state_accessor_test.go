package ledger

import (
	"io/ioutil"
	"testing"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/stretchr/testify/assert"
)

func TestChainLedger_QueryByPrefix(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)

	addr := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{1}, 20))
	key0 := []byte{100, 100}
	key1 := []byte{100, 101}
	key2 := []byte{100, 102}

	ledger.SetState(addr, key0, []byte("0"))
	ledger.SetState(addr, key1, []byte("1"))
	ledger.SetState(addr, key2, []byte("2"))

	ledger.Commit(1)

	ok, vals := ledger.QueryByPrefix(addr, string([]byte{100}))
	assert.True(t, ok)
	assert.Equal(t, 3, len(vals))
	assert.Equal(t, []byte("0"), vals[0])
	assert.Equal(t, []byte("1"), vals[1])
	assert.Equal(t, []byte("2"), vals[2])
}

func TestChainLedger_GetAccount(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
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

	ledger.Commit(1)

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
	ledger.Commit(2)

	ledger.SetState(addr, key0, val0)
	ledger.SetState(addr, key0, val1)
	ledger.Commit(3)
	ok, val := ledger.GetState(addr, key0)
	assert.True(t, ok)
	assert.Equal(t, val1, val)
}

func TestChainLedger_GetCode(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)

	addr := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{1}, 20))
	key0 := []byte{100, 100}
	key1 := []byte{100, 101}
	key2 := []byte{100, 102}

	ledger.SetState(addr, key0, []byte("0"))
	ledger.SetState(addr, key1, []byte("1"))
	ledger.SetState(addr, key2, []byte("2"))

	code := bytesutil.LeftPadBytes([]byte{10}, 120)
	ledger.SetCode(addr, code)

	ledger.Commit(1)

	vals := ledger.GetCode(addr)
	assert.Equal(t, code, vals)

	ledger.Commit(2)

	vals = ledger.GetCode(addr)
	assert.Equal(t, code, vals)
}
