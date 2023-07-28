package events

import (
	"github.com/meshplus/bitxhub-kit/types"
)

type NewTxsEvent struct {
	Txs []*types.Transaction
}

type ExecutedEvent struct {
	Block      *types.Block
	TxHashList []*types.Hash
}
