package ledger

import (
	"fmt"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/sirupsen/logrus"
)

var _ Ledger = (*ChainLedger)(nil)

var (
	ErrorRollbackToHigherNumber  = fmt.Errorf("rollback to higher blockchain height")
	ErrorRollbackWithoutJournal  = fmt.Errorf("rollback to blockchain height without journal")
	ErrorRollbackTooMuch         = fmt.Errorf("rollback too much block")
	ErrorRemoveJournalOutOfRange = fmt.Errorf("remove journal out of range")
)

type ChainLedger struct {
	logger          logrus.FieldLogger
	blockchainStore storage.Storage
	ldb             storage.Storage
	minJnlHeight    uint64
	maxJnlHeight    uint64
	events          map[string][]*pb.Event
	accounts        map[string]*Account
	accountCache    *AccountCache
	prevJnlHash     types.Hash

	chainMutex sync.RWMutex
	chainMeta  *pb.ChainMeta

	journalMutex sync.RWMutex
	readOnly     bool
}

type BlockData struct {
	Block          *pb.Block
	Receipts       []*pb.Receipt
	Accounts       map[string]*Account
	Journal        *BlockJournal
	InterchainMeta *pb.InterchainMeta
}

// New create a new ledger instance
func New(repoRoot string, blockchainStore storage.Storage, accountCache *AccountCache, logger logrus.FieldLogger, readOnly bool) (*ChainLedger, error) {
	ldb, err := leveldb.New(repo.GetStoragePath(repoRoot, "ledger"))
	if err != nil {
		return nil, fmt.Errorf("create tm-leveldb: %w", err)
	}

	chainMeta, err := loadChainMeta(blockchainStore)
	if err != nil {
		return nil, fmt.Errorf("load chain meta: %w", err)
	}

	minJnlHeight, maxJnlHeight := getJournalRange(ldb)

	prevJnlHash := types.Hash{}
	if maxJnlHeight != 0 {
		blockJournal := getBlockJournal(maxJnlHeight, ldb)
		prevJnlHash = blockJournal.ChangedHash
	}

	ledger := &ChainLedger{
		logger:          logger,
		chainMeta:       chainMeta,
		blockchainStore: blockchainStore,
		ldb:             ldb,
		minJnlHeight:    minJnlHeight,
		maxJnlHeight:    maxJnlHeight,
		events:          make(map[string][]*pb.Event, 10),
		accounts:        make(map[string]*Account),
		accountCache:    accountCache,
		prevJnlHash:     prevJnlHash,
		readOnly:        readOnly,
	}

	height := maxJnlHeight
	if maxJnlHeight > chainMeta.Height {
		height = chainMeta.Height
	}

	if err := ledger.Rollback(height); err != nil {
		return nil, err
	}

	return ledger, nil
}

// PersistBlockData persists block data
func (l *ChainLedger) PersistBlockData(blockData *BlockData) error {
	if l.readOnly {
		return writeToReadOnlyErr()
	}

	current := time.Now()
	block := blockData.Block
	receipts := blockData.Receipts
	accounts := blockData.Accounts
	journal := blockData.Journal
	meta := blockData.InterchainMeta

	if err := l.Commit(block.BlockHeader.Number, accounts, journal); err != nil {
		panic(err)
	}

	if err := l.PersistExecutionResult(block, receipts, meta); err != nil {
		panic(err)
	}

	l.logger.WithFields(logrus.Fields{
		"height": block.BlockHeader.Number,
		"hash":   block.BlockHash.ShortString(),
		"count":  len(block.Transactions),
	}).Info("Persist block")
	PersistBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))

	return nil
}

// Rollback rollback ledger to history version
func (l *ChainLedger) Rollback(height uint64) error {
	if l.readOnly {
		return writeToReadOnlyErr()
	}

	if err := l.rollbackState(height); err != nil {
		return err
	}

	if err := l.rollbackBlockChain(height); err != nil {
		return err
	}

	return nil
}

// RemoveJournalsBeforeBlock removes ledger journals whose block number < height
func (l *ChainLedger) RemoveJournalsBeforeBlock(height uint64) error {
	if l.readOnly {
		return writeToReadOnlyErr()
	}

	l.journalMutex.Lock()
	defer l.journalMutex.Unlock()

	if height > l.maxJnlHeight {
		return ErrorRemoveJournalOutOfRange
	}

	if height <= l.minJnlHeight {
		return nil
	}

	batch := l.ldb.NewBatch()
	for i := l.minJnlHeight; i < height; i++ {
		batch.Delete(compositeKey(journalKey, i))
	}
	batch.Put(compositeKey(journalKey, minHeightStr), marshalHeight(height))
	batch.Commit()

	l.minJnlHeight = height

	return nil
}

// AddEvent add ledger event
func (l *ChainLedger) AddEvent(event *pb.Event) error {
	if l.readOnly {
		return writeToReadOnlyErr()
	}

	hash := event.TxHash.Hex()
	l.events[hash] = append(l.events[hash], event)

	return nil
}

// Events return ledger events
func (l *ChainLedger) Events(txHash string) []*pb.Event {
	return l.events[txHash]
}

// Close close the ledger instance
func (l *ChainLedger) Close() {
	l.ldb.Close()
}
