package governance

import (
	"github.com/meshplus/bitxhub/internal/executor/system/common"
	vm "github.com/meshplus/eth-kit/evm"
	"github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

var _ common.SystemContract = (*NodeManager)(nil)

type NodeManager struct {
	gov *Governance

	statedb ledger.StateDB
}

func NewNodeManager(logger logrus.FieldLogger) *NodeManager {
	gov, err := NewGov(NodeUpdate, logger)
	if err != nil {
		panic(err)
	}

	return &NodeManager{
		gov: gov,
	}
}

func (nm *NodeManager) Reset(statedb ledger.StateDB) {
	nm.statedb = statedb
}

func (nm *NodeManager) Run(msg *vm.Message) (*vm.ExecutionResult, error) {
	// parse method and arguments from msg payload

	// TODO: execute, then generate proposal id

	return &vm.ExecutionResult{
		UsedGas:    0,
		Err:        nil,
		ReturnData: []byte{1},
	}, nil
}
