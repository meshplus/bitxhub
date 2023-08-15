package system

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/executor/system/governance"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
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

func SetNodeMember(genesis *repo.Genesis, lg *ledger.Ledger) error {
	//read member config, write to Ledger
	c, err := json.Marshal(genesis.Members)
	if err != nil {
		return err
	}
	lg.SetState(types.NewAddressByStr(common.NodeManagerContractAddr), []byte(common.NodeManagerContractAddr), c)

	return nil
}

func IsExistNodeMember(peerID string, lg *ledger.Ledger) (bool, error) {
	var isExist = false
	success, data := lg.GetState(types.NewAddressByStr(common.NodeManagerContractAddr), []byte(common.NodeManagerContractAddr))
	if success {
		stringData := strings.Split(string(data), ",")
		for _, nodeID := range stringData {
			if peerID == nodeID {
				isExist = true
				break
			}
		}
	} else {
		return isExist, fmt.Errorf("get nodeMember err")
	}

	return isExist, nil
}
