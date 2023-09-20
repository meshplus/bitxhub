package ledger

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/pkg/repo"
)

var _ StateLedger = (*StateLedgerImpl)(nil)

var (
	ErrorRollbackToHigherNumber  = errors.New("rollback to higher blockchain height")
	ErrorRollbackWithoutJournal  = errors.New("rollback to blockchain height without journal")
	ErrorRollbackTooMuch         = errors.New("rollback too much block")
	ErrorRemoveJournalOutOfRange = errors.New("remove journal out of range")
)

type revision struct {
	id           int
	changerIndex int
}

type StateLedgerImpl struct {
	logger        logrus.FieldLogger
	ldb           storage.Storage
	minJnlHeight  uint64
	maxJnlHeight  uint64
	accounts      map[string]IAccount
	accountCache  *AccountCache
	blockJournals map[string]*BlockJournal
	prevJnlHash   *types.Hash
	repo          *repo.Repo
	blockHeight   uint64
	thash         *types.Hash
	txIndex       int

	validRevisions []revision
	nextRevisionId int
	changer        *stateChanger

	accessList *AccessList
	preimages  map[types.Hash][]byte
	refund     uint64
	logs       *evmLogs

	transientStorage transientStorage
}

// Copy copy state ledger
// Attention this is shallow copy
func (l *StateLedgerImpl) Copy() StateLedger {
	// TODO: if readonly, not need cache, opt it
	copyLedger, err := NewSimpleLedger(&repo.Repo{Config: l.repo.Config}, l.ldb, nil, l.logger)
	if err != nil {
		l.logger.Errorf("copy ledger error: %w", err)
		return nil
	}

	return copyLedger
}

func (l *StateLedgerImpl) Finalise() {
	l.ClearChangerAndRefund()
}

// NewSimpleLedger create a new ledger instance
func NewSimpleLedger(repo *repo.Repo, ldb storage.Storage, accountCache *AccountCache, logger logrus.FieldLogger) (StateLedger, error) {
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

	ledger := &StateLedgerImpl{
		repo:          repo,
		logger:        logger,
		ldb:           ldb,
		minJnlHeight:  minJnlHeight,
		maxJnlHeight:  maxJnlHeight,
		accounts:      make(map[string]IAccount),
		accountCache:  accountCache,
		prevJnlHash:   prevJnlHash,
		preimages:     make(map[types.Hash][]byte),
		changer:       NewChanger(),
		accessList:    NewAccessList(),
		logs:          NewEvmLogs(),
		blockJournals: make(map[string]*BlockJournal),
	}

	return ledger, nil
}

func (l *StateLedgerImpl) AccountCache() *AccountCache {
	return l.accountCache
}

func (l *StateLedgerImpl) SetTxContext(thash *types.Hash, ti int) {
	l.thash = thash
	l.txIndex = ti
}

// removeJournalsBeforeBlock removes ledger journals whose block number < height
func (l *StateLedgerImpl) removeJournalsBeforeBlock(height uint64) error {
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

// Close close the ledger instance
func (l *StateLedgerImpl) Close() {
	_ = l.ldb.Close()
}
