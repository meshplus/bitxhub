package peermgr

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom-ledger/internal/executor/system/base"
	"github.com/axiomesh/axiom-ledger/internal/executor/system/common"
	"github.com/axiomesh/axiom-ledger/internal/ledger"
	"github.com/axiomesh/axiom-ledger/internal/ledger/mock_ledger"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
	network "github.com/axiomesh/axiom-p2p"
)

func getAddr(p2p network.Network) (peer.AddrInfo, error) {
	realAddr := fmt.Sprintf("%s/p2p/%s", p2p.LocalAddr(), p2p.PeerID())
	multiaddr, err := ma.NewMultiaddr(realAddr)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiaddr)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	return *addrInfo, nil
}

func NewSwarms(t *testing.T, peerCnt int, versionChange bool) []*Swarm {
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
	stateLedger.EXPECT().NewView().Return(stateLedger).AnyTimes()
	stateLedger.EXPECT().GetState(gomock.Any(), gomock.Any()).DoAndReturn(func(addr *types.Address, key []byte) (bool, []byte) {
		return false, nil
	}).AnyTimes()

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	ld, err := leveldb.New(filepath.Join(repoRoot, "peermgr"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.EpochManagerContractAddr), ledger.NewChanger())
	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()

	epochInfo := repo.GenesisEpochInfo(false)
	epochInfo.CandidateSet = append(epochInfo.CandidateSet, &rbft.NodeInfo{
		ID:        5,
		P2PNodeID: "16Uiu2HAmJ3bjAhtYc7QabCWWUKagY9RLddypDPXhFYkmFxSwzHQd",
	})
	err = base.InitEpochInfo(mockLedger.StateLedger, epochInfo)
	assert.Nil(t, err)

	var addrs []peer.AddrInfo
	for i := 0; i < peerCnt; i++ {
		rep, err := repo.DefaultWithNodeIndex(t.TempDir(), i, true)
		require.Nil(t, err)
		if versionChange && i == peerCnt-1 {
			repo.BuildVersionSecret = "Shanghai"
		}

		rep.Config.Port.P2P = 0
		rep.Config.Genesis.EpochInfo.P2PBootstrapNodeAddresses = []string{}
		rep.EpochInfo.P2PBootstrapNodeAddresses = []string{}
		swarm, err := New(rep, log.NewWithModule(fmt.Sprintf("swarm%d", i)), mockLedger)
		require.Nil(t, err)
		err = swarm.Start()
		require.Nil(t, err)

		swarms = append(swarms, swarm)
		addr, err := getAddr(swarm.p2p)
		require.Nil(t, err)
		addrs = append(addrs, addr)
	}

	for i := 0; i < peerCnt; i++ {
		for j := 0; j < peerCnt; j++ {
			if i != j {
				err = swarms[i].p2p.Connect(addrs[j])
				require.Nil(t, err)
			}
		}
	}

	return swarms
}

func TestSwarm_GetBlockPack(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt, false)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}

	msg := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte("aaa"),
	}
	var err error
	_, err = swarms[0].Send(swarms[1].PeerID(), msg)
	require.NotNil(t, err)
	msg.Type = 100
	_, err = swarms[0].Send(swarms[1].PeerID(), msg)
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
	swarms := NewSwarms(t, peerCnt, false)
	defer stopSwarms(t, swarms)

	for swarms[0].CountConnectedPeers() != 3 {
		time.Sleep(100 * time.Millisecond)
	}
	gater := newConnectionGater(swarms[0].logger, swarms[0].ledger)
	require.False(t, gater.InterceptPeerDial("1"))
	for _, validator := range swarms[0].repo.EpochInfo.ValidatorSet {
		peerID, err := peer.Decode(validator.P2PNodeID)
		require.Nil(t, err)
		require.True(t, gater.InterceptPeerDial(peerID))
	}
	for _, candidate := range swarms[0].repo.EpochInfo.CandidateSet {
		peerID, err := peer.Decode(candidate.P2PNodeID)
		require.Nil(t, err)
		require.True(t, gater.InterceptPeerDial(peerID))
	}
	require.True(t, gater.InterceptAccept(nil))
}

func TestSwarm_Send(t *testing.T) {
	peerCnt := 4
	swarms := NewSwarms(t, peerCnt, false)
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
		res, err = swarms[0].Send(swarms[1].PeerID(), msg)
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
