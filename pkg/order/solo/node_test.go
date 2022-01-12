package solo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	to          = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	ErrorConfig = "Illegal parameter"
)

func TestNode_Start(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "node")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Logf("close file err")
		}
	}(repoRoot)
	assert.Nil(t, err)
	// write config file for order module
	fileData, err := ioutil.ReadFile("./testdata/order.toml")
	require.Nil(t, err)
	err = ioutil.WriteFile(filepath.Join(repoRoot, "order.toml"), fileData, 0644)
	require.Nil(t, err)

	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	peers := make(map[uint64]*pb.VpInfo)
	mockPeermgr.EXPECT().OrderPeers().Return(peers).AnyTimes()

	nodes := make(map[uint64]*pb.VpInfo)
	vpInfo := &pb.VpInfo{
		Id:      uint64(1),
		Account: types.NewAddressByStr("000000000000000000000000000000000000000a").String(),
	}
	nodes[1] = vpInfo

	solo, err := NewNode(
		order.WithRepoRoot(repoRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithLogger(log.NewWithModule("consensus")),
		order.WithApplied(1),
		order.WithPeerManager(mockPeermgr),
		order.WithID(1),
		order.WithNodes(nodes),
		order.WithApplied(1),
		order.WithGetAccountNonceFunc(func(address *types.Address) uint64 {
			return 0
		}),
	)
	require.Nil(t, err)

	_ = solo.Start()
	require.Nil(t, err)

	var msg []byte
	require.Nil(t, solo.Step(msg))
	require.Equal(t, uint64(1), solo.Quorum())

	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	tx := &pb.BxhTransaction{
		From:      from,
		To:        types.NewAddressByStr(to),
		Timestamp: time.Now().UnixNano(),
		Nonce:     0,
	}
	tx.TransactionHash = tx.Hash()
	err = tx.Sign(privKey)
	require.Nil(t, err)

	for {
		time.Sleep(200 * time.Millisecond)
		err := solo.Ready()
		if err == nil {
			break
		}
	}

	err = solo.Prepare(tx)
	require.Nil(t, err)

	commitEvent := <-solo.Commit()
	require.Equal(t, uint64(2), commitEvent.Block.BlockHeader.Number)
	require.Equal(t, 1, len(commitEvent.Block.Transactions.Transactions))

	txHashList := make([]*types.Hash, 0)
	txHashList = append(txHashList, tx.TransactionHash)
	solo.ReportState(commitEvent.Block.Height(), commitEvent.Block.BlockHash, txHashList)
	solo.Stop()
}

func TestGetPendingNonceByAccount(t *testing.T) {
	ast := assert.New(t)
	defer func() {
		err := os.RemoveAll("./testdata/storage")
		if err != nil {
			t.Logf("close file err")
		}
	}()
	node, err := mockSoloNode(t, false)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)
	nonce := node.GetPendingNonceByAccount("account1")
	ast.Equal(uint64(0), nonce)
	err = node.DelNode(uint64(1))
}

func TestGetpendingTxByHash(t *testing.T) {
	ast := assert.New(t)
	defer func() {
		err := os.RemoveAll("./testdata/storage")
		if err != nil {
			t.Logf("close file err")
		}
	}()
	node, err := mockSoloNode(t, false)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)

	tx := generateTx()
	err = node.Prepare(tx)
	ast.Nil(err)
	time.Sleep(200 * time.Millisecond)
	ast.Equal(tx, node.GetPendingTxByHash(tx.GetHash()))
}

func TestWrongConfig(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "node")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Logf("close file err")
		}
	}(repoRoot)
	assert.Nil(t, err)

	// test read wrong config from order,toml
	fileData, err := ioutil.ReadFile("./testdata/wrongOrder.toml")
	require.Nil(t, err)
	err = ioutil.WriteFile(filepath.Join(repoRoot, "order.toml"), fileData, 0644)
	require.Nil(t, err)
	_, err = NewNode(
		order.WithRepoRoot(repoRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithLogger(log.NewWithModule("consensus")),
	)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), ErrorConfig)
}

func TestTimedBlock(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, true)
	ast.Nil(err)
	defer node.Stop()
	defer func() {
		err := os.RemoveAll("./testdata/storage")
		if err != nil {
			t.Logf("close file err")
		}
	}()

	err = node.Start()
	ast.Nil(err)
	event := <-node.commitC
	ast.NotNil(event)
	ast.Equal(len(event.Block.Transactions.Transactions), 0)

}
