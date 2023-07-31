package ledger

import (
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

var _ ledger.StateLedger = (*StateLedger)(nil)

var (
	ErrorRollbackToHigherNumber  = fmt.Errorf("rollback to higher blockchain height")
	ErrorRollbackWithoutJournal  = fmt.Errorf("rollback to blockchain height without journal")
	ErrorRollbackTooMuch         = fmt.Errorf("rollback too much block")
	ErrorRemoveJournalOutOfRange = fmt.Errorf("remove journal out of range")
)

type revision struct {
	id           int
	changerIndex int
}

type StateLedger struct {
	logger        logrus.FieldLogger
	ldb           storage.Storage
	minJnlHeight  uint64
	maxJnlHeight  uint64
	events        sync.Map
	accounts      map[string]ledger.IAccount
	accountCache  *AccountCache
	blockJournals sync.Map
	prevJnlHash   *types.Hash
	repo          *repo.Repo
	blockHeight   uint64
	thash         *types.Hash
	txIndex       int

	journalMutex sync.RWMutex
	lock         sync.RWMutex

	validRevisions []revision
	nextRevisionId int
	changer        *stateChanger

	accessList *ledger.AccessList
	preimages  map[types.Hash][]byte
	refund     uint64
	logs       *evmLogs

	transientStorage transientStorage
}

func (l *StateLedger) Copy() ledger.StateLedger {
	return l
}

func (l *StateLedger) Finalise(b bool) {
	l.ClearChangerAndRefund()
}

// New create a new ledger instance
func NewSimpleLedger(repo *repo.Repo, ldb storage.Storage, accountCache *AccountCache, logger logrus.FieldLogger) (ledger.StateLedger, error) {
	var err error
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
			return nil, fmt.Errorf("init account cache failed: %w", err)
		}
	}

	ledger := &StateLedger{
		repo:         repo,
		logger:       logger,
		ldb:          ldb,
		minJnlHeight: minJnlHeight,
		maxJnlHeight: maxJnlHeight,
		accounts:     make(map[string]ledger.IAccount),
		accountCache: accountCache,
		prevJnlHash:  prevJnlHash,
		preimages:    make(map[types.Hash][]byte),
		changer:      newChanger(),
		accessList:   ledger.NewAccessList(),
		logs:         NewEvmLogs(),
	}

	return ledger, nil
}

func (l *StateLedger) AccountCache() *AccountCache {
	return l.accountCache
}

func (l *StateLedger) SetTxContext(thash *types.Hash, ti int) {
	l.thash = thash
	l.txIndex = ti
}

// removeJournalsBeforeBlock removes ledger journals whose block number < height
func (l *StateLedger) removeJournalsBeforeBlock(height uint64) error {
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
func (l *StateLedger) AddEvent(event *types.Event) {
	var events []*types.Event
	hash := event.TxHash
	if hash == nil {
		hash = l.logs.thash
	}

	value, ok := l.events.Load(hash.String())
	if ok {
		events = value.([]*types.Event)
	}
	events = append(events, event)
	l.events.Store(hash.String(), events)
}

// Events return ledger events
func (l *StateLedger) Events(txHash string) []*types.Event {
	events, ok := l.events.Load(txHash)
	if !ok {
		return nil
	}
	return events.([]*types.Event)
}

// Close close the ledger instance
func (l *StateLedger) Close() {
	l.ldb.Close()
}
