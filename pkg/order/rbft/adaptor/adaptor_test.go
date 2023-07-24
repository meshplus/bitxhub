package adaptor

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hyperchain/go-hpc-rbft/common/consensus"
	rbfttypes "github.com/hyperchain/go-hpc-rbft/types"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order/rbft/testutil"
	ethtypes "github.com/meshplus/eth-kit/types"
	"github.com/stretchr/testify/assert"
)

func mockAdaptor(ctrl *gomock.Controller, t *testing.T) *RBFTAdaptor {
	logger := log.NewWithModule("order")
	blockC := make(chan *pb.CommitEvent, 1024)
	_, cancel := context.WithCancel(context.Background())
	stack, err := NewRBFTAdaptor(testutil.MockOrderConfig(logger, ctrl, t), blockC, cancel, false)
	assert.Nil(t, err)
	return stack
}

func TestSignAndVerify(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	adaptor := mockAdaptor(ctrl, t)
	msgSign, err := adaptor.Sign([]byte("test sign"))
	ast.Nil(err)

	// TODO: impl it
	// err = adaptor.Verify(adaptor.Nodes[0].Pid, msgSign, []byte("wrong sign"))
	// ast.NotNil(err)

	err = adaptor.Verify(adaptor.Nodes[0].Pid, msgSign, []byte("test sign"))
	ast.Nil(err)
}

func TestExecute(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	adaptor := mockAdaptor(ctrl, t)

	txs := make([][]byte, 0)
	tx := ethtypes.EthTransaction{
		Inner: &ethtypes.DynamicFeeTx{
			Nonce: 0,
		},
		Time: time.Time{},
	}
	raw, err := tx.RbftMarshal()
	ast.Nil(err)
	txs = append(txs, raw, raw)
	adaptor.Execute(txs, []bool{true}, uint64(2), time.Now().Unix())
	ready := <-adaptor.ReadyC
	ast.Equal(uint64(2), ready.Height)
}

func TestStateUpdate(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	adaptor := mockAdaptor(ctrl, t)
	block := testutil.ConstructBlock("block3", uint64(3))
	adaptor.StateUpdate(block.BlockHeader.Number, block.BlockHash.String(), nil)
	ast.Equal(true, adaptor.StateUpdating)

	block = testutil.ConstructBlock("block2", uint64(2))
	adaptor.StateUpdate(block.BlockHeader.Number, block.BlockHash.String(), nil)
	targetB := <-adaptor.BlockC
	ast.Equal(uint64(2), targetB.Block.BlockHeader.Number)
	ast.Equal(true, adaptor.StateUpdating)
}

// refactor this unit test
func TestNetwork(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	adaptor := mockAdaptor(ctrl, t)

	msg := &consensus.ConsensusMessage{}
	to := uint64(1)
	err := adaptor.Unicast(context.Background(), msg, to)
	ast.Nil(err)
	err = adaptor.UnicastByHostname(context.Background(), msg, "1")
	ast.Nil(err)

	err = adaptor.Broadcast(context.Background(), msg)
	ast.Nil(err)

	adaptor.SendFilterEvent(rbfttypes.InformTypeFilterFinishRecovery)
}

func TestEpochService(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	adaptor := mockAdaptor(ctrl, t)

	r := adaptor.Reconfiguration()
	ast.Equal(uint64(0), r)

	nodes := adaptor.GetNodeInfos()
	ast.Equal(4, len(nodes))

	v := adaptor.GetAlgorithmVersion()
	ast.Equal("RBFT", v)

	e := adaptor.GetEpoch()
	ast.Equal(uint64(1), e)

	c := adaptor.IsConfigBlock(1)
	ast.Equal(false, c)

	checkpoint := adaptor.GetLastCheckpoint()
	ast.Equal(uint64(0), checkpoint.Checkpoint.Epoch)
	ast.Equal(uint64(0), checkpoint.Checkpoint.ExecuteState.Height)

	_, err := adaptor.GetCheckpointOfEpoch(0)
	ast.Error(err)

	err = adaptor.VerifyEpochChangeProof(nil, nil)
	ast.Nil(err)
}
