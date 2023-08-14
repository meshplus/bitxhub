package common

import (
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger"
)

const (
	// ProposalIDContractAddr is the contract to used to generate the proposal ID
	ProposalIDContractAddr = "0x0000000000000000000000000000000000001000"
	// system contract address range 0x1001-0xffff
	NodeManagerContractAddr    = "0x0000000000000000000000000000000000001001"
	CouncilManagerContractAddr = "0x0000000000000000000000000000000000001002"
	// node members address, used for admitting connections between nodes
	NodeMemberContractAddr = "0x0000000000000000000000000000000000001003"
)

type SystemContract interface {
	Reset(ledger.StateLedger)
	Run(*vm.Message) (*vm.ExecutionResult, error)
}

func IsInSlice[T ~uint8 | ~string](value T, slice []T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}
