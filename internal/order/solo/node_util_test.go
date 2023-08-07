package solo

import (
	"context"
	"testing"
	"time"

	"github.com/axiomesh/axiom-bft/mempool"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/peermgr/mock_peermgr"
	"github.com/axiomesh/axiom/pkg/repo"
)

func mockSoloNode(t *testing.T, enableTimed bool) (*Node, error) {
	logger := log.NewWithModule("consensus")
	repoRoot := t.TempDir()
	r, err := repo.Load(repoRoot)
	require.Nil(t, err)
	cfg := r.OrderConfig

	recvCh := make(chan consensusEvent, maxChanSize)
	batchTimerMgr := NewTimerManager(recvCh, logger)
	batchTimerMgr.newTimer(Batch, cfg.Solo.Mempool.BatchTimeout.ToDuration())
	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	mempoolConf := mempool.Config{
		ID:        uint64(1),
		IsTimed:   cfg.TimedGenBlock.Enable,
		Logger:    &order.Logger{FieldLogger: logger},
		BatchSize: cfg.Solo.Mempool.BatchSize,
		PoolSize:  cfg.Solo.Mempool.PoolSize,
		GetAccountNonce: func(address string) uint64 {
			return 0
		},
	}
	var noTxBatchTimeout time.Duration
	if enableTimed {
		mempoolConf.IsTimed = true
		noTxBatchTimeout = 50 * time.Millisecond
	} else {
		mempoolConf.IsTimed = false
		noTxBatchTimeout = cfg.TimedGenBlock.NoTxBatchTimeout.ToDuration()
	}
	batchTimerMgr.newTimer(NoTxBatch, noTxBatchTimeout)
	mempoolInst := mempool.NewMempool[types.Transaction, *types.Transaction](mempoolConf)
	ctx, cancel := context.WithCancel(context.Background())

	soloNode := &Node{
		ID:               uint64(1),
		lastExec:         uint64(0),
		isTimed:          mempoolConf.IsTimed,
		noTxBatchTimeout: noTxBatchTimeout,
		batchTimeout:     cfg.Solo.Mempool.BatchTimeout.ToDuration(),
		commitC:          make(chan *types.CommitEvent, maxChanSize),
		stateC:           make(chan *chainState),
		blockCh:          make(chan *mempool.RequestHashBatch[types.Transaction, *types.Transaction], maxChanSize),
		mempool:          mempoolInst,
		batchMgr:         batchTimerMgr,
		peerMgr:          mockPeermgr,
		recvCh:           recvCh,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}
	return soloNode, nil
}
