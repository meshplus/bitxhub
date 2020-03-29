package ledger

import (
	"io/ioutil"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/stretchr/testify/require"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/hexutil"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/stretchr/testify/assert"
	db "github.com/tendermint/tm-db"
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

	dir, err := ioutil.TempDir("", "")
	require.Nil(t, err)
	ldb, err := db.NewGoLevelDB("ledger", dir)
	assert.Nil(t, err)

	h := hexutil.Encode(bytesutil.LeftPadBytes([]byte{11}, 20))
	addr := types.String2Address(h)
	account := newAccount(ledger, ldb, addr)

	account.SetState([]byte("a"), []byte("b"))
	ok, v := account.GetState([]byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("b"), v)

	// save into db
	hash, err := account.Commit()
	assert.Nil(t, err)
	assert.Equal(t, "0x07babb4c717e4c854558e806f6c0c82344009b234f1733b638c14730f625e8d1", hash.Hex())

	// recreate account
	account2 := newAccount(ledger, ldb, addr)
	ok2, v2 := account2.GetState([]byte("a"))
	assert.True(t, ok2)
	assert.Equal(t, []byte("b"), v2)
}

func TestAccount_Commit(t *testing.T) {
	_, err := asym.GenerateKey(asym.ECDSASecp256r1)
	assert.Nil(t, err)
	repoRoot, err := ioutil.TempDir("", "ledger_commit")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("ChainLedger"))
	assert.Nil(t, err)

	dir, err := ioutil.TempDir("", "")
	require.Nil(t, err)
	ldb, err := db.NewGoLevelDB("ledger", dir)
	assert.Nil(t, err)

	h := hexutil.Encode(bytesutil.LeftPadBytes([]byte{11}, 20))
	addr := types.String2Address(h)
	account := newAccount(ledger, ldb, addr)

	account.SetState([]byte("alice"), []byte("bob"))

	// save into db
	hash, err := account.Commit()
	assert.Nil(t, err)
	assert.Equal(t, "0x73f56a5593a5ab27d7db1c91bd1c78d26ebd3c5a235a226428f57d4abaa49fab", hash.Hex())

	account.SetState([]byte("a"), []byte("b"))
	hash, err = account.Commit()
	assert.Nil(t, err)
	assert.Equal(t, "0x8ebbcd9523cf21e5b284325542f8e7dcf588f3cddd2e94342a2acdbfb9dd3358", hash.Hex())
}
