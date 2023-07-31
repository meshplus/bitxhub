package events

import (
	"github.com/axiomesh/axiom-kit/types"
)

type NewTxsEvent struct {
	Txs []*types.Transaction
}

type ExecutedEvent struct {
	Block      *types.Block
	TxHashList []*types.Hash
}
