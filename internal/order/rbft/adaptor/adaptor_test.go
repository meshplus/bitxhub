package adaptor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/axiomesh/axiom-bft/common/consensus"
	rbfttypes "github.com/axiomesh/axiom-bft/types"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/order/common"
	"github.com/axiomesh/axiom-ledger/internal/order/rbft/testutil"
	network "github.com/axiomesh/axiom-p2p"
)

func mockAdaptor(ctrl *gomock.Controller, kvType string, t *testing.T) *RBFTAdaptor {
	logger := log.NewWithModule("order")
	blockC := make(chan *common.CommitEvent, 1024)
	_, cancel := context.WithCancel(context.Background())
	stack, err := NewRBFTAdaptor(testutil.MockOrderConfig(logger, ctrl, kvType, t), blockC, cancel)
	assert.Nil(t, err)

	consensusMsgPipes := make(map[int32]network.Pipe, len(consensus.Type_name))
	for id, name := range consensus.Type_name {
		msgPipe, err := stack.config.PeerMgr.CreatePipe(context.Background(), "test_pipe_"+name)
		assert.Nil(t, err)
		consensusMsgPipes[id] = msgPipe
	}
	globalMsgPipe, err := stack.config.PeerMgr.CreatePipe(context.Background(), "test_pipe_global")
	assert.Nil(t, err)
	stack.SetMsgPipes(consensusMsgPipes, globalMsgPipe)
	err = stack.UpdateEpoch()
	assert.Nil(t, err)
	return stack
}

func TestSignAndVerify(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}
	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			adaptor := mockAdaptor(ctrl, tc.kvType, t)
			msgSign, err := adaptor.Sign([]byte("test sign"))
			ast.Nil(err)

			// TODO: impl it
			// err = adaptor.Verify(adaptor.Nodes[0].Pid, msgSign, []byte("wrong sign"))
			// ast.NotNil(err)

			err = adaptor.Verify(adaptor.config.GenesisEpochInfo.ValidatorSet[0].P2PNodeID, msgSign, []byte("test sign"))
			ast.Nil(err)
		})
	}
}

func TestExecute(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			adaptor := mockAdaptor(ctrl, tc.kvType, t)
			txs := make([]*types.Transaction, 0)
			tx := &types.Transaction{
				Inner: &types.DynamicFeeTx{
					Nonce: 0,
				},
				Time: time.Time{},
			}

			txs = append(txs, tx, tx)
			adaptor.Execute(txs, []bool{true}, uint64(2), time.Now().UnixNano(), "")
			ready := <-adaptor.ReadyC
			ast.Equal(uint64(2), ready.Height)
		})
	}
}

func TestStateUpdate(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}
	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			adaptor := mockAdaptor(ctrl, tc.kvType, t)
			block := testutil.ConstructBlock("block3", uint64(3))
			adaptor.StateUpdate(block.BlockHeader.Number, block.BlockHash.String(), nil)
			ast.Equal(true, adaptor.StateUpdating)

			block = testutil.ConstructBlock("block2", uint64(2))
			adaptor.StateUpdate(block.BlockHeader.Number, block.BlockHash.String(), nil)
			targetB := <-adaptor.BlockC
			ast.Equal(uint64(2), targetB.Block.BlockHeader.Number)
			ast.Equal(true, adaptor.StateUpdating)
		})
	}
}

// refactor this unit test
func TestNetwork(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			adaptor := mockAdaptor(ctrl, tc.kvType, t)
			adaptor.config.Config.Rbft.EnableMultiPipes = false
			msg := &consensus.ConsensusMessage{}
			err := adaptor.Unicast(context.Background(), msg, "1")
			ast.Nil(err)
			err = adaptor.Broadcast(context.Background(), msg)
			ast.Nil(err)

			adaptor.config.Config.Rbft.EnableMultiPipes = true
			msg = &consensus.ConsensusMessage{}
			err = adaptor.Unicast(context.Background(), msg, "1")
			ast.Nil(err)

			err = adaptor.Unicast(context.Background(), &consensus.ConsensusMessage{Type: consensus.Type(-1)}, "1")
			ast.Error(err)

			err = adaptor.Broadcast(context.Background(), msg)
			ast.Nil(err)

			err = adaptor.Broadcast(context.Background(), &consensus.ConsensusMessage{Type: consensus.Type(-1)})
			ast.Error(err)

			adaptor.SendFilterEvent(rbfttypes.InformTypeFilterFinishRecovery)
		})
	}
}

func TestEpochService(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			adaptor := mockAdaptor(ctrl, tc.kvType, t)
			e, err := adaptor.GetEpochInfo(1)
			ast.Nil(err)
			ast.Equal(uint64(1), e.Epoch)

			e, err = adaptor.GetCurrentEpochInfo()
			ast.Nil(err)
			ast.Equal(uint64(1), e.Epoch)
		})
	}
}
