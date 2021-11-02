package ledger

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

type Ledger struct {
	ledger.ChainLedger
	ledger.StateLedger
}

type BlockData struct {
	Block          *pb.Block
	Receipts       []*pb.Receipt
	Accounts       map[string]ledger.IAccount
	InterchainMeta *pb.InterchainMeta
	TxHashList     []*types.Hash
}

func New(repo *repo.Repo, blockchainStore storage.Storage, ldb stateStorage, bf *blockfile.BlockFile, accountCache *AccountCache, logger logrus.FieldLogger) (*Ledger, error) {
	chainLedger, err := NewChainLedgerImpl(blockchainStore, bf, repo, logger)
	if err != nil {
		return nil, fmt.Errorf("init chain ledger failed: %w", err)
	}

	meta := chainLedger.GetChainMeta()

	var stateLedger ledger.StateLedger

	switch v := ldb.(type) {
	case storage.Storage:
		stateLedger, err = NewSimpleLedger(repo, ldb.(storage.Storage), accountCache, logger)
		if err != nil {
			return nil, fmt.Errorf("init state ledger failed: %w", err)
		}
	case ethdb.Database:
		db := state.NewDatabaseWithConfig(ldb.(ethdb.Database), &trie.Config{
			Cache:     256,
			Journal:   "",
			Preimages: true,
		})

		root := &types.Hash{}
		if meta.Height > 0 {
			block, err := chainLedger.GetBlock(meta.Height)
			if err != nil {
				return nil, fmt.Errorf("get block with height %d failed: %w", meta.Height, err)
			}
			root = block.BlockHeader.StateRoot
		}

		stateLedger, err = ledger.New(root, db, logger)
		if err != nil {
			return nil, fmt.Errorf("init state ledger failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknow storage type %T, expect simple or historical", v)
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
	meta := blockData.InterchainMeta

	err := l.StateLedger.Commit(block.BlockHeader.Number, accounts, block.BlockHeader.StateRoot)
	if err != nil {
		panic(err)
	}

	if err := l.ChainLedger.PersistExecutionResult(block, receipts, meta); err != nil {
		panic(err)
	}

	PersistBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))
}

// Rollback rollback ledger to history version
func (l *Ledger) Rollback(height uint64) error {
	if err := l.StateLedger.RollbackState(height); err != nil {
		return fmt.Errorf("rollback state to height %d failed: %w", height, err)
	}

	if err := l.ChainLedger.RollbackBlockChain(height); err != nil {
		return fmt.Errorf("rollback block to height %d failed: %w", height, err)
	}

	return nil
}

func (l *Ledger) Close() {
	l.ChainLedger.Close()
	l.StateLedger.Close()
}

type stateStorage interface{}

func OpenStateDB(file string, typ string) (stateStorage, error) {
	var storage stateStorage
	var err error

	if typ == "simple" {
		storage, err = leveldb.New(file)
		if err != nil {
			return nil, fmt.Errorf("init leveldb failed: %w", err)
		}
	} else if typ == "complex" {
		storage, err = rawdb.NewLevelDBDatabase(file, 0, 0, "", false)
		if err != nil {
			return nil, fmt.Errorf("init rawdb failed: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unknow storage type %s, expect simple or complex", typ)
	}

	return storage, nil
}
