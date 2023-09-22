package common

import (
	"github.com/axiomesh/axiom-bft/common/consensus"
	"github.com/axiomesh/axiom-kit/types"
)

const (
	LocalTxEvent = iota
	RemoteTxEvent
)

// UncheckedTxEvent represents misc event sent by local modules
type UncheckedTxEvent struct {
	EventType int
	Event     any
}

type TxWithResp struct {
	Tx     *types.Transaction
	RespCh chan *TxResp
}

type TxResp struct {
	Status   bool
	ErrorMsg string
}

type CommitEvent struct {
	Block                  *types.Block
	StateUpdatedCheckpoint *consensus.Checkpoint
}
