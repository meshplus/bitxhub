package events

import (
	"github.com/axiomesh/axiom-bft/common/consensus"
	"github.com/axiomesh/axiom-kit/types"
)

type ExecutedEvent struct {
	Block                  *types.Block
	TxHashList             []*types.Hash
	StateUpdatedCheckpoint *consensus.Checkpoint
}
