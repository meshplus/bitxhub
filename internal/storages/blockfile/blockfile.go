package blockfile

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/prometheus/tsdb/fileutil"
	"github.com/sirupsen/logrus"
)

type BlockFile struct {
	blocks uint64 // Number of blocks

	tables       map[string]*BlockTable // Data tables for stroring blocks
	instanceLock fileutil.Releaser      // File-system lock to prevent double opens

	logger    logrus.FieldLogger
	closeOnce sync.Once
}

func NewBlockFile(repoRoot string, logger logrus.FieldLogger) (*BlockFile, error) {
	if info, err := os.Lstat(repoRoot); !os.IsNotExist(err) {
		if info.Mode()&os.ModeSymlink != 0 {
			logger.WithField("path", repoRoot).Error("Symbolic link is not supported")
			return nil, fmt.Errorf("symbolic link datadir is not supported")
		}
	}
	lock, _, err := fileutil.Flock(filepath.Join(repoRoot, "FLOCK"))
	if err != nil {
		return nil, err
	}
	blockfile := &BlockFile{
		tables:       make(map[string]*BlockTable),
		instanceLock: lock,
		logger:       logger,
	}
	for name := range BlockFileSchema {
		table, err := newTable(repoRoot, name, 2*1000*1000*1000, logger)
		if err != nil {
			for _, table := range blockfile.tables {
				table.Close()
			}
			_ = lock.Release()
			return nil, err
		}
		blockfile.tables[name] = table
	}
	if err := blockfile.repair(); err != nil {
		for _, table := range blockfile.tables {
			table.Close()
		}
		_ = lock.Release()
		return nil, err
	}

	return blockfile, nil
}

func (bf *BlockFile) Blocks() (uint64, error) {
	return atomic.LoadUint64(&bf.blocks), nil
}

func (bf *BlockFile) Get(kind string, number uint64) ([]byte, error) {
	if table := bf.tables[kind]; table != nil {
		return table.Retrieve(number - 1)
	}
	return nil, fmt.Errorf("unknown table")
}

func (bf *BlockFile) AppendBlock(number uint64, hash, body, receipts, transactions, interchainMetas []byte) (err error) {
	if atomic.LoadUint64(&bf.blocks) != number {
		return fmt.Errorf("the append operation is out-order")
	}
	defer func() {
		if err != nil {
			rerr := bf.repair()
			if rerr != nil {
				bf.logger.WithField("err", err).Errorf("Failed to repair blockfile")
			}
			bf.logger.WithFields(logrus.Fields{
				"number": number,
				"err":    err,
			}).Info("Append block failed")
		}
	}()
	if err := bf.tables[BlockFileHashTable].Append(bf.blocks, hash); err != nil {
		bf.logger.WithFields(logrus.Fields{
			"number": bf.blocks,
			"hash":   hash,
			"err":    err,
		}).Error("Failed to append block hash")
		return err
	}
	if err := bf.tables[BlockFileBodiesTable].Append(bf.blocks, body); err != nil {
		bf.logger.WithFields(logrus.Fields{
			"number": bf.blocks,
			"hash":   hash,
			"err":    err,
		}).Error("Failed to append block body")
		return err
	}
	if err := bf.tables[BlockFileTXsTable].Append(bf.blocks, transactions); err != nil {
		bf.logger.WithFields(logrus.Fields{
			"number": bf.blocks,
			"hash":   hash,
			"err":    err,
		}).Error("Failed to append block transactions")
		return err
	}
	if err := bf.tables[BlockFileReceiptTable].Append(bf.blocks, receipts); err != nil {
		bf.logger.WithFields(logrus.Fields{
			"number": bf.blocks,
			"hash":   hash,
			"err":    err,
		}).Error("Failed to append block receipt")
		return err
	}
	if err := bf.tables[BlockFileInterchainTable].Append(bf.blocks, interchainMetas); err != nil {
		bf.logger.WithFields(logrus.Fields{
			"number": bf.blocks,
			"hash":   hash,
			"err":    err,
		}).Error("Failed to append block interchain metas")
		return err
	}
	atomic.AddUint64(&bf.blocks, 1) // Only modify atomically
	return nil
}

func (bf *BlockFile) TruncateBlocks(items uint64) error {
	if atomic.LoadUint64(&bf.blocks) <= items {
		return nil
	}
	for _, table := range bf.tables {
		if err := table.truncate(items); err != nil {
			return err
		}
	}
	atomic.StoreUint64(&bf.blocks, items)
	return nil
}

// repair truncates all data tables to the same length.
func (bf *BlockFile) repair() error {
	min := uint64(math.MaxUint64)
	for _, table := range bf.tables {
		items := atomic.LoadUint64(&table.items)
		if min > items {
			min = items
		}
	}
	for _, table := range bf.tables {
		if err := table.truncate(min); err != nil {
			return err
		}
	}
	atomic.StoreUint64(&bf.blocks, min)
	return nil
}

func (bf *BlockFile) Close() error {
	var errs []error
	bf.closeOnce.Do(func() {
		for _, table := range bf.tables {
			if err := table.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		if err := bf.instanceLock.Release(); err != nil {
			errs = append(errs, err)
		}
	})
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}
