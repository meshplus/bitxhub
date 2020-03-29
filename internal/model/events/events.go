package events

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type NewBlockEvent struct {
	Block *pb.Block
}

type CheckpointEvent struct {
	Index  uint64
	Digest types.Hash
}

type OrderMessageEvent struct {
	Data []byte
}
