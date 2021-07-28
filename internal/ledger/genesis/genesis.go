package genesis

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/meshplus/bitxhub/internal/executor/contracts"

	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, nodes []*repo.NetworkNodes, primaryN uint64, lg *ledger.Ledger, executor executor.Executor) error {
	lg.PrepareBlock(nil, 1)

	for _, ad := range genesis.Admins {
		admin := &contracts.Role{
			ID:       ad.Address,
			RoleType: contracts.GovernanceAdmin,
			Weight:   ad.Weight,
			Status:   governance.GovernanceAvailable,
		}
		adminData, err := json.Marshal(admin)
		if err != nil {
			return err
		}
		lg.SetState(constant.RoleContractAddr.Address(), []byte(fmt.Sprintf("%s-%s", contracts.ROLEPREFIX, admin.ID)), adminData)
	}

	admin, err := json.Marshal(genesis.Dider)
	if err != nil {
		return err
	}
	lg.SetState(constant.MethodRegistryContractAddr.Address(), []byte("admin-method"), admin)
	lg.SetState(constant.DIDRegistryContractAddr.Address(), []byte("admin-did"), admin)

	balance, _ := new(big.Int).SetString(genesis.Balance, 10)
	for _, admin := range genesis.Admins {
		lg.SetBalance(types.NewAddressByStr(admin.Address), balance)
	}
	lg.SetState(constant.RoleContractAddr.Address(), []byte(contracts.GenesisBalance), []byte(genesis.Balance))

	for k, v := range genesis.Strategy {
		lg.SetState(constant.GovernanceContractAddr.Address(), []byte(k), []byte(v))
	}

	for i := 0; i < int(primaryN); i++ {
		node := &node_mgr.Node{
			VPNodeId: nodes[i].ID,
			Pid:      nodes[i].Pid,
			Account:  nodes[i].Account,
			NodeType: node_mgr.VPNode,
			Primary:  true,
			Status:   governance.GovernanceAvailable,
		}
		nodeData, err := json.Marshal(node)
		if err != nil {
			return err
		}
		lg.SetState(constant.NodeManagerContractAddr.Address(), []byte(fmt.Sprintf("%s-%d", node_mgr.VP_NODE_ID_PREFIX, node.VPNodeId)), []byte(node.Pid))
		lg.SetState(constant.NodeManagerContractAddr.Address(), []byte(fmt.Sprintf("%s-%s", node_mgr.NODEPREFIX, node.Pid)), nodeData)
	}

	lg.SetState(constant.InterchainContractAddr.Address(), []byte(contracts.BitXHubID), []byte(fmt.Sprintf("%d", genesis.ChainID)))

	// avoid being deleted by complex state ledger
	for addr := range executor.GetBoltContracts() {
		lg.SetNonce(types.NewAddressByStr(addr), 1)
	}

	accounts, stateRoot := lg.FlushDirtyData()

	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number:    1,
			StateRoot: stateRoot,
			Bloom:     &types.Bloom{},
		},
		Transactions: &pb.Transactions{},
	}
	block.BlockHash = block.Hash()
	blockData := &ledger.BlockData{
		Block:          block,
		Receipts:       nil,
		Accounts:       accounts,
		InterchainMeta: &pb.InterchainMeta{},
	}

	lg.PersistBlockData(blockData)

	return nil
}
