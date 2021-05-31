package wasm

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/sirupsen/logrus"
)

// Context represents the context of wasm
type Context struct {
	caller          *types.Address
	callee          *types.Address
	ledger          *ledger.Ledger
	transactionData *pb.TransactionData
	nonce           int64
	logger          logrus.FieldLogger
}

// NewContext creates a context of wasm instance
func NewContext(tx pb.Transaction, data *pb.TransactionData, ledger *ledger.Ledger, logger logrus.FieldLogger) *Context {
	return &Context{
		caller:          tx.GetFrom(),
		callee:          tx.GetTo(),
		ledger:          ledger,
		transactionData: data,
		nonce:           int64(tx.GetNonce()),
		logger:          logger,
	}
}

// Caller returns the tx caller address
func (ctx *Context) Caller() string {
	return ctx.caller.String()
}

// Callee returns the tx callee address
func (ctx *Context) Callee() string {
	return ctx.callee.String()
}

// Logger returns the log instance
func (ctx *Context) Logger() logrus.FieldLogger {
	return ctx.logger
}
