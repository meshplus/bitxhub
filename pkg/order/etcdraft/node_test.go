package etcdraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/etcd/raft/raftpb"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_Start(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "node")
	assert.Nil(t, err)
	defer os.RemoveAll(repoRoot)

	var ID uint64 = 1
	nodes := make(map[uint64]*pb.VpInfo)
	vpInfo := &pb.VpInfo{
		Id:      ID,
		Account: types.NewAddressByStr("000000000000000000000000000000000000000a").String(),
	}
	nodes[ID] = vpInfo
	fileData, err := ioutil.ReadFile("./testdata/order.toml")
	require.Nil(t, err)
	err = ioutil.WriteFile(filepath.Join(repoRoot, "order.toml"), fileData, 0644)
	require.Nil(t, err)

	mockCtl := gomock.NewController(t)

	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	peers := make(map[uint64]*pb.VpInfo)
	otherPeers := make(map[uint64]*peer.AddrInfo, 5)
	cntPeers := uint64(0)
	mockPeermgr.EXPECT().OrderPeers().Return(peers).AnyTimes()
	mockPeermgr.EXPECT().OtherPeers().Return(otherPeers).AnyTimes()
	mockPeermgr.EXPECT().Broadcast(gomock.Any()).AnyTimes()
	mockPeermgr.EXPECT().CountConnectedPeers().Return(cntPeers).AnyTimes()

	order, err := NewNode(
		order.WithRepoRoot(repoRoot),
		order.WithID(ID),
		order.WithNodes(nodes),
		order.WithPeerManager(mockPeermgr),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithLogger(log.NewWithModule("consensus")),
		order.WithApplied(1),
		order.WithGetAccountNonceFunc(func(address *types.Address) uint64 {
			return 0
		}),
	)
	require.Nil(t, err)

	err = order.Start()
	require.Nil(t, err)

	for {
		time.Sleep(200 * time.Millisecond)
		err := order.Ready()
		if err == nil {
			break
		}
	}
	tx := generateTx()
	err = order.Prepare(tx)
	require.Nil(t, err)

	require.Equal(t, tx, order.GetPendingTxByHash(tx.GetHash()))

	commitEvent := <-order.Commit()
	require.Equal(t, uint64(2), commitEvent.Block.BlockHeader.Number)
	require.Equal(t, 1, len(commitEvent.Block.Transactions.Transactions))

	order.Stop()
}

func TestMulti_Node_Start(t *testing.T) {
	peerCnt := 4
	swarms, nodes := newSwarms(t, peerCnt, true)
	defer stopSwarms(t, swarms)

	repoRoot, err := ioutil.TempDir("", "nodes")
	defer os.RemoveAll(repoRoot)

	fileData, err := ioutil.ReadFile("../../../config/order.toml")
	require.Nil(t, err)

	orders := make([]order.Order, 0)
	for i := 0; i < peerCnt; i++ {
		nodePath := fmt.Sprintf("node%d", i)
		nodeRepo := filepath.Join(repoRoot, nodePath)
		err := os.Mkdir(nodeRepo, 0744)
		require.Nil(t, err)
		orderPath := filepath.Join(nodeRepo, "order.toml")
		err = ioutil.WriteFile(orderPath, fileData, 0744)
		require.Nil(t, err)

		ID := i + 1
		order, err := NewNode(
			order.WithRepoRoot(nodeRepo),
			order.WithID(uint64(ID)),
			order.WithNodes(nodes),
			order.WithPeerManager(swarms[i]),
			order.WithStoragePath(repo.GetStoragePath(nodeRepo, "order")),
			order.WithLogger(log.NewWithModule("consensus")),
			order.WithGetBlockByHeightFunc(nil),
			order.WithApplied(1),
			order.WithGetAccountNonceFunc(func(address *types.Address) uint64 {
				return 0
			}),
		)
		require.Nil(t, err)
		err = order.Start()
		require.Nil(t, err)
		orders = append(orders, order)
		go listen(t, order, swarms[i])
	}

	for {
		time.Sleep(200 * time.Millisecond)
		err := orders[0].Ready()
		if err == nil {
			break
		}
	}
	tx := generateTx()
	err = orders[0].Prepare(tx)
	require.Nil(t, err)
	for i := 0; i < len(orders); i++ {
		commitEvent := <-orders[i].Commit()
		require.Equal(t, uint64(2), commitEvent.Block.BlockHeader.Number)
		require.Equal(t, 1, len(commitEvent.Block.Transactions.Transactions))
	}
}

