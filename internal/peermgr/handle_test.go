package peermgr

import (
	"fmt"
	"testing"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
)

func NewSwarms(t *testing.T, peerCnt int) []*Swarm {
	var swarms []*Swarm
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	chainLedger.EXPECT().GetBlock(gomock.Any()).Return(&types.Block{
		BlockHeader: &types.BlockHeader{
			Number: 1,
		},
	}, nil).AnyTimes()

	chainLedger.EXPECT().GetBlockSign(gomock.Any()).Return([]byte("sign"), nil).AnyTimes()
	chainLedger.EXPECT().GetTransaction(gomock.Any()).Return(&types.Transaction{}, nil).AnyTimes()
	stateLedger.EXPECT().Copy().Return(stateLedger).AnyTimes()
	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(func(addr *types.Address, key []byte) (bool, []byte) {
		return false, nil
	}).AnyTimes()

	for i := 0; i < peerCnt; i++ {
		repo, err := repo.DefaultWithNodeIndex(t.TempDir(), i)
		require.Nil(t, err)

		swarm, err := New(repo, log.NewWithModule(fmt.Sprintf("swarm%d", i)), mockLedger)
		require.Nil(t, err)
		err = swarm.Start()
		require.Nil(t, err)

		swarms = append(swarms, swarm)
	}
	return swarms
}

func TestSwarm_GetBlockPack(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte("aaa"),
	}
	var err error
	_, err = swarms[0].Send(uint64(2), msg)
	require.NotNil(t, err)
	msg.Type = 100
	_, err = swarms[0].Send(uint64(2), msg)
	require.NotNil(t, err)
	for i := 0; i < len(swarms); i++ {
		err = swarms[i].Stop()
		require.Nil(t, err)
	}
}

func stopSwarms(t *testing.T, swarms []*Swarm) error {
	for _, swarm := range swarms {
		err := swarm.Stop()
		assert.Nil(t, err)
	}
	return nil
}

func TestSwarm_Gater(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}
	gater := newConnectionGater(swarms[0].logger, swarms[0].ledger)
	require.True(t, gater.InterceptPeerDial(peer.ID("1")))
	require.True(t, gater.InterceptAddrDial("1", swarms[1].multiAddrs[1].Addrs[0]))
	require.True(t, gater.InterceptAccept(nil))
}

func TestSwarm_Send(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte("1"),
	}
	var res *pb.Message
	var err error
	err = retry.Retry(func(attempt uint) error {
		res, err = swarms[0].Send(uint64(2), msg)
		if err != nil {
			swarms[0].logger.Errorf(err.Error())
			return err
		}
		return nil
	}, strategy.Wait(50*time.Millisecond))
	require.Nil(t, err)
	require.Equal(t, pb.Message_GET_BLOCK_ACK, res.Type)
	var block types.Block
	err = block.Unmarshal(res.Data)
	require.Nil(t, err)
	require.Equal(t, uint64(1), block.BlockHeader.Number)
}
