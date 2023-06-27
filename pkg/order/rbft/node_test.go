package rbft

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
	"github.com/ultramesh/rbft"
	"github.com/ultramesh/rbft/mempool"
	"github.com/ultramesh/rbft/rbftpb"
)

func TestPrepare(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	order := mockOrder(ctrl, t)
	tx1 := mempool.ConstructTx("account1")
	tx1.(*pb.BxhTransaction).Nonce = uint64(0)
	err := order.Prepare(tx1)
	ast.NotNil(err)
	ast.Equal("system is in pending state", err.Error())

	err = order.Start()
	ast.Nil(err)
	err = order.Prepare(tx1)
	ast.NotNil(err)
	ast.Equal("system is in recovery", err.Error())

	pendingNonce := order.GetPendingNonceByAccount(tx1.GetFrom().String())
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

	node.stack.readyC <- &ready{
		height: uint64(2),
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
	rbftConf, err := generateRbftConfig("./testdata/", mockOrderConfig(logger, ctrl))
	ast.Nil(err)
	rbftConf.Logger.Critical()
	rbftConf.Logger.Criticalf("test critical")
	rbftConf.Logger.Notice()
	rbftConf.Logger.Noticef("test critical")
	ast.Equal(25, rbftConf.SetSize)
	ast.Equal(uint64(500), rbftConf.PoolConfig.BatchSize)
	ast.Equal(uint64(50000), rbftConf.PoolConfig.PoolSize)
	ast.Equal(500*time.Millisecond, rbftConf.BatchTimeout)
	ast.Equal(3*time.Minute, rbftConf.CheckPoolTimeout)
	ast.Equal(5*time.Minute, rbftConf.PoolConfig.ToleranceTime)
}

func TestStep(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	err := node.Step([]byte("test"))
	ast.NotNil(err)
	msg := &rbftpb.ConsensusMessage{}
	msgBytes, _ := msg.Marshal()
	err = node.Step(msgBytes)
	ast.Nil(err)
}

func TestDelNode(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	err := node.DelNode(uint64(2))
	ast.Nil(err)
}

func TestReportState(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)

	block := constructBlock("blockHash", uint64(20))
	node.stack.stateUpdating = true
	node.stack.stateUpdateHeight = 20
	node.ReportState(uint64(10), block.BlockHash, nil)
	ast.Equal(true, node.stack.stateUpdating)

	state := &rbftpb.ServiceState{
		Applied: uint64(20),
		Digest:  block.BlockHash.String(),
	}
	node.n.ReportExecuted(state)
	node.ReportState(uint64(20), block.BlockHash, nil)
	ast.Equal(false, node.stack.stateUpdating)

	node.ReportState(uint64(21), block.BlockHash, nil)
	ast.Equal(false, node.stack.stateUpdating)
}

func TestQuorum(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	node.stack.nodes = make(map[uint64]*pb.VpInfo)
	node.stack.nodes[1] = &pb.VpInfo{Id: uint64(1)}
	node.stack.nodes[2] = &pb.VpInfo{Id: uint64(2)}
	node.stack.nodes[3] = &pb.VpInfo{Id: uint64(3)}
	node.stack.nodes[4] = &pb.VpInfo{Id: uint64(4)}
	// N = 3f + 1, f=1
	quorum := node.Quorum()
	ast.Equal(uint64(3), quorum)

	node.stack.nodes[5] = &pb.VpInfo{Id: uint64(5)}
	// N = 3f + 2, f=1
	quorum = node.Quorum()
	ast.Equal(uint64(4), quorum)

	node.stack.nodes[6] = &pb.VpInfo{Id: uint64(6)}
	// N = 3f + 3, f=1
	quorum = node.Quorum()
	ast.Equal(uint64(4), quorum)
}

func TestStatus2String(t *testing.T) {
	ast := assert.New(t)
	statusStr := status2String(rbft.Normal)
	ast.Equal("Normal", statusStr)
	statusStr = status2String(rbft.InViewChange)
	ast.Equal("system is in view change", statusStr)
	statusStr = status2String(rbft.InRecovery)
	ast.Equal("system is in recovery", statusStr)
	statusStr = status2String(rbft.InUpdatingN)
	ast.Equal("system is in updatingN", statusStr)
	statusStr = status2String(rbft.PoolFull)
	ast.Equal("system is too busy", statusStr)
	statusStr = status2String(rbft.StateTransferring)
	ast.Equal("system is in state update", statusStr)
	statusStr = status2String(rbft.Pending)
	ast.Equal("system is in pending state", statusStr)
	statusStr = status2String(rbft.InSyncState)
	ast.Equal("Unknown status", statusStr)
}
