package solo

import (
	"math/big"
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
	defer node.Stop()

	nonce := node.GetPendingNonceByAccount("account1")
	ast.Equal(uint64(0), nonce)

	tx, err := types.GenerateEmptyTransactionAndSigner()
	require.Nil(t, err)
	ast.Equal(uint64(0), node.GetPendingNonceByAccount(tx.RbftGetFrom()))

	err = node.Prepare(tx)
	ast.Nil(err)
	ast.Equal(uint64(1), node.GetPendingNonceByAccount(tx.RbftGetFrom()))

	err = node.DelNode(uint64(1))
	ast.Nil(err)
}

func TestGetPendingTxByHash(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, false)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)
	defer node.Stop()

	tx, err := types.GenerateEmptyTransactionAndSigner()
	require.Nil(t, err)

	err = node.Prepare(tx)
	ast.Nil(err)
	tx1 := node.GetPendingTxByHash(tx.GetHash())
	ast.NotNil(tx1.GetPayload())
	ast.Equal(tx.GetPayload(), tx1.GetPayload())

	tx2 := node.GetPendingTxByHash(types.NewHashByStr("0x123"))
	ast.Nil(tx2)
}

func TestTimedBlock(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, true)
	ast.Nil(err)
	defer node.Stop()

	err = node.Start()
	ast.Nil(err)
	defer node.Stop()

	event := <-node.commitC
	ast.NotNil(event)
	ast.Equal(len(event.Block.Transactions), 0)
}

func TestNode_ReportState(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, false)
	ast.Nil(err)

	err = node.Start()
	ast.Nil(err)
	defer node.Stop()
	node.batchDigestM[10] = "test"
	node.ReportState(10, types.NewHashByStr("0x123"), []*types.Hash{})
	time.Sleep(10 * time.Millisecond)
	ast.Equal(0, len(node.batchDigestM))

	txList, signer := prepareMultiTx(t, 10)
	ast.Equal(10, len(txList))
	for _, tx := range txList {
		err = node.Prepare(tx)
		ast.Nil(err)
		// sleep to make sure the tx is generated to the batch
		time.Sleep(batchTimeout + 10*time.Millisecond)
	}
	t.Run("test pool full", func(t *testing.T) {
		ast.Equal(10, len(node.batchDigestM))
		tx11, err := types.GenerateTransactionWithSigner(uint64(11),
			types.NewAddressByStr("0xdAC17F958D2ee523a2206206994597C13D831ec7"), big.NewInt(0), nil, signer)

		ast.Nil(err)
		err = node.Prepare(tx11)
		ast.NotNil(err)
		ast.Contains(err.Error(), ErrPoolFull)
		time.Sleep(100 * time.Millisecond)
		ast.True(node.isPoolFull())
		ast.Equal(10, len(node.batchDigestM), "the pool should be full, tx11 is not add in mempool successfully")

		tx12, err := types.GenerateTransactionWithSigner(uint64(12),
			types.NewAddressByStr("0xdAC17F958D2ee523a2206206994597C13D831ec7"), big.NewInt(0), nil, signer)

		err = node.Prepare(tx12)
		ast.NotNil(err)
		ast.Contains(err.Error(), ErrPoolFull)
	})

	ast.NotNil(node.mempool.GetPendingTxByHash(txList[9].RbftGetTxHash()), "tx10 should be in mempool")
	// trigger the report state
	node.ReportState(10, types.NewHashByStr("0x123"), []*types.Hash{txList[9].GetHash()})
	time.Sleep(50 * time.Millisecond)
	ast.Equal(0, len(node.batchDigestM))
	ast.Nil(node.mempool.GetPendingTxByHash(txList[9].RbftGetTxHash()), "tx10 should be removed from mempool")

}

func prepareMultiTx(t *testing.T, count int) ([]*types.Transaction, *types.Signer) {
	signer, err := types.GenerateSigner()
	require.Nil(t, err)
	txList := make([]*types.Transaction, 0)
	for i := 0; i < count; i++ {
		tx, err := types.GenerateTransactionWithSigner(uint64(i),
			types.NewAddressByStr("0xdAC17F958D2ee523a2206206994597C13D831ec7"), big.NewInt(0), nil, signer)
		require.Nil(t, err)
		txList = append(txList, tx)
	}
	return txList, signer
}

func TestNode_RemoveTxFromPool(t *testing.T) {
	ast := assert.New(t)
	node, err := mockSoloNode(t, false)
	ast.Nil(err)

	err = node.Start()
	ast.Nil(err)
	defer node.Stop()

	txList, _ := prepareMultiTx(t, 10)
	// remove the first tx
	txList = txList[1:]
	ast.Equal(9, len(txList))
	for _, tx := range txList {
		err = node.Prepare(tx)
		ast.Nil(err)
	}
	// lack nonce 0, so the txs will not be generated to the batch
	ast.Equal(0, len(node.batchDigestM))
	ast.NotNil(node.mempool.GetPendingTxByHash(txList[8].RbftGetTxHash()), "tx9 should be in mempool")
	// sleep to make sure trigger the remove tx from pool
	time.Sleep(2*removeTxTimeout + 50*time.Millisecond)

	ast.Nil(node.mempool.GetPendingTxByHash(txList[8].RbftGetTxHash()), "tx9 should be removed from mempool")
}
