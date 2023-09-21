package solo

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/axiomesh/axiom-bft/mempool"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/order/common"
	"github.com/axiomesh/axiom-ledger/internal/order/precheck"
	"github.com/axiomesh/axiom-ledger/internal/order/precheck/mock_precheck"
	"github.com/axiomesh/axiom-ledger/internal/peermgr/mock_peermgr"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

const (
	poolSize        = 10
	adminBalance    = 100000000000000000
	batchTimeout    = 50 * time.Millisecond
	removeTxTimeout = 1 * time.Second
)

var validTxsCh = make(chan *precheck.ValidTxs, maxChanSize)

func mockSoloNode(t *testing.T, enableTimed bool) (*Node, error) {
	logger := log.NewWithModule("consensus")
	logger.Logger.SetLevel(logrus.DebugLevel)
	repoRoot := t.TempDir()
	r, err := repo.Load(repoRoot)
	require.Nil(t, err)
	cfg := r.OrderConfig

	recvCh := make(chan consensusEvent, maxChanSize)
	batchTimerMgr := NewTimerManager(recvCh, logger)
	batchTimerMgr.newTimer(Batch, batchTimeout)
	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	mockPrecheck := mock_precheck.NewMockMinPreCheck(mockCtl, validTxsCh)

	mempoolConf := mempool.Config{
		IsTimed:             r.Config.Genesis.EpochInfo.ConsensusParams.EnableTimedGenEmptyBlock,
		Logger:              &common.Logger{FieldLogger: logger},
		BatchSize:           r.Config.Genesis.EpochInfo.ConsensusParams.BlockMaxTxNum,
		PoolSize:            poolSize,
		ToleranceRemoveTime: removeTxTimeout,
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
	batchTimerMgr.newTimer(RemoveTx, mempoolConf.ToleranceRemoveTime)
	mempoolInst := mempool.NewMempool[types.Transaction, *types.Transaction](mempoolConf)
	ctx, cancel := context.WithCancel(context.Background())

	soloNode := &Node{
		config: &common.Config{
			Config: r.OrderConfig,
		},
		lastExec:         uint64(0),
		isTimed:          mempoolConf.IsTimed,
		noTxBatchTimeout: noTxBatchTimeout,
		batchTimeout:     cfg.Mempool.BatchTimeout.ToDuration(),
		commitC:          make(chan *common.CommitEvent, maxChanSize),
		blockCh:          make(chan *mempool.RequestHashBatch[types.Transaction, *types.Transaction], maxChanSize),
		mempool:          mempoolInst,
		batchMgr:         batchTimerMgr,
		peerMgr:          mockPeermgr,
		batchDigestM:     make(map[uint64]string),
		checkpoint:       10,
		recvCh:           recvCh,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		txPreCheck:       mockPrecheck,
	}
	return soloNode, nil
}
