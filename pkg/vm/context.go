package vm

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/sirupsen/logrus"
)

// Context represents the context of wasm
type Context struct {
	Caller           *types.Address
	Callee           *types.Address
	CurrentCaller    *types.Address
	Ledger           *ledger.Ledger
	TransactionIndex uint64
	TransactionData  *pb.TransactionData
	CurrentHeight    uint64
	Nonce            uint64
	Tx               pb.Transaction
	Logger           logrus.FieldLogger
}

// NewContext creates a context of wasm instance
func NewContext(tx pb.Transaction, txIndex uint64, data *pb.TransactionData, currentHeight uint64, ledger *ledger.Ledger, logger logrus.FieldLogger) *Context {
	return &Context{
		Caller:           tx.GetFrom(),
		Callee:           tx.GetTo(),
		CurrentCaller:    tx.GetFrom(),
		Ledger:           ledger,
		TransactionIndex: txIndex,
		TransactionData:  data,
		CurrentHeight:    currentHeight,
		Tx:               tx,
		Nonce:            tx.GetNonce(),
		Logger:           logger,
	}
}
