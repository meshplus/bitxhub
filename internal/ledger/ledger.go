package ledger

import (
	"fmt"
	"sync"

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
	"github.com/syndtr/goleveldb/leveldb/opt"
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
	// current := time.Now()
	block := blockData.Block
	receipts := blockData.Receipts
	accounts := blockData.Accounts
	meta := blockData.InterchainMeta

	// persist StateLedger and ChainLedger concurrently
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		err := l.StateLedger.Commit(block.BlockHeader.Number, accounts, block.BlockHeader.StateRoot)
		if err != nil {
			panic(err)
		}
		wg.Done()
	}()
	go func() {
		if err := l.ChainLedger.PersistExecutionResult(block, receipts, meta); err != nil {
			panic(err)
		}
		wg.Done()
	}()
	wg.Wait()

	// PersistBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))
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

const (
	SimpleLedgerTyp  = "simple"  // use custom DB
	ComplexLedgerTyp = "complex" // use DB defined by Ethereum
)

type stateStorage interface{}

// OpenStateDB open db that storage state. Why return 'stateStorage' type?
// Because simple return 'storage.Storage' type, and complex return 'ethdb.Database' type.
func OpenStateDB(path string, ledgerConf *repo.Ledger) (stateStorage, error) {
	var s stateStorage
	var err error

	if ledgerConf.Type == SimpleLedgerTyp {
		// simple: new custom DB, return 'storage.Storage' type
		s, err = NewLevelDB(path, ledgerConf.LeveldbType,
			ledgerConf.GetLeveldbWriteBuffer(), ledgerConf.GetMultiLdbThreshold())
		if err != nil {
			return nil, fmt.Errorf("init leveldb failed: %w", err)
		}
	} else if ledgerConf.Type == ComplexLedgerTyp {
		// complex: new DB defined by Ethereum, return 'ethdb.Database' type
		s, err = rawdb.NewLevelDBDatabase(path, 0, 0, "", false)
		if err != nil {
			return nil, fmt.Errorf("init rawdb failed: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unknow storage type %s, expect simple or complex", ledgerConf.Type)
	}

	return s, nil
}

// OpenChainDB open db that storage chain meta, e.g. tx -> blockHeight, blockHash -> blockHeight.
func OpenChainDB(path string, ledgerConf *repo.Ledger) (storage.Storage, error) {
	s, err := NewLevelDB(path, ledgerConf.LeveldbType,
		ledgerConf.GetLeveldbWriteBuffer(), ledgerConf.GetMultiLdbThreshold())
	if err != nil {
		return nil, fmt.Errorf("init leveldb failed: %w", err)
	}
	return s, nil
}

const (
	NormalLeveldb            = "normal" // a normal leveldb provided by 'github.com/syndtr/goleveldb/'
	MultiLeveldb             = "multi"  // a multi-layer leveldb that encapsulation of 'github.com/syndtr/goleveldb/'
	defaultWriteBuffer       = 4 * opt.MiB
	defaultMultiLdbThreshold = 100 * opt.GiB
)

// NewLevelDB create leveldb under path.
//
// 'writeBuffer' defines maximum size of a 'memdb' for leveldb. The default value is 4MiB.
// Larger values increase performance, especially during bulk loads.
//
// 'multiLdbThreshold' defines maximum size of each layer in multi-leveldb. The default value is 100GiB.
func NewLevelDB(path string, typ string, writeBuffer int64, multiLdbThreshold int64) (storage.Storage, error) {
	var (
		s   storage.Storage
		err error
	)

	// check writeBuffer
	if writeBuffer < 0 {
		return nil, fmt.Errorf("the 'writeBuffer' value of leveldb is error: %d", writeBuffer)
	} else if writeBuffer > int64(^uint(0)>>1) {
		return nil, fmt.Errorf("the 'writeBuffer' value of leveldb exceed INT_MAX: %d", writeBuffer)
	} else if writeBuffer == 0 {
		writeBuffer = defaultWriteBuffer
	}

	// check multiLdbThreshold
	if multiLdbThreshold < 0 {
		return nil, fmt.Errorf("the 'multiLdbThreshold' value of multi-leveldb is error: %d", multiLdbThreshold)
	} else if multiLdbThreshold == 0 {
		multiLdbThreshold = defaultMultiLdbThreshold
	}

	// generate option of leveldb
	multiplier := int(writeBuffer) / opt.DefaultWriteBuffer
	ldbOpt := &opt.Options{
		WriteBuffer:            opt.DefaultWriteBuffer * multiplier,
		CompactionTableSize:    opt.DefaultCompactionTableSize * multiplier,
		CompactionTotalSize:    opt.DefaultCompactionTotalSize * multiplier,
		CompactionL0Trigger:    opt.DefaultCompactionL0Trigger * multiplier,
		WriteL0SlowdownTrigger: opt.DefaultWriteL0SlowdownTrigger * multiplier,
		WriteL0PauseTrigger:    opt.DefaultWriteL0PauseTrigger * multiplier,
	}

	if typ == NormalLeveldb {
		s, err = leveldb.NewWithOpt(path, ldbOpt)
		if err != nil {
			return nil, fmt.Errorf("init normal leveldb failed: %w", err)
		}
	} else if typ == MultiLeveldb {
		s, err = leveldb.NewMultiLdb(path, ldbOpt, multiLdbThreshold)
		if err != nil {
			return nil, fmt.Errorf("init normal leveldb failed: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unknow leveldb type: %s, expect: %s or %s", typ, NormalLeveldb, MultiLeveldb)
	}

	return s, nil
}
