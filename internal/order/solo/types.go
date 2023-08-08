package solo

import "github.com/axiomesh/axiom-kit/types"

type chainState struct {
	Height     uint64
	BlockHash  *types.Hash
	TxHashList []*types.Hash
}

type getTxReq struct {
	Hash string
	Resp chan *types.Transaction
}
