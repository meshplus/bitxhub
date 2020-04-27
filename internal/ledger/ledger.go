package ledger

import (
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/meshplus/bitxhub/pkg/storage/leveldb"
	"github.com/sirupsen/logrus"
)

var _ Ledger = (*ChainLedger)(nil)

var (
	ErrorRollbackTohigherNumber = fmt.Errorf("rollback to higher blockchain height")
)

type ChainLedger struct {
	logger          logrus.FieldLogger
	blockchainStore storage.Storage
	ldb             storage.Storage
	height          uint64
	events          map[string][]*pb.Event
	accounts        map[string]*Account
	prevJournalHash types.Hash

	chainMutex sync.RWMutex
	chainMeta  *pb.ChainMeta
}

// New create a new ledger instance
func New(repoRoot string, blockchainStore storage.Storage, logger logrus.FieldLogger) (*ChainLedger, error) {
	ldb, err := leveldb.New(repo.GetStoragePath(repoRoot, "ledger"))
	if err != nil {
		return nil, fmt.Errorf("create tm-leveldb: %w", err)
	}

	chainMeta, err := loadChainMeta(blockchainStore)
	if err != nil {
		return nil, fmt.Errorf("load chain meta: %w", err)
	}

	height, blockJournal, err := getLatestJournal(ldb)
	if err != nil {
		return nil, fmt.Errorf("get journal height: %w", err)
	}

	if height < chainMeta.Height {
		// TODO(xcc): how to handle this case
		panic("state tree height is less than blockchain height")
	}

	return &ChainLedger{
		logger:          logger,
		chainMeta:       chainMeta,
		blockchainStore: blockchainStore,
		ldb:             ldb,
		height:          height,
		events:          make(map[string][]*pb.Event, 10),
		accounts:        make(map[string]*Account),
		prevJournalHash: blockJournal.ChangedHash,
	}, nil
}

// Rollback rollback ledger to history version
func (l *ChainLedger) Rollback(height uint64) error {
	if l.height < height {
		return ErrorRollbackTohigherNumber
	}

	if l.height == height {
		return nil
	}

	// clean cache account
	l.Clear()

	for i := l.height; i > height; i-- {
		batch := l.ldb.NewBatch()
		blockJournal := getBlockJournal(i, l.ldb)

		for _, journal := range blockJournal.Journals {
			journal.revert(batch)
		}

		if err := batch.Commit(); err != nil {
			panic(err)
		}

		if err := l.ldb.Delete(compositeKey(journalKey, i)); err != nil {
			panic(err)
		}
	}

	height, journal, err := getLatestJournal(l.ldb)
	if err != nil {
		return fmt.Errorf("get journal during rollback: %w", err)
	}

	l.prevJournalHash = journal.ChangedHash

	l.height = height

	return nil
}

// AddEvent add ledger event
func (l *ChainLedger) AddEvent(event *pb.Event) {
	hash := event.TxHash.Hex()
	l.events[hash] = append(l.events[hash], event)
}

// Events return ledger events
func (l *ChainLedger) Events(txHash string) []*pb.Event {
	return l.events[txHash]
}

// Close close the ledger instance
func (l *ChainLedger) Close() {
	l.ldb.Close()
}
