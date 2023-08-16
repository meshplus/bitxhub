package genesis

import (
	"time"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor"
	"github.com/axiomesh/axiom/internal/executor/system"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, nodes []*repo.NetworkNodes, primaryN uint64, lg *ledger.Ledger, executor executor.Executor) error {
	lg.PrepareBlock(nil, 1)

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
