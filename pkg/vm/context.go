package vm

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/sirupsen/logrus"
)

// Context represents the context of wasm
type Context struct {
	Caller               *types.Address
	Callee               *types.Address
	CurrentCaller        *types.Address
	Ledger               ledger.Ledger
	TransactionIndex     uint64
	TransactionHash      *types.Hash
	TransactionData      *pb.TransactionData
	TransactionTimestamp int64
	Nonce                uint64
	Logger               logrus.FieldLogger
}

// NewContext creates a context of wasm instance
func NewContext(tx *pb.Transaction, txIndex uint64, data *pb.TransactionData, ledger ledger.Ledger, logger logrus.FieldLogger) *Context {
	return &Context{
		Caller:               tx.From,
		Callee:               tx.To,
		CurrentCaller:        tx.From,
		Ledger:               ledger,
		TransactionIndex:     txIndex,
		TransactionHash:      tx.TransactionHash,
		TransactionData:      data,
		TransactionTimestamp: tx.Timestamp,
		Nonce:                tx.Nonce,
		Logger:               logger,
	}
}
