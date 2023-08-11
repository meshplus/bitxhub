package solo

import "github.com/axiomesh/axiom-kit/types"

const (
	maxChanSize = 1024
	ErrPoolFull = "mempool is full"
)

// consensusEvent is a type meant to clearly convey that the return type or parameter to a function will be supplied to/from an events.Manager
type consensusEvent any

// chainState is a type for reportState
type chainState struct {
	Height     uint64
	BlockHash  *types.Hash
	TxHashList []*types.Hash
}

// getTxReq is a type for api request getTx
type getTxReq struct {
	Hash string
	Resp chan *types.Transaction
}
