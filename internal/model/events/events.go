package events

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type ExecutedEvent struct {
	Block          *pb.Block
	InterchainMeta *pb.InterchainMeta
	TxHashList     []*types.Hash
}

type CheckpointEvent struct {
	Index  uint64
	Digest types.Hash
}

type OrderMessageEvent struct {
	Data []byte
}
