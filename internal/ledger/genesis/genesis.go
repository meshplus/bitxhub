package genesis

import (
	"math/big"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, nodes []*repo.NetworkNodes, primaryN uint64, lg *ledger.Ledger, executor executor.Executor) error {
	lg.PrepareBlock(nil, 1)

	balance, _ := new(big.Int).SetString(genesis.Balance, 10)
	for _, admin := range genesis.Admins {
		lg.SetBalance(types.NewAddressByStr(admin.Address), balance)
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
