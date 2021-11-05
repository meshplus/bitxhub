package syncer

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

var _ Syncer = (*StateSyncer)(nil)

const defaultBlockFetch = 5

type StateSyncer struct {
	blockFetch uint64                        // amount of blocks to be fetched per retrieval request
	peerMgr    orderPeerMgr.OrderPeerManager // network manager
	badPeers   *sync.Map                     // peer node set who return bad block
	quorum     uint64                        // quorum node numbers
	peerIds    []uint64                      // peers who have current newly consensus state
	logger     logrus.FieldLogger
}

type rangeHeight struct {
	begin uint64
	end   uint64
}

func New(blockFetch uint64, peerMgr orderPeerMgr.OrderPeerManager, quorum uint64, peerIds []uint64, logger logrus.FieldLogger) (*StateSyncer, error) {
	if blockFetch == 0 {
		blockFetch = defaultBlockFetch
	}
	if quorum <= 0 {
		return nil, fmt.Errorf("the vp nodes' quorum must be positive")
	}
	return &StateSyncer{
		blockFetch: blockFetch,
		peerMgr:    peerMgr,
		logger:     logger,
		quorum:     quorum,
		peerIds:    peerIds,
		badPeers:   &sync.Map{},
	}, nil
}

func (s *StateSyncer) SyncCFTBlocks(begin, end uint64, blockCh chan *pb.Block) error {
	rangeHeights, err := s.calcRangeHeight(begin, end)
	if err != nil {
		return fmt.Errorf("calculate range height failed: %w", err)
	}

	for _, rangeHeight := range rangeHeights {
		rangeTmp := rangeHeight
		err := retry.Retry(func(attempt uint) error {
			id, err := s.randPeers()
			if err != nil {
				s.logger.Errorf(err.Error())
				return err
			}

			s.logger.WithFields(logrus.Fields{
				"begin":   rangeTmp.begin,
				"end":     rangeTmp.end,
				"peer_id": id,
			}).Info("syncing range block")

			blocks, err := s.fetchBlocks(id, rangeTmp.begin, rangeTmp.end)
			if err != nil {
				s.badPeers.Store(id, nil)
				s.logger.Errorf("fetch blocks failed: %w", err)
				return fmt.Errorf("fetch blocks failed: %w", err)
			}
			for _, block := range blocks {
				blockCh <- block
			}
			return nil
		}, strategy.Wait(100*time.Millisecond))
		if err != nil {
			s.logger.Error(err)
		}
	}
	blockCh <- nil

	return nil
}

func (s *StateSyncer) SyncBFTBlocks(begin, end uint64, metaHash *types.Hash, blockCh chan *pb.Block) error {
	rangeHeights, err := s.calcRangeHeight(begin, end)
	if err != nil {
		return fmt.Errorf("calculate range height failed: %w", err)
	}

	var parentBlockHash *types.Hash
	for i, rangeHeight := range rangeHeights {
		if i == 0 {
			parentBlockHash = metaHash
		}
		rangeTmp := rangeHeight
		headers := s.syncQuorumRangeBlockHeaders(rangeTmp, parentBlockHash)
		if headers == nil {
			return fmt.Errorf("fetch and verify the quorum peers' block header error: %v", rangeTmp)
		}
		blocks := s.syncRangeBlocks(headers)
		if blocks == nil {
			return fmt.Errorf("fetch and verify peers' block error: %v", rangeTmp)
		}
		for _, block := range blocks {
			blockCh <- block
		}
		parentBlockHash = blocks[len(blocks)-1].Hash()
	}
	blockCh <- nil
	return nil
}

func (s *StateSyncer) syncQuorumRangeBlockHeaders(rangeHeight *rangeHeight, parentBlockHash *types.Hash) []*pb.BlockHeader {
	var isQuorum bool
	var hash string
	latestBlockHeaderCounter := make(map[string]uint64)
	blockHeadersM := make(map[string][]*pb.BlockHeader)

	fetchAndVerifyBlockHeaders := func(id uint64) {
		s.logger.WithFields(logrus.Fields{
			"begin":   rangeHeight.begin,
			"end":     rangeHeight.end,
			"peer_id": id,
		}).Info("syncing range block header")
		headers, err := s.fetchBlockHeaders(id, rangeHeight.begin, rangeHeight.end)
		if err != nil {
			s.logger.Errorf("fetch block headers error:%w", err)
			return
		}
		err = s.verifyBlockHeaders(parentBlockHash, headers)
		if err != nil {
			s.badPeers.Store(id, nil)
			s.logger.Errorf("check block headers error:%w", err)
			return
		}
		latestBlock := &pb.Block{BlockHeader: headers[len(headers)-1]}
		blockHash := latestBlock.Hash()
		latestBlockHeaderCounter[blockHash.String()]++
		blockHeadersM[blockHash.String()] = headers
	}

	for _, id := range s.peerIds {

		fetchAndVerifyBlockHeaders(id)
		for latestHash, counter := range latestBlockHeaderCounter {
			if counter >= s.quorum {
				hash = latestHash
				isQuorum = true
				break
			}
		}
		if isQuorum {
			break
		}
	}
	if !isQuorum {
		return nil
	}

	return blockHeadersM[hash]

}

