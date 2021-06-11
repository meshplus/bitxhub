package genesis

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/bytesutil"
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
	body, err := json.Marshal(genesis.Admins)
	if err != nil {
		return err
	}

	admin, err := json.Marshal(genesis.Dider)
	if err != nil {
		return err
	}

	lg.SetState(constant.RoleContractAddr.Address(), []byte("admin-roles"), body)
	lg.SetState(constant.MethodRegistryContractAddr.Address(), []byte("admin-method"), admin)
	lg.SetState(constant.DIDRegistryContractAddr.Address(), []byte("admin-did"), admin)

	balance, _ := new(big.Int).SetString("100000000000000000000000000000000000", 10)
	for _, admin := range genesis.Admins {
		lg.SetBalance(types.NewAddressByStr(admin.Address), balance)
	}

	for k, v := range genesis.Strategy {
		lg.SetState(constant.GovernanceContractAddr.Address(), []byte(k), []byte(v))
	}

	for i := 0; i < int(primaryN); i++ {
		node := &node_mgr.Node{
			Id:       nodes[i].ID,
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
		lg.SetState(constant.NodeManagerContractAddr.Address(), []byte(fmt.Sprintf("%s-%s", node_mgr.NODE_PID_PREFIX, node.Pid)), []byte(strconv.Itoa(int(node.Id))))
		lg.SetState(constant.NodeManagerContractAddr.Address(), []byte(fmt.Sprintf("%s-%s", node_mgr.NODEPREFIX, strconv.Itoa(int(node.Id)))), nodeData)
	}

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
