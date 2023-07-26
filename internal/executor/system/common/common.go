package common

import (
	vm "github.com/meshplus/eth-kit/evm"
	"github.com/meshplus/eth-kit/ledger"
)

type SystemContract interface {
	Reset(ledger.StateDB)
	Run(*vm.Message) (*vm.ExecutionResult, error)
}