func (s *StateSyncer) syncRangeBlocks(headers []*pb.BlockHeader) []*pb.Block {
	var blocks []*pb.Block
	begin := headers[0].Number
	end := headers[len(headers)-1].Number

	fetchAndVerifyBlocks := func(id uint64) {
		s.logger.WithFields(logrus.Fields{
			"begin":   begin,
			"end":     end,
			"peer_id": id,
		}).Info("syncing range block")
		fetchBlocks, err := s.fetchBlocks(id, begin, end)
		if err != nil {
			s.badPeers.Store(id, nil)
			s.logger.Errorf("fetch block headers error:%w", err)
			return
		}
		for i, block := range fetchBlocks {
			err := s.verifyBlock(headers[i], block)
			if err != nil {
				s.badPeers.Store(id, nil)
				s.logger.Errorf("check block headers error:%w", err)
				return
			}
		}
		blocks = fetchBlocks
	}
	for _, id := range s.peerIds {
		if blocks != nil {
			break
		}
		fetchAndVerifyBlocks(id)
	}
	return blocks
}

func (s *StateSyncer) randPeers() (uint64, error) {
	ids := make([]uint64, 0)
	for _, id := range s.peerIds {
		_, ok := s.badPeers.Load(id)
		if ok {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("peers nums is 0")
	}
	randIndex := rand.Int63n(int64(len(ids)))
	return ids[randIndex], nil
}

func (s *StateSyncer) calcRangeHeight(begin, end uint64) ([]*rangeHeight, error) {
	if begin > end {
		return nil, fmt.Errorf("the end height:%d is less than the start height:%d", end, begin)
	}
	startNo := begin / s.blockFetch
	rangeHeights := make([]*rangeHeight, 0)
	for begin <= end {
		rangeBegin := begin
		rangeEnd := (startNo + 1) * s.blockFetch
		if rangeEnd > end {
			rangeEnd = end
		}

		rangeHeights = append(rangeHeights, &rangeHeight{
			begin: rangeBegin,
			end:   rangeEnd,
		})
		begin = rangeEnd + 1
		startNo++
	}
	return rangeHeights, nil
}

func (s *StateSyncer) fetchBlockHeaders(id uint64, begin, end uint64) ([]*pb.BlockHeader, error) {
	if begin > end {
		return nil, fmt.Errorf("the end height:%d is less than the start height:%d", end, begin)
	}

	req := &pb.GetBlockHeadersRequest{
		Start: begin,
		End:   end,
	}
	data, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal get block header request error: %w", err)
	}
	m := &pb.Message{
		Type: pb.Message_GET_BLOCK_HEADERS,
		Data: data,
	}

	res, err := s.peerMgr.Send(id, m)
	if err != nil {
		return nil, fmt.Errorf("send message to %d failed: %w", id, err)
	}

	blockHeaders := &pb.GetBlockHeadersResponse{}
	if err := blockHeaders.Unmarshal(res.Data); err != nil {
		return nil, fmt.Errorf("unmarshal get block header response error: %w", err)
	}
	return blockHeaders.BlockHeaders, nil
}

func (s *StateSyncer) fetchBlocks(id uint64, begin, end uint64) ([]*pb.Block, error) {
	if begin > end {
		return nil, fmt.Errorf("the end height:%d is less than the start height: %d", end, begin)
	}

	req := &pb.GetBlocksRequest{
		Start: begin,
		End:   end,
	}
	data, err := req.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal get block header request error: %w", err)
	}
	m := &pb.Message{
		Type: pb.Message_GET_BLOCKS,
		Data: data,
	}

	res, err := s.peerMgr.Send(id, m)
	if err != nil {
		return nil, fmt.Errorf("send message to %d failed: %w", id, err)
	}

	blocks := &pb.GetBlocksResponse{}
	if err := blocks.Unmarshal(res.Data); err != nil {
		return nil, fmt.Errorf("unmarshal get block header response error: %w", err)
	}
	return blocks.Blocks, nil
}

func (s *StateSyncer) verifyBlockHeaders(parentHash *types.Hash, headers []*pb.BlockHeader) error {
	if parentHash == nil || len(headers) == 0 {
		return fmt.Errorf("args must not be nil or empty")
	}
	for _, header := range headers {
		block := &pb.Block{BlockHeader: header}
		hash := block.Hash()
		ok, _ := parentHash.Equals(header.ParentHash)
		if !ok {
			return fmt.Errorf("block number is %d, hash is %s, but parent hash is %s", header.Number, hash.String(), header.ParentHash)
		}
		parentHash = hash
	}
	return nil
}

func (s *StateSyncer) verifyBlock(header *pb.BlockHeader, block *pb.Block) error {
	if header == nil || block == nil {
		return fmt.Errorf("args must not be nil or empty")
	}
	originBlock := &pb.Block{BlockHeader: header}
	hash := originBlock.Hash()
	//todo(jz): need to calc txs merkle root and compare with block's tx root
	ok, _ := hash.Equals(block.BlockHash)
	if !ok {
		return fmt.Errorf("block hash is not equals, number is %d", block.Height())
	}
	return nil
}
