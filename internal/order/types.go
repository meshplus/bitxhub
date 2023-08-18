package order

import "github.com/axiomesh/axiom-kit/types"

const (
	LocalTxEvent = iota
	RemoteTxEvent
)

// UncheckedTxEvent represents misc event sent by local modules
type UncheckedTxEvent struct {
	EventType int
	Event     interface{}
}

type TxWithResp struct {
	Tx     *types.Transaction
	RespCh chan *TxResp
}

type TxResp struct {
	Status   bool
	ErrorMsg string
}
