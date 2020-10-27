package ledger

import (
	"fmt"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/internal/storages/blockfile"
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
	bf              *blockfile.BlockFile
	minJnlHeight    uint64
	maxJnlHeight    uint64
	events          sync.Map
	accounts        map[string]*Account
	accountCache    *AccountCache
	prevJnlHash     *types.Hash
	repo            *repo.Repo

	chainMutex sync.RWMutex
	chainMeta  *pb.ChainMeta

	journalMutex sync.RWMutex
	lock         sync.RWMutex
}

type BlockData struct {
	Block          *pb.Block
	Receipts       []*pb.Receipt
	Accounts       map[string]*Account
	Journal        *BlockJournal
	InterchainMeta *pb.InterchainMeta
}

// New create a new ledger instance
func New(repo *repo.Repo, blockchainStore storage.Storage, ldb storage.Storage, bf *blockfile.BlockFile, accountCache *AccountCache, logger logrus.FieldLogger) (*ChainLedger, error) {
	chainMeta, err := loadChainMeta(blockchainStore)
	if err != nil {
		return nil, fmt.Errorf("load chain meta: %w", err)
	}

	minJnlHeight, maxJnlHeight := getJournalRange(ldb)

	prevJnlHash := &types.Hash{}
	if maxJnlHeight != 0 {
		blockJournal := getBlockJournal(maxJnlHeight, ldb)
		if blockJournal == nil {
			return nil, fmt.Errorf("get empty block journal for block: %d", maxJnlHeight)
		}
		prevJnlHash = blockJournal.ChangedHash
	}

	if accountCache == nil {
		accountCache, err = NewAccountCache()
		if err != nil {
			return nil, err
		}
	}

	ledger := &ChainLedger{
		repo:            repo,
		logger:          logger,
		chainMeta:       chainMeta,
		blockchainStore: blockchainStore,
		ldb:             ldb,
		bf:              bf,
		minJnlHeight:    minJnlHeight,
		maxJnlHeight:    maxJnlHeight,
		accounts:        make(map[string]*Account),
		accountCache:    accountCache,
		prevJnlHash:     prevJnlHash,
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

func (l *ChainLedger) AccountCache() *AccountCache {
	return l.accountCache
}

// PersistBlockData persists block data
func (l *ChainLedger) PersistBlockData(blockData *BlockData) {
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

	PersistBlockDuration.Observe(float64(time.Since(current)) / float64(time.Second))
}

// Rollback rollback ledger to history version
func (l *ChainLedger) Rollback(height uint64) error {
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
func (l *ChainLedger) AddEvent(event *pb.Event) {
	var events []*pb.Event
	hash := event.TxHash.String()
	value, ok := l.events.Load(hash)
	if ok {
		events = value.([]*pb.Event)
	}
	events = append(events, event)
	l.events.Store(hash, events)
}

// Events return ledger events
func (l *ChainLedger) Events(txHash string) []*pb.Event {
	events, ok := l.events.Load(txHash)
	if !ok {
		return nil
	}
	return events.([]*pb.Event)
}

// Close close the ledger instance
func (l *ChainLedger) Close() {
	l.ldb.Close()
	l.blockchainStore.Close()
	l.bf.Close()
}
