package events

import (
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type NewTxsEvent struct {
	Txs []pb.Transaction
}

type ExecutedEvent struct {
	Block          *pb.Block
	InterchainMeta *pb.InterchainMeta
	TxHashList     []*types.Hash
}

type CheckpointEvent struct {
	Index  uint64
	Digest types.Hash
}

type NodeEvent struct {
	NodeId        uint64
	NodeEventType governance.EventType
}
