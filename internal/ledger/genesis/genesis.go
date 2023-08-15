package genesis

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/executor/system/governance"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, nodes []*repo.NetworkNodes, primaryN uint64, lg *ledger.Ledger, executor executor.Executor) error {
	lg.PrepareBlock(nil, 1)

	balance, _ := new(big.Int).SetString(genesis.Balance, 10)
	council := &governance.Council{}
	for _, admin := range genesis.Admins {
		lg.SetBalance(types.NewAddressByStr(admin.Address), balance)

		council.Members = append(council.Members, &governance.CouncilMember{
			Address: admin.Address,
			Weight:  admin.Weight,
		})
	}
	account := lg.GetOrCreateAccount(types.NewAddressByStr(common.CouncilManagerContractAddr))
	b, err := json.Marshal(council)
	if err != nil {
		return err
	}
	account.SetState([]byte(governance.CouncilKey), b)

	//read member config, write to Ledger
	c, err := json.Marshal(genesis.Members)
	if err != nil {
		return err
	}
	lg.SetState(types.NewAddressByStr(common.NodeManagerContractAddr), []byte(common.NodeManagerContractAddr), c)
	accounts, stateRoot := lg.FlushDirtyData()

	block := &types.Block{
		BlockHeader: &types.BlockHeader{
			Number:      1,
			StateRoot:   stateRoot,
			TxRoot:      &types.Hash{},
			ReceiptRoot: &types.Hash{},
			ParentHash:  &types.Hash{},
			Timestamp:   time.Now().Unix(),
			GasPrice:    int64(genesis.GasPrice),
			Version:     []byte{},
			Bloom:       new(types.Bloom),
		},
		Transactions: []*types.Transaction{},
	}
	block.BlockHash = block.Hash()
	blockData := &ledger.BlockData{
		Block:    block,
		Accounts: accounts,
	}

	lg.PersistBlockData(blockData)

	return nil
}
