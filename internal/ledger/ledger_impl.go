package ledger

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

type Ledger struct {
	ChainLedger ChainLedger
	StateLedger StateLedger
}

type BlockData struct {
	Block      *types.Block
	Receipts   []*types.Receipt
	Accounts   map[string]IAccount
	TxHashList []*types.Hash
}

func New(repo *repo.Repo, blockchainStore storage.Storage, ldb storage.Storage, bf *blockfile.BlockFile, accountCache *AccountCache, logger logrus.FieldLogger) (*Ledger, error) {
	chainLedger, err := NewChainLedgerImpl(blockchainStore, bf, repo, logger)
	if err != nil {
		return nil, fmt.Errorf("init chain ledger failed: %w", err)
	}

	meta := chainLedger.GetChainMeta()

	var stateLedger StateLedger

	stateLedger, err = NewSimpleLedger(repo, ldb, accountCache, logger)
	if err != nil {
		return nil, fmt.Errorf("init state ledger failed: %w", err)
	}
	ledger := &Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	if err := ledger.Rollback(meta.Height); err != nil {
		return nil, fmt.Errorf("rollback ledger to height %d failed: %w", meta.Height, err)
	}

	return ledger, nil
}

// PersistBlockData persists block data
func (l *Ledger) PersistBlockData(blockData *BlockData) {
	current := time.Now()
	block := blockData.Block
	receipts := blockData.Receipts
	accounts := blockData.Accounts

	err := l.StateLedger.Commit(block.BlockHeader.Number, accounts, block.BlockHeader.StateRoot)
	if err != nil {
		panic(err)
	}

	if err := l.ChainLedger.PersistExecutionResult(block, receipts); err != nil {
		panic(err)
	}

	persistBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))
	blockHeightMetric.Set(float64(block.BlockHeader.Number))
}

// Rollback rollback ledger to history version
func (l *Ledger) Rollback(height uint64) error {
	if err := l.StateLedger.RollbackState(height); err != nil {
		return fmt.Errorf("rollback state to height %d failed: %w", height, err)
	}

	if err := l.ChainLedger.RollbackBlockChain(height); err != nil {
		return fmt.Errorf("rollback block to height %d failed: %w", height, err)
	}

	blockHeightMetric.Set(float64(height))
	return nil
}

func (l *Ledger) Close() {
	l.ChainLedger.Close()
	l.StateLedger.Close()
}

func (l *Ledger) NewView() *Ledger {
	return &Ledger{
		ChainLedger: l.ChainLedger,
		StateLedger: l.StateLedger.NewView(),
	}
}
