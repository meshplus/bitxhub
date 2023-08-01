package solo

import (
	"context"
	"testing"
	"time"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/pkg/order/mempool"
	"github.com/axiomesh/axiom/pkg/peermgr/mock_peermgr"
	"github.com/golang/mock/gomock"
)

func mockSoloNode(t *testing.T, enableTimed bool) (*Node, error) {
	logger := log.NewWithModule("consensus")
	txCache := mempool.NewTxCache(25*time.Millisecond, uint64(2), logger)
	repoRoot := "./testdata/"
	batchTimeout, memConfig, timedGenBlock, _ := generateSoloConfig(repoRoot)
	eventCh := make(chan batchTimeoutEvent)
	batchTimerMgr := NewTimerManager(eventCh, logger)
	batchTimerMgr.newTimer(Batch, batchTimeout)
	batchTimerMgr.newTimer(NoTxBatch, timedGenBlock.NoTxBatchTimeout)
	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	mempoolConf := &mempool.Config{
		ID:               uint64(1),
		IsTimed:          timedGenBlock.Enable,
		NoTxBatchTimeout: timedGenBlock.NoTxBatchTimeout,
		Logger:           logger,

		BatchSize:      memConfig.BatchSize,
		PoolSize:       memConfig.PoolSize,
		TxSliceSize:    memConfig.TxSliceSize,
		TxSliceTimeout: memConfig.TxSliceTimeout,
		GetAccountNonce: func(address *types.Address) uint64 {
			return 0
		},
	}
	if enableTimed == true {
		mempoolConf.IsTimed = true
	}

	mempoolInst, err := mempool.NewMempool(mempoolConf)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())

	soloNode := &Node{
		ID:               uint64(1),
		lastExec:         uint64(0),
		isTimed:          mempoolConf.IsTimed,
		noTxBatchTimeout: mempoolConf.NoTxBatchTimeout,
		batchTimeout:     batchTimeout,
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
