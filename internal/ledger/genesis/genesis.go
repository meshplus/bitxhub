package genesis

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/iancoleman/orderedmap"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, nodes []*repo.NetworkNodes, primaryN uint64, lg *ledger.Ledger, executor executor.Executor) error {
	lg.PrepareBlock(nil, 1)

	idMap := map[string]struct{}{}
	for _, ad := range genesis.Admins {
		idMap[ad.Address] = struct{}{}
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
		lg.SetState(constant.RoleContractAddr.Address(), []byte(contracts.RoleKey(admin.ID)), adminData)
	}
	idMapData, err := json.Marshal(idMap)
	if err != nil {
		return err
	}
	lg.SetState(constant.RoleContractAddr.Address(), []byte(contracts.RoleTypeKey(string(contracts.GovernanceAdmin))), idMapData)

	balance, _ := new(big.Int).SetString(genesis.Balance, 10)
	for _, admin := range genesis.Admins {
		lg.SetBalance(types.NewAddressByStr(admin.Address), balance)
	}
	lg.SetState(constant.RoleContractAddr.Address(), []byte(contracts.GenesisBalance), []byte(genesis.Balance))

	for k, v := range genesis.Strategy {
		lg.SetState(constant.GovernanceContractAddr.Address(), []byte(k), []byte(v))
	}

	nodePidMap := orderedmap.New()
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
		lg.SetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.VpNodeIdKey(strconv.Itoa(int(node.VPNodeId)))), []byte(node.Pid))
		lg.SetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.NodeKey(node.Pid)), nodeData)
		nodePidMap.Set(node.Pid, struct{}{})
	}
	nodePidMapData, err := json.Marshal(nodePidMap)
	if err != nil {
		return err
	}
	lg.SetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.NodeTypeKey(string(node_mgr.VPNode))), nodePidMapData)

	lg.SetState(constant.InterchainContractAddr.Address(), []byte(contracts.BitXHubID), []byte(fmt.Sprintf("%d", genesis.ChainID)))

	// avoid being deleted by complex state ledger
	for addr := range executor.GetBoltContracts() {
		lg.SetNonce(types.NewAddressByStr(addr), 1)
	}

	accounts, stateRoot := lg.FlushDirtyData()

	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number:      1,
			StateRoot:   stateRoot,
			TxRoot:      &types.Hash{},
			ReceiptRoot: &types.Hash{},
			ParentHash:  &types.Hash{},
			Bloom:       &types.Bloom{},
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
