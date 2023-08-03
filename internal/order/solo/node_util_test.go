package solo

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order/mempool"
	"github.com/axiomesh/axiom/internal/peermgr/mock_peermgr"
	"github.com/axiomesh/axiom/pkg/repo"
)

func mockSoloNode(t *testing.T, enableTimed bool) (*Node, error) {
	logger := log.NewWithModule("consensus")
	txCache := mempool.NewTxCache(25*time.Millisecond, uint64(2), logger)
	repoRoot := t.TempDir()
	r, err := repo.Load(repoRoot)
	require.Nil(t, err)
	cfg := r.OrderConfig

	eventCh := make(chan batchTimeoutEvent)
	batchTimerMgr := NewTimerManager(eventCh, logger)
	batchTimerMgr.newTimer(Batch, cfg.Solo.Mempool.BatchTimeout.ToDuration())
	batchTimerMgr.newTimer(NoTxBatch, cfg.TimedGenBlock.NoTxBatchTimeout.ToDuration())
	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	mempoolConf := &mempool.Config{
		ID:               uint64(1),
		IsTimed:          cfg.TimedGenBlock.Enable,
		NoTxBatchTimeout: cfg.TimedGenBlock.NoTxBatchTimeout.ToDuration(),
		Logger:           logger,

		BatchSize:      cfg.Solo.Mempool.BatchSize,
		PoolSize:       cfg.Solo.Mempool.PoolSize,
		TxSliceSize:    cfg.Solo.Mempool.TxSliceSize,
		TxSliceTimeout: cfg.Solo.Mempool.TxSliceTimeout.ToDuration(),
		GetAccountNonce: func(address *types.Address) uint64 {
			return 0
		},
	}
	if enableTimed {
		mempoolConf.IsTimed = true
	}

	mempoolInst := mempool.NewMempool(mempoolConf)
	ctx, cancel := context.WithCancel(context.Background())

	soloNode := &Node{
		ID:               uint64(1),
		lastExec:         uint64(0),
		isTimed:          mempoolConf.IsTimed,
		noTxBatchTimeout: mempoolConf.NoTxBatchTimeout,
		batchTimeout:     cfg.Solo.Mempool.BatchTimeout.ToDuration(),
		commitC:          make(chan *types.CommitEvent, 1024),
		stateC:           make(chan *mempool.ChainState),
		mempool:          mempoolInst,
		txCache:          txCache,
		batchTimeoutCh:   eventCh,
		batchMgr:         batchTimerMgr,
		peerMgr:          mockPeermgr,
		recvCh:           make(chan consensusEvent),
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}
	return soloNode, nil
}
