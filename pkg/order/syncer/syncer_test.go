package syncer

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/stretchr/testify/require"
)

func preparePeerMgr(t *testing.T) peermgr.PeerManager {
	ctrl := gomock.NewController(t)
	mockPeerMgr := mock_peermgr.NewMockPeerManager(ctrl)
	genblocks := genBlocks(1024)
	mockPeerMgr.EXPECT().Send(gomock.Any(), gomock.Any()).DoAndReturn(func(id uint64, m *pb.Message) (*pb.Message, error) {
		switch m.Type {
		case pb.Message_GET_BLOCK_HEADERS:
			req := &pb.GetBlockHeadersRequest{}
			err := req.Unmarshal(m.Data)
			require.Nil(t, err)
			res := &pb.GetBlockHeadersResponse{}
			blockHeaders := make([]*pb.BlockHeader, 0)
			for i := req.Start; i <= req.End; i++ {
				blockHeaders = append(blockHeaders, genblocks[i-1].BlockHeader)
			}
			res.BlockHeaders = blockHeaders
			v, err := res.Marshal()
			require.Nil(t, err)
			m := &pb.Message{
				Type: pb.Message_GET_BLOCK_HEADERS_ACK,
				Data: v,
			}
			return m, nil
		case pb.Message_GET_BLOCKS:
			req := &pb.GetBlocksRequest{}
			err := req.Unmarshal(m.Data)
			require.Nil(t, err)
			res := &pb.GetBlocksResponse{}
			blocks := make([]*pb.Block, 0)
			for i := req.Start; i <= req.End; i++ {
				blocks = append(blocks, genblocks[i-1])
			}
			res.Blocks = blocks
			v, err := res.Marshal()
			require.Nil(t, err)
			m := &pb.Message{
				Type: pb.Message_GET_BLOCKS_ACK,
				Data: v,
			}
			return m, nil
		}
		return nil, fmt.Errorf("unhapply")
	}).AnyTimes()
	return mockPeerMgr
}

func TestStateSyncer_SyncCFTBlocks(t *testing.T) {
	mockPeerMgr := preparePeerMgr(t)
	peerIds := []uint64{2, 3, 4}
	logger := log.NewWithModule("syncer")
	syncer, err := New(10, mockPeerMgr, 2, peerIds, logger)
	require.Nil(t, err)

	begin := 2
	end := 100
	blockCh := make(chan *pb.Block, 1024)
	go syncer.SyncCFTBlocks(uint64(begin), uint64(end), blockCh)

	blocks := make([]*pb.Block, 0)
	for block := range blockCh {
		if block == nil {
			break
		}
		blocks = append(blocks, block)
	}

	require.Equal(t, len(blocks), end-begin+1)

}

func TestStateSyncer_SyncBFTBlocks(t *testing.T) {
	mockPeerMgr := preparePeerMgr(t)
	peerIds := []uint64{2, 3, 4}
	logger := log.NewWithModule("syncer")
	syncer, err := New(10, mockPeerMgr, 3, peerIds, logger)
	require.Nil(t, err)

	begin := 2
	end := 100
	blockCh := make(chan *pb.Block, 1024)

	metaHash := types.NewHashByStr("0xbC1C6897f97782F3161492d5CcfBE0691502f15894A0b2f2f40069C995E33cCB")
	go syncer.SyncBFTBlocks(uint64(begin), uint64(end), metaHash, blockCh)

	blocks := make([]*pb.Block, 0)
	for block := range blockCh {
		if block == nil {
			break
		}
		blocks = append(blocks, block)
	}

	require.Equal(t, len(blocks), end-begin+1)
}

func genBlocks(count int) []*pb.Block {
	blocks := make([]*pb.Block, 0, count)
	for height := 1; height <= count; height++ {
		block := &pb.Block{}
		if height == 1 {
			block.BlockHeader = &pb.BlockHeader{
				Number:      1,
				StateRoot:   types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"),
				TxRoot:      nil,
				ReceiptRoot: nil,
				ParentHash:  nil,
				Timestamp:   0,
				Version:     nil,
			}
			block.BlockHash = types.NewHashByStr("0xbC1C6897f97782F3161492d5CcfBE0691502f15894A0b2f2f40069C995E33cCB")
		} else {
			block.BlockHeader = &pb.BlockHeader{
				Number:      uint64(height),
				StateRoot:   blocks[len(blocks)-1].BlockHeader.StateRoot,
				TxRoot:      nil,
				ReceiptRoot: nil,
				ParentHash:  blocks[len(blocks)-1].BlockHash,
				Timestamp:   0,
				Version:     nil,
			}
			block.BlockHash = block.Hash()
		}
		blocks = append(blocks, block)
	}
	return blocks
}
