package testutil

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/peermgr/mock_peermgr"
	"github.com/axiomesh/axiom/pkg/repo"
)

func ConstructBlock(blockHashStr string, height uint64) *types.Block {
	from := make([]byte, 0)
	strLen := len(blockHashStr)
	for i := 0; i < 32; i++ {
		from = append(from, blockHashStr[i%strLen])
	}
	fromStr := hex.EncodeToString(from)
	blockHash := types.NewHashByStr(fromStr)
	header := &types.BlockHeader{
		Number:     height,
		ParentHash: blockHash,
		Timestamp:  time.Now().Unix(),
	}
	return &types.Block{
		BlockHash:    blockHash,
		BlockHeader:  header,
		Transactions: []*types.Transaction{},
	}
}

func MockMiniPeerManager(ctrl *gomock.Controller) *mock_peermgr.MockPeerManager {
	mock := mock_peermgr.NewMockPeerManager(ctrl)

	mockPipe := mock_peermgr.NewMockPipe(ctrl)
	mockPipe.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockPipe.EXPECT().Broadcast(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockPipe.EXPECT().Receive(gomock.Any()).Return(nil).AnyTimes()

	mock.EXPECT().CreatePipe(gomock.Any(), gomock.Any()).Return(mockPipe, nil).AnyTimes()

	block := ConstructBlock("block2", uint64(2))
	blockBytes, _ := block.Marshal()
	res := &pb.Message{Data: blockBytes}
	mock.EXPECT().Send(gomock.Any(), gomock.Any()).Return(res, nil).AnyTimes()

	nodes := make(map[uint64]*types.VpInfo)
	nodes[1] = &types.VpInfo{Id: uint64(1)}
	nodes[2] = &types.VpInfo{Id: uint64(2)}
	nodes[3] = &types.VpInfo{Id: uint64(3)}
	N := len(nodes)
	f := (N - 1) / 3
	mock.EXPECT().OrderPeers().Return(nodes).AnyTimes()
	mock.EXPECT().CountConnectedPeers().Return(uint64((N + f + 2) / 2)).AnyTimes()
	return mock
}

func MockOrderConfig(logger logrus.FieldLogger, ctrl *gomock.Controller, kvType string, t *testing.T) *order.Config {
	nodes := make(map[uint64]*types.VpInfo)
	s, err := types.GenerateSigner()
	assert.Nil(t, err)
	nodes[0] = &types.VpInfo{Id: uint64(1), Account: s.Addr.String(), Pid: "1"}
	nodes[1] = &types.VpInfo{Id: uint64(2), Pid: "2"}
	nodes[2] = &types.VpInfo{Id: uint64(3), Pid: "3"}
	nodes[3] = &types.VpInfo{Id: uint64(4), Pid: "4"}

	peerMgr := MockMiniPeerManager(ctrl)
	peerMgr.EXPECT().Peers().Return([]peer.AddrInfo{}).AnyTimes()

	conf := &order.Config{
		StoragePath:      t.TempDir(),
		Config:           repo.DefaultOrderConfig(),
		StorageType:      kvType,
		ID:               uint64(1),
		Nodes:            nodes,
		IsNew:            false,
		Logger:           logger,
		PeerMgr:          peerMgr,
		PrivKey:          s.Sk,
		GetChainMetaFunc: GetChainMetaFunc,
		GetAccountNonce: func(address *types.Address) uint64 {
			return 0
		},
	}
	return conf
}

func GetChainMetaFunc() *types.ChainMeta {
	block := ConstructBlock("block1", uint64(1))
	return &types.ChainMeta{Height: uint64(1), BlockHash: block.BlockHash}
}
