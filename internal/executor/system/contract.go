package system

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/executor/system/common"
	"github.com/meshplus/bitxhub/internal/executor/system/governance"
	"github.com/sirupsen/logrus"
)

// Addr2Contract is address to system contract
var Addr2Contract map[types.Address]common.SystemContract

const (
	// system contract address range 0x1000-0xffff
	NodeManagerContractAddr = "0x0000000000000000000000000000000000001000"
)

func Initialize(logger logrus.FieldLogger) {
	Addr2Contract = map[types.Address]common.SystemContract{
		*types.NewAddressByStr(NodeManagerContractAddr): governance.NewNodeManager(logger),
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