func TestMulti_Node_Start_Without_Cert_Verification(t *testing.T) {
	peerCnt := 4
	swarms, nodes := newSwarms(t, peerCnt, false)

	repoRoot, err := ioutil.TempDir("", "nodes")
	defer os.RemoveAll(repoRoot)

	fileData, err := ioutil.ReadFile("../../../config/order.toml")
	require.Nil(t, err)

	orders := make([]order.Order, 0)
	for i := 0; i < peerCnt; i++ {
		nodePath := fmt.Sprintf("node%d", i)
		nodeRepo := filepath.Join(repoRoot, nodePath)
		err := os.Mkdir(nodeRepo, 0744)
		require.Nil(t, err)
		orderPath := filepath.Join(nodeRepo, "order.toml")
		err = ioutil.WriteFile(orderPath, fileData, 0744)
		require.Nil(t, err)

		ID := i + 1
		order, err := NewNode(
			order.WithRepoRoot(nodeRepo),
			order.WithID(uint64(ID)),
			order.WithNodes(nodes),
			order.WithPeerManager(swarms[i]),
			order.WithStoragePath(repo.GetStoragePath(nodeRepo, "order")),
			order.WithLogger(log.NewWithModule("consensus")),
			order.WithGetBlockByHeightFunc(nil),
			order.WithApplied(1),
			order.WithGetAccountNonceFunc(func(address *types.Address) uint64 {
				return 0
			}),
		)
		require.Nil(t, err)
		err = order.Start()
		require.Nil(t, err)
		orders = append(orders, order)
		go listen(t, order, swarms[i])
	}

	for {
		time.Sleep(200 * time.Millisecond)
		err := orders[0].Ready()
		if err == nil {
			break
		}
	}
	tx := generateTx()
	err = orders[0].Prepare(tx)
	require.Nil(t, err)
	for i := 0; i < len(orders); i++ {
		commitEvent := <-orders[i].Commit()
		require.Equal(t, uint64(2), commitEvent.Block.BlockHeader.Number)
		require.Equal(t, 1, len(commitEvent.Block.Transactions.Transactions))
	}
}

func TestReportState(t *testing.T) {
	ast := assert.New(t)
	defer os.RemoveAll("./testdata/storage")
	node, err := mockRaftNode(t)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)
	blockHash := &types.Hash{
		RawHash: [types.HashLength]byte{0},
	}
	txHashList := make([]*types.Hash, 0)
	txHash := &types.Hash{
		RawHash: [types.HashLength]byte{0},
	}
	txHashList = append(txHashList, txHash)
	node.blockAppliedIndex.Store(uint64(10), uint64(10))
	node.ReportState(uint64(10), blockHash, txHashList)
	appliedIndex, ok := node.blockAppliedIndex.Load(uint64(10))
	ast.Equal(true, ok)
	ast.Equal(uint64(10), appliedIndex.(uint64))
	nonce := node.GetPendingNonceByAccount("account1")
	ast.Equal(uint64(0), nonce)
	err = node.DelNode(uint64(1))
}

func TestGetPendingNonceByAccount(t *testing.T) {
	ast := assert.New(t)
	defer os.RemoveAll("./testdata/storage")
	node, err := mockRaftNode(t)
	ast.Nil(err)
	err = node.Start()
	ast.Nil(err)
	nonce := node.GetPendingNonceByAccount("account1")
	ast.Equal(uint64(0), nonce)
	err = node.DelNode(uint64(1))
}

func TestRun(t *testing.T) {
	ast := assert.New(t)
	defer os.RemoveAll("./testdata/storage")
	node, err := mockRaftNode(t)
	ast.Nil(err)
	node.checkInterval = 200 * time.Millisecond
	err = node.Start()
	ast.Nil(err)
	ast.Nil(node.checkQuorum())
	// test confChangeC
	node.confChangeC <- raftpb.ConfChange{ID: uint64(2)}
	// test rebroadcastTicker
	txs := make([]pb.Transaction, 0)
	txs = append(txs, constructTx(1))
	node.mempool.ProcessTransactions(txs, false, true)
	time.Sleep(250 * time.Millisecond)
}
