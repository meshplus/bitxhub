package genesis

import (
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, nodes []*repo.NetworkNodes, primaryN uint64, lg *ledger.Ledger, executor executor.Executor) error {
	lg.PrepareBlock(nil, 1)

	accounts, stateRoot := lg.FlushDirtyData()

	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number:      1,
			StateRoot:   stateRoot,
			TxRoot:      &types.Hash{},
			ReceiptRoot: &types.Hash{},
			ParentHash:  &types.Hash{},
			Bloom:       &types.Bloom{},
			Timestamp:   time.Now().Unix(),
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
