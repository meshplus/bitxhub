package system

import (
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/executor/system/governance"
	"github.com/sirupsen/logrus"
)

// Addr2Contract is address to system contract
var Addr2Contract map[types.Address]common.SystemContract

func Initialize(logger logrus.FieldLogger) {
	Addr2Contract = map[types.Address]common.SystemContract{
		*types.NewAddressByStr(common.NodeManagerContractAddr):    governance.NewNodeManager(logger),
		*types.NewAddressByStr(common.CouncilManagerContractAddr): governance.NewCouncilManager(logger),
	}
}

// GetSystemContract get system contract
// return true if system contract, false if not
func GetSystemContract(addr *types.Address) (common.SystemContract, bool) {
	if addr == nil {
		return nil, false
	}

	if contract, ok := Addr2Contract[*addr]; ok {
		return contract, true
	}
	return nil, false
}
