package solo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const to = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

func TestNode_Start(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "node")
	defer os.RemoveAll(repoRoot)
	assert.Nil(t, err)

	// write config file for order module
	fileData, err := ioutil.ReadFile("./testdata/order.toml")
	require.Nil(t, err)
	err = ioutil.WriteFile(filepath.Join(repoRoot, "order.toml"), fileData, 0644)
	require.Nil(t, err)

	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	peers := make(map[uint64]*pb.VpInfo)
	mockPeermgr.EXPECT().Peers().Return(peers).AnyTimes()

	nodes := make(map[uint64]*pb.VpInfo)
	vpInfo := &pb.VpInfo{
		Id:      uint64(1),
		Account: types.NewAddressByStr("000000000000000000000000000000000000000a").String(),
	}
	nodes[1] = vpInfo

	order, err := NewNode(
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

	_ = order.Start()
	require.Nil(t, err)
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	require.Nil(t, err)

	from, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	tx := &pb.Transaction{
		From:      from,
		To:        types.NewAddressByStr(to),
		Timestamp: time.Now().UnixNano(),
		Nonce:     1,
	}
	tx.TransactionHash = tx.Hash()
	err = tx.Sign(privKey)
	require.Nil(t, err)

	for {
		time.Sleep(200 * time.Millisecond)
		err := order.Ready()
		if err == nil {
			break
		}
	}

	err = order.Prepare(tx)
	require.Nil(t, err)

	commitEvent := <-order.Commit()
	require.Equal(t, uint64(2), commitEvent.Block.BlockHeader.Number)
	require.Equal(t, 1, len(commitEvent.Block.Transactions))

	txHashList := make([]*types.Hash, 0)
	txHashList = append(txHashList, tx.TransactionHash)
	order.ReportState(commitEvent.Block.Height(), commitEvent.Block.BlockHash, txHashList)
	order.Stop()
}
