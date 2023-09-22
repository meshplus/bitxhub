package testutil

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom-ledger/internal/consensus/common"
	"github.com/axiomesh/axiom-ledger/internal/network/mock_network"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

var (
	mockBlockLedger      = make(map[uint64]*types.Block)
	mockLocalBlockLedger = make(map[uint64]*types.Block)
	mockChainMeta        *types.ChainMeta
)

func SetMockChainMeta(chainMeta *types.ChainMeta) {
	mockChainMeta = chainMeta
}

func ResetMockChainMeta() {
	block := ConstructBlock("block1", uint64(1))
	mockChainMeta = &types.ChainMeta{Height: uint64(1), BlockHash: block.BlockHash}
}

func SetMockBlockLedger(block *types.Block, local bool) {
	if local {
		mockLocalBlockLedger[block.Height()] = block
	} else {
		mockBlockLedger[block.Height()] = block
	}
}

func getMockBlockLedger(height uint64) (*types.Block, error) {
	if block, ok := mockBlockLedger[height]; ok {
		return block, nil
	}
	return nil, errors.New("block not found")
}

func ResetMockBlockLedger() {
	mockBlockLedger = make(map[uint64]*types.Block)
	mockLocalBlockLedger = make(map[uint64]*types.Block)
}

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

func MockMiniNetwork(ctrl *gomock.Controller) *mock_network.MockNetwork {
	mock := mock_network.NewMockNetwork(ctrl)
	mockPipe := mock_network.NewMockPipe(ctrl)
	mockPipe.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockPipe.EXPECT().Broadcast(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockPipe.EXPECT().Receive(gomock.Any()).Return(nil).AnyTimes()

	mock.EXPECT().CreatePipe(gomock.Any(), gomock.Any()).Return(mockPipe, nil).AnyTimes()

	mock.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(
		func(to string, msg *pb.Message) (*pb.Message, error) {
			if msg.Type == pb.Message_GET_BLOCK {
				num, err := strconv.Atoi(string(msg.Data))
				if err != nil {
					return nil, fmt.Errorf("convert %s string to int failed: %w", string(msg.Data), err)
				}
				block, err := getMockBlockLedger(uint64(num))
				if err != nil {
					return nil, fmt.Errorf("get block with height %d failed: %w", num, err)
				}
				v, err := block.Marshal()
				if err != nil {
					return nil, fmt.Errorf("marshal block with height %d failed: %w", num, err)
				}
				res := &pb.Message{Type: pb.Message_GET_BLOCK_ACK, Data: v}
				return res, nil
			}
			return nil, nil
		}).AnyTimes()

	N := 3
	f := (N - 1) / 3
	mock.EXPECT().CountConnectedPeers().Return(uint64((N + f + 2) / 2)).AnyTimes()
	return mock
}

func MockConsensusConfig(logger logrus.FieldLogger, ctrl *gomock.Controller, t *testing.T) *common.Config {
	s, err := types.GenerateSigner()
	assert.Nil(t, err)

	mockNetwork := MockMiniNetwork(ctrl)
	mockNetwork.EXPECT().Peers().Return([]peer.AddrInfo{}).AnyTimes()

	genesisEpochInfo := repo.GenesisEpochInfo(false)
	conf := &common.Config{
		Config:             repo.DefaultConsensusConfig(),
		Logger:             logger,
		ConsensusType:      "",
		PrivKey:            s.Sk,
		SelfAccountAddress: genesisEpochInfo.ValidatorSet[0].AccountAddress,
		GenesisEpochInfo:   genesisEpochInfo,
		Network:            mockNetwork,
		Applied:            0,
		Digest:             "",
		GetEpochInfoFromEpochMgrContractFunc: func(epoch uint64) (*rbft.EpochInfo, error) {
			return genesisEpochInfo, nil
		},
		GetChainMetaFunc: GetChainMetaFunc,
		GetBlockFunc: func(height uint64) (*types.Block, error) {
			if block, ok := mockLocalBlockLedger[height]; ok {
				return block, nil
			} else {
				return nil, errors.New("block not found")
			}
		},
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
	if mockChainMeta == nil {
		ResetMockChainMeta()
	}
	return mockChainMeta
}
