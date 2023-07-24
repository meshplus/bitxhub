package rbft

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	rbft "github.com/hyperchain/go-hpc-rbft"
	"github.com/hyperchain/go-hpc-rbft/common/consensus"
	rbfttypes "github.com/hyperchain/go-hpc-rbft/types"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/rbft/adaptor"
	"github.com/meshplus/bitxhub/pkg/order/rbft/testutil"
	ethtypes "github.com/meshplus/eth-kit/types"
	"github.com/stretchr/testify/assert"
)

func mockNode(ctrl *gomock.Controller, t *testing.T) *Node {
	order, err := newNode(withID(), withIsNew(), withRepoRoot(), withStoragePath(t),
		withLogger(), withNodes(), withApplied(), withDigest(), withPeerManager(ctrl), WithGetAccountNonceFunc())
	assert.Nil(t, err)
	return order
}

func withID() order.Option {
	return func(config *order.Config) {
		config.ID = uint64(1)
	}
}

func withIsNew() order.Option {
	return func(config *order.Config) {
		config.IsNew = false
	}
}

func withRepoRoot() order.Option {
	return func(config *order.Config) {
		config.RepoRoot = "./testdata/"
	}
}

func withStoragePath(t *testing.T) order.Option {
	return func(config *order.Config) {
		config.StoragePath = t.TempDir()
	}
}

func withLogger() order.Option {
	return func(config *order.Config) {
		config.Logger = log.NewWithModule("order")
	}
}

func withPeerManager(ctrl *gomock.Controller) order.Option {
	return func(config *order.Config) {
		config.PeerMgr = testutil.MockMiniPeerManager(ctrl)
	}
}

func WithGetAccountNonceFunc() order.Option {
	return func(config *order.Config) {
		config.GetAccountNonce = func(address *types.Address) uint64 {
			return 0
		}
	}
}

func withNodes() order.Option {
	nodes := make(map[uint64]*pb.VpInfo)
	nodes[1] = &pb.VpInfo{Id: uint64(1)}
	nodes[2] = &pb.VpInfo{Id: uint64(2)}
	nodes[3] = &pb.VpInfo{Id: uint64(3)}
	return func(config *order.Config) {
		config.Nodes = nodes
	}
}

func withApplied() order.Option {
	return func(config *order.Config) {
		config.Applied = uint64(1)
	}
}

func withDigest() order.Option {
	return func(config *order.Config) {
		config.Digest = "digest"
	}
}

func TestPrepare(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	order := mockNode(ctrl, t)
	tx1 := &ethtypes.EthTransaction{
		Inner: &ethtypes.DynamicFeeTx{},
		Time:  time.Now(),
	}
	err := order.Prepare(tx1)
	ast.NotNil(err)
	ast.Equal("system is in pending state", err.Error())

	err = order.Start()
	ast.Nil(err)
	err = order.Prepare(tx1)
	ast.Nil(err)

	// TODO: impl it
	// pendingNonce := order.GetPendingNonceByAccount(tx1.GetFrom().String())
	pendingNonce := order.GetPendingNonceByAccount("")
	ast.Equal(uint64(0), pendingNonce)
}

func TestStop(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)

	//test start
	err := node.Start()
	ast.Nil(err)
	ast.Nil(node.checkQuorum())

	node.stack.ReadyC <- &adaptor.Ready{
		Height: uint64(2),
	}
	block := <-node.Commit()
	ast.Equal(uint64(2), block.Block.Height())

	// test stop
	node.Stop()
	time.Sleep(1 * time.Second)
	_, ok := <-node.txCache.close
	ast.Equal(false, ok)
}

func TestReadConfig(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	logger := log.NewWithModule("order")
	rbftConf, txpoolConfig, err := generateRbftConfig("./testdata/", testutil.MockOrderConfig(logger, ctrl, t))
	ast.Nil(err)
	rbftConf.Logger.Critical()
	rbftConf.Logger.Criticalf("test critical")
	rbftConf.Logger.Notice()
	rbftConf.Logger.Noticef("test critical")
	ast.Equal(25, rbftConf.SetSize)
	ast.Equal(500, txpoolConfig.BatchSize)
	ast.Equal(50000, txpoolConfig.PoolSize)
	ast.Equal(500*time.Millisecond, rbftConf.BatchTimeout)
	ast.Equal(3*time.Minute, rbftConf.CheckPoolTimeout)
	ast.Equal(5*time.Minute, txpoolConfig.ToleranceTime)
}

func TestStep(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	err := node.Step([]byte("test"))
	ast.NotNil(err)
	msg := &consensus.ConsensusMessage{}
	msgBytes, _ := msg.Marshal()
	err = node.Step(msgBytes)
	ast.Nil(err)
}

func TestDelNode(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	err := node.DelNode(uint64(2))
	ast.Error(err)
}

func TestReportState(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)

	block := testutil.ConstructBlock("blockHash", uint64(20))
	node.stack.StateUpdating = true
	node.stack.StateUpdateHeight = 20
	node.ReportState(uint64(10), block.BlockHash, nil)
	ast.Equal(true, node.stack.StateUpdating)

	state := &rbfttypes.ServiceState{
		MetaState: &rbfttypes.MetaState{
			Height: 20,
			Digest: block.BlockHash.String(),
		},
		Epoch: 0,
	}
	node.n.ReportExecuted(state)
	node.ReportState(uint64(20), block.BlockHash, nil)
	ast.Equal(false, node.stack.StateUpdating)

	node.ReportState(uint64(21), block.BlockHash, nil)
	ast.Equal(false, node.stack.StateUpdating)
}

func TestQuorum(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	node.stack.Nodes = make(map[uint64]*pb.VpInfo)
	node.stack.Nodes[1] = &pb.VpInfo{Id: uint64(1)}
	node.stack.Nodes[2] = &pb.VpInfo{Id: uint64(2)}
	node.stack.Nodes[3] = &pb.VpInfo{Id: uint64(3)}
	node.stack.Nodes[4] = &pb.VpInfo{Id: uint64(4)}
	// N = 3f + 1, f=1
	quorum := node.Quorum()
	ast.Equal(uint64(3), quorum)

	node.stack.Nodes[5] = &pb.VpInfo{Id: uint64(5)}
	// N = 3f + 2, f=1
	quorum = node.Quorum()
	ast.Equal(uint64(4), quorum)

	node.stack.Nodes[6] = &pb.VpInfo{Id: uint64(6)}
	// N = 3f + 3, f=1
	quorum = node.Quorum()
	ast.Equal(uint64(4), quorum)
}

func TestStatus2String(t *testing.T) {
	ast := assert.New(t)

	assertMapping := map[rbft.StatusType]string{
		rbft.Normal: "Normal",

		rbft.InConfChange:      "system is in conf change",
		rbft.InViewChange:      "system is in view change",
		rbft.InRecovery:        "system is in recovery",
		rbft.StateTransferring: "system is in state update",
		rbft.PoolFull:          "system is too busy",
		rbft.Pending:           "system is in pending state",
		rbft.Stopped:           "system is stopped",
		1000:                   "Unknown status: 1000",
	}

	for status, assertStatusStr := range assertMapping {
		statusStr := status2String(status)
		ast.Equal(assertStatusStr, statusStr)
	}
}
