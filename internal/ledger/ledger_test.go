package ledger

import (
	"io/ioutil"
	"testing"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
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
	hash, err := ledger.Commit()
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), ledger.Version())
	assert.Equal(t, "0x711ba7e0fbb4011960870c9c98fdf930809a384243f580aae3b5d9d0d3f19f50", hash.Hex())

	hash, err = ledger.Commit()
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), ledger.Version())
	assert.Equal(t, "0x711ba7e0fbb4011960870c9c98fdf930809a384243f580aae3b5d9d0d3f19f50", hash.Hex())

	ledger.SetState(account, []byte("a"), []byte("3"))
	ledger.SetState(account, []byte("a"), []byte("2"))
	hash, err = ledger.Commit()
	assert.Nil(t, err)
	assert.Equal(t, uint64(3), ledger.Version())
	assert.Equal(t, "0x102f75930e478956e0cc1ae4f79d24723d893bdf40e65186b6bb109c6f17131e", hash.Hex())

	ledger.Close()

	// load ChainLedger from db
	ldg, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)

	ok, value := ldg.GetState(account, []byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("2"), value)
	ver := ldg.Version()
	assert.Equal(t, uint64(3), ver)
	assert.Equal(t, "0x102f75930e478956e0cc1ae4f79d24723d893bdf40e65186b6bb109c6f17131e", hash.Hex())
}

func TestLedger_Rollback(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "ledger_rollback")
	assert.Nil(t, err)
	blockStorage, err := leveldb.New(repoRoot)
	assert.Nil(t, err)
	ledger, err := New(repoRoot, blockStorage, log.NewWithModule("executor"))
	assert.Nil(t, err)

	// create an account
	account := types.Bytes2Address(bytesutil.LeftPadBytes([]byte{100}, 20))

	ledger.SetState(account, []byte("a"), []byte("b"))
	_, err = ledger.Commit()
	assert.Nil(t, err)
	ledger.SetState(account, []byte("a"), []byte("c"))
	_, err = ledger.Commit()
	assert.Nil(t, err)
	ledger.SetState(account, []byte("a"), []byte("d"))
	_, err = ledger.Commit()
	assert.Nil(t, err)

	block1 := &pb.Block{BlockHeader: &pb.BlockHeader{Number: 1}}
	block1.BlockHash = block1.Hash()
	block2 := &pb.Block{BlockHeader: &pb.BlockHeader{Number: 2}}
	block2.BlockHash = block2.Hash()
	block3 := &pb.Block{BlockHeader: &pb.BlockHeader{Number: 3}}
	block3.BlockHash = block3.Hash()
	err = ledger.PutBlock(1, block1)
	assert.Nil(t, err)
	err = ledger.PutBlock(2, block2)
	assert.Nil(t, err)
	err = ledger.PutBlock(3, block3)
	assert.Nil(t, err)

	ledger.UpdateChainMeta(&pb.ChainMeta{
		Height:    3,
		BlockHash: types.Hash{},
	})

	err = ledger.Rollback(1)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), ledger.Version())
	err = ledger.Rollback(2)
	assert.Equal(t, ErrorRollbackTohigherNumber, err)

	ok, value := ledger.GetState(account, []byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("b"), value)

	ledger.SetState(account, []byte("a"), []byte("c"))
	_, err = ledger.Commit()
	assert.Nil(t, err)
	ledger.SetState(account, []byte("a"), []byte("d"))
	_, err = ledger.Commit()
	assert.Nil(t, err)
	err = ledger.PutBlock(2, block2)
	assert.Nil(t, err)
	err = ledger.PutBlock(3, block3)
	assert.Nil(t, err)

	ok, value = ledger.GetState(account, []byte("a"))
	assert.True(t, ok)
	assert.Equal(t, []byte("d"), value)
	assert.Equal(t, uint64(3), ledger.Version())
}
