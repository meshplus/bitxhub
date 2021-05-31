package boltvm

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/sirupsen/logrus"
)

type Context struct {
	caller           *types.Address
	callee           *types.Address
	ledger           *ledger.Ledger
	transactionIndex uint64
	transactionHash  *types.Hash
	logger           logrus.FieldLogger
}

func NewContext(tx pb.Transaction, txIndex uint64, data *pb.TransactionData, ledger *ledger.Ledger, logger logrus.FieldLogger) *Context {
	return &Context{
		caller:           tx.GetFrom(),
		callee:           tx.GetTo(),
		ledger:           ledger,
		transactionIndex: txIndex,
		transactionHash:  tx.GetHash(),
		logger:           logger,
	}
}

func (ctx *Context) Caller() string {
	return ctx.caller.String()
}

func (ctx *Context) Callee() string {
	return ctx.callee.String()
}

func (ctx *Context) TransactionIndex() uint64 {
	return ctx.transactionIndex
}

func (ctx *Context) TransactionHash() *types.Hash {
	return ctx.transactionHash
}

func (ctx *Context) Logger() logrus.FieldLogger {
	return ctx.logger
}
