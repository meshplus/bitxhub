package ledger

import (
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"io/ioutil"
	"testing"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/hexutil"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/stretchr/testify/assert"
)

func TestAccount_GetState(t *testing.T) {
	_, err := asym.GenerateKey(asym.ECDSASecp256r1)
	assert.Nil(t, err)
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("ChainLedger"))
	assert.Nil(t, err)

	ldb := ledger.ldb

	h := hexutil.Encode(bytesutil.LeftPadBytes([]byte{11}, 20))
	addr := types.String2Address(h)
	account := newAccount(ldb, addr)

	account.SetState([]byte("a"), []byte("b"))
	ok, v := account.GetState([]byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("b"), v)

	account.SetState([]byte("a"), nil)
	ok, v = account.GetState([]byte("a"))
	assert.False(t, ok)
}
