package common

import (
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger"
)

const (
	// internal contract
	// EpochManagerContractAddr is the contract to used to manager chain epoch info
	EpochManagerContractAddr = "0x0000000000000000000000000000000000000001"

	// ProposalIDContractAddr is the contract to used to generate the proposal ID
	ProposalIDContractAddr = "0x0000000000000000000000000000000000001000"

	// system contract address range 0x1001-0xffff
	NodeManagerContractAddr    = "0x0000000000000000000000000000000000001001"
	CouncilManagerContractAddr = "0x0000000000000000000000000000000000001002"

	// Addr2NameContractAddr for unique name mapping to address
	Addr2NameContractAddr = "0x0000000000000000000000000000000000001003"
)

type SystemContractConfig struct {
	Logger logrus.FieldLogger
}

type SystemContractConstruct func(cfg *SystemContractConfig) SystemContract

type SystemContract interface {
	// Reset the state of the system contract
	Reset(ledger.StateLedger)

	// Run the system contract
	Run(*vm.Message) (*vm.ExecutionResult, error)

	// EstimateGas estimate the gas cost of the system contract
	EstimateGas(*types.CallArgs) (uint64, error)

	// CheckAndUpdateState check and update state
	CheckAndUpdateState(uint64, ledger.StateLedger)
}

func IsInSlice[T ~uint8 | ~string](value T, slice []T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}

type Log struct {
	Address *types.Address
	Topics  []*types.Hash
	Data    []byte
	Removed bool
}
