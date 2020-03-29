package ledger

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/sirupsen/logrus"
	"github.com/tendermint/iavl"
	db "github.com/tendermint/tm-db"
	"github.com/wonderivan/logger"
)

var _ Ledger = (*ChainLedger)(nil)

var (
	ErrorRollbackTohigherNumber = fmt.Errorf("rollback to higher blockchain height")
)

const (
	defaultIAVLCacheSize = 10000
)

type ChainLedger struct {
	logger          logrus.FieldLogger
	blockchainStore storage.Storage
	ldb             db.DB
	tree            *iavl.MutableTree
	height          uint64
	events          map[string][]*pb.Event
	accounts        map[string]*Account
	modifiedAccount map[string]bool

	chainMutex sync.RWMutex
	chainMeta  *pb.ChainMeta
}

// New create a new ledger instance
func New(repoRoot string, blockchainStore storage.Storage, logger logrus.FieldLogger) (*ChainLedger, error) {
	ldb, err := db.NewGoLevelDB("ledger", repo.GetStoragePath(repoRoot))
	if err != nil {
		return nil, fmt.Errorf("create tm-leveldb: %w", err)
	}

	chainMeta, err := loadChainMeta(blockchainStore)
	if err != nil {
		return nil, fmt.Errorf("load chain meta: %w", err)
	}

	tree := iavl.NewMutableTree(db.NewPrefixDB(ldb, []byte(ledgerTreePrefix)), defaultIAVLCacheSize)
	height, err := tree.LoadVersionForOverwriting(int64(chainMeta.Height))
	if err != nil {
		return nil, fmt.Errorf("load state tree: %w", err)
	}

	if uint64(height) < chainMeta.Height {
		// TODO(xcc): how to handle this case
		panic("state tree height is less than blockchain height")
	}

	return &ChainLedger{
		logger:          logger,
		chainMeta:       chainMeta,
		blockchainStore: blockchainStore,
		ldb:             ldb,
		tree:            tree,
		events:          make(map[string][]*pb.Event, 10),
		accounts:        make(map[string]*Account),
		modifiedAccount: make(map[string]bool),
	}, nil
}

// Rollback rollback to history version
func (l *ChainLedger) Rollback(height uint64) error {
	if l.chainMeta.Height <= height {
		return ErrorRollbackTohigherNumber
	}

	block, err := l.GetBlock(height)
	if err != nil {
		return err
	}

	count, err := getInterchainTxCount(block.BlockHeader)
	if err != nil {
		return err
	}

	l.UpdateChainMeta(&pb.ChainMeta{
		Height:            height,
		BlockHash:         block.BlockHash,
		InterchainTxCount: count,
	})

	// clean cache account
	l.Clear()

	_, err = l.tree.LoadVersionForOverwriting(int64(height))
	if err != nil {
		return err
	}

	begin, end := bytesPrefix([]byte(accountKey))
	l.tree.IterateRange(begin, end, false, func(key []byte, value []byte) bool {
		arr := bytes.Split(key, []byte("-"))
		if len(arr) != 2 {
			logger.Info("wrong account key")
		}

		a := newAccount(l, l.ldb, types.String2Address(string(arr[1])))
		if err := a.Unmarshal(value); err != nil {
			logger.Error(err)
		}

		if _, err := a.tree.LoadVersionForOverwriting(a.Version); err != nil {
			logger.Error(err)
		}

		_, err = a.Commit()
		if err != nil {
			logger.Error(err)
		}

		return false
	})

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
