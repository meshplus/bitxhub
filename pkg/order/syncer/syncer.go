package syncer

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type Syncer interface {
	// SyncCFTBlocks fetches the block list from other node, and just fetches but not verifies the block
	SyncCFTBlocks(begin, end uint64, blockCh chan *pb.Block) error

	// SyncBFTBlocks fetches the block list from quorum nodes, and verifies all the block
	SyncBFTBlocks(begin, end uint64, metaHash *types.Hash, blockCh chan *pb.Block) error
}
