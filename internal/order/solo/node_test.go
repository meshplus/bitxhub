package solo

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/peermgr/mock_peermgr"
	"github.com/axiomesh/axiom/pkg/repo"
)

func TestNode_Start(t *testing.T) {
	repoRoot := t.TempDir()
	r, err := repo.Load(repoRoot)
	require.Nil(t, err)

	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	peers := make(map[uint64]*types.VpInfo)
	mockPeermgr.EXPECT().OrderPeers().Return(peers).AnyTimes()

	nodes := make(map[uint64]*types.VpInfo)
	vpInfo := &types.VpInfo{
		Id:      uint64(1),
		Account: types.NewAddressByStr("000000000000000000000000000000000000000a").String(),
	}
	nodes[1] = vpInfo

	solo, err := NewNode(
		order.WithConfig(r.OrderConfig),
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

	err = solo.Start()
	require.Nil(t, err)

	var msg []byte
	require.Nil(t, solo.Step(msg))
	require.Equal(t, uint64(1), solo.Quorum())

	tx, err := types.GenerateEmptyTransactionAndSigner()
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
	require.Equal(t, 1, len(commitEvent.Block.Transactions))
	blockHash := commitEvent.Block.Hash()

	txHashList := make([]*types.Hash, 0)
	txHashList = append(txHashList, tx.GetHash())
	solo.ReportState(commitEvent.Block.Height(), blockHash, txHashList)
	solo.Stop()
}

func TestGetPendingNonceByAccount(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, false)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)
	nonce := node.GetPendingNonceByAccount("account1")
	ast.Equal(uint64(0), nonce)
	err = node.DelNode(uint64(1))
	ast.Nil(err)
}

func TestGetpendingTxByHash(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, false)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)

	tx, err := types.GenerateEmptyTransactionAndSigner()
	require.Nil(t, err)

	err = node.Prepare(tx)
	ast.Nil(err)
	ast.Nil(node.GetPendingTxByHash(tx.GetHash()))
}

func TestTimedBlock(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, true)
	ast.Nil(err)
	defer node.Stop()

	err = node.Start()
	ast.Nil(err)
	event := <-node.commitC
	ast.NotNil(event)
	ast.Equal(len(event.Block.Transactions), 0)
}
