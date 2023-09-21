package testutil

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom-ledger/internal/order/common"
	"github.com/axiomesh/axiom-ledger/internal/peermgr/mock_peermgr"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
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

	N := 3
	f := (N - 1) / 3
	mock.EXPECT().CountConnectedPeers().Return(uint64((N + f + 2) / 2)).AnyTimes()
	return mock
}

func MockOrderConfig(logger logrus.FieldLogger, ctrl *gomock.Controller, kvType string, t *testing.T) *common.Config {
	s, err := types.GenerateSigner()
	assert.Nil(t, err)

	peerMgr := MockMiniPeerManager(ctrl)
	peerMgr.EXPECT().Peers().Return([]peer.AddrInfo{}).AnyTimes()

	genesisEpochInfo := repo.GenesisEpochInfo(false)
	conf := &common.Config{
		Config:             repo.DefaultOrderConfig(),
		Logger:             logger,
		StoragePath:        t.TempDir(),
		StorageType:        kvType,
		OrderType:          "",
		PrivKey:            s.Sk,
		SelfAccountAddress: genesisEpochInfo.ValidatorSet[0].AccountAddress,
		GenesisEpochInfo:   genesisEpochInfo,
		PeerMgr:            peerMgr,
		Applied:            0,
		Digest:             "",
		GetEpochInfoFromEpochMgrContractFunc: func(epoch uint64) (*rbft.EpochInfo, error) {
			return genesisEpochInfo, nil
		},
		GetChainMetaFunc: GetChainMetaFunc,
		GetAccountNonce: func(address *types.Address) uint64 {
			return 0
		},
		GetCurrentEpochInfoFromEpochMgrContractFunc: func() (*rbft.EpochInfo, error) {
			return genesisEpochInfo, nil
		},
	}
	return conf
}

func GetChainMetaFunc() *types.ChainMeta {
	block := ConstructBlock("block1", uint64(1))
	return &types.ChainMeta{Height: uint64(1), BlockHash: block.BlockHash}
}
