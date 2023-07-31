package common

import (
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger"
)

type SystemContract interface {
	Reset(ledger.StateDB)
	Run(*vm.Message) (*vm.ExecutionResult, error)
}
