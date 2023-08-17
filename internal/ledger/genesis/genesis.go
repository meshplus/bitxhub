package genesis

import (
	"math/big"
	"time"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor"
	"github.com/axiomesh/axiom/internal/executor/system"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, lg *ledger.Ledger, executor executor.Executor) error {
	lg.PrepareBlock(nil, 1)

	balance, _ := new(big.Int).SetString(genesis.Balance, 10)
	for _, addr := range genesis.Accounts {
		lg.SetBalance(types.NewAddressByStr(addr), balance)
	}
	err := system.InitGenesisData(genesis, lg)
	if err != nil {
		return err
	}

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
			Epoch:       genesis.EpochInfo.Epoch,
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
