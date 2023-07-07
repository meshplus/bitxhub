package rbft

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
	"github.com/ultramesh/rbft/rbftpb"
)

func TestAddNode(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)

	change := &rbftpb.ConfChange{
		NodeID: uint64(5),
		Type:   rbftpb.ConfChangeType_ConfChangeAddNode,
	}
	change.Context = []byte("test")
	node.stack.UpdateTable(change)
	ast.Equal(4, len(node.stack.nodes), "Unmarshal router failed")

	change = &rbftpb.ConfChange{
		NodeID: uint64(5),
		Type:   rbftpb.ConfChangeType_ConfChangeAddNode,
	}
	peerInfo := &rbftpb.Peer{
		Context: []byte("test"),
	}
	change.Context, _ = peerInfo.Marshal()
	node.stack.UpdateTable(change)
	ast.Equal(4, len(node.stack.nodes), "Unmarshal router failed")

	change = &rbftpb.ConfChange{
		NodeID: uint64(5),
		Type:   rbftpb.ConfChangeType_ConfChangeAddNode,
	}
	vpInfo := &pb.VpInfo{}
	vpInfoBytes, _ := vpInfo.Marshal()
	peerInfo = &rbftpb.Peer{
		Context: vpInfoBytes,
	}
	change.Context, _ = peerInfo.Marshal()
	node.stack.UpdateTable(change)
	ast.Equal(5, len(node.stack.nodes))
}

func TestRemoveNode(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	change := &rbftpb.ConfChange{
		NodeID: uint64(2),
		Type:   rbftpb.ConfChangeType_ConfChangeRemoveNode,
	}
	node.stack.UpdateTable(change)
	ast.Equal(3, len(node.stack.nodes))
}

func TestUpdateNode(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)

	change := &rbftpb.ConfChange{
		Type: rbftpb.ConfChangeType_ConfChangeUpdateNode,
	}
	route := &rbftpb.Router{}
	peers := make([]*rbftpb.Peer, 0)
	vpInfo1 := &pb.VpInfo{Id: uint64(1)}
	vpInfo2 := &pb.VpInfo{Id: uint64(2)}
	vpInfoBytes1, _ := vpInfo1.Marshal()
	vpInfoBytes2, _ := vpInfo2.Marshal()
	peer1 := &rbftpb.Peer{
		Context: vpInfoBytes1,
	}
	peer2 := &rbftpb.Peer{
		Context: vpInfoBytes2,
	}
	peers = append(peers, peer1, peer2)
	route.Peers = peers
	routeBytes, _ := route.Marshal()

	change = &rbftpb.ConfChange{
		Type: rbftpb.ConfChangeType_ConfChangeUpdateNode,
	}
	change.Context = []byte("test")
	node.stack.UpdateTable(change)
	ast.Equal(4, len(node.stack.nodes), "Unmarshal router failed")

	change.Context = routeBytes
	node.stack.UpdateTable(change)
	ast.Equal(2, len(node.stack.nodes))
}

func TestSignAndVerify(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	msgSign, err := node.stack.Sign([]byte("test sign"))
	ast.Nil(err)

	err = node.stack.Verify(uint64(0), msgSign, []byte("wrong sign"))
	ast.NotNil(err)

	err = node.stack.Verify(uint64(0), msgSign, []byte("test sign"))
	ast.Nil(err)
}

func TestExecute(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	err := node.Start()
	ast.Nil(err)
	txs := make([]pb.Transaction, 0)
	txs = append(txs, &pb.BxhTransaction{Nonce: uint64(1)}, &pb.BxhTransaction{Nonce: uint64(2)})
	node.stack.Execute(txs, []bool{true}, uint64(2), time.Now().Unix())
	block := <-node.blockC
	ast.Equal(uint64(2), block.Block.Height())
}

// refactor this unit test
func TestUnicast(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	err := node.Start()
	ast.Nil(err)

	msg := &rbftpb.ConsensusMessage{}
	to := uint64(1)
	err = node.stack.Unicast(msg, to)
	ast.Nil(err)

	node.stack.SendFilterEvent(rbftpb.InformType_FilterFinishRecovery)
}

func TestStateUpdate(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)
	block := constructBlock("block3", uint64(3))
	node.stack.StateUpdate(block.BlockHeader.Number, block.BlockHash.String(), []uint64{1, 2, 3})
	ast.Equal(true, node.stack.stateUpdating)

	block = constructBlock("block2", uint64(2))
	node.stack.StateUpdate(block.BlockHeader.Number, block.BlockHash.String(), []uint64{1, 2, 3})
	targetB := <-node.stack.blockC
	ast.Equal(uint64(2), targetB.Block.BlockHeader.Number)
	ast.Equal(true, node.stack.stateUpdating)
}
