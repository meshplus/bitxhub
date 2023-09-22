package solo

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/axiomesh/axiom-bft/txpool"
	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/consensus/common"
	"github.com/axiomesh/axiom-ledger/internal/consensus/precheck"
	"github.com/axiomesh/axiom-ledger/internal/consensus/precheck/mock_precheck"
	"github.com/axiomesh/axiom-ledger/internal/network/mock_network"
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
	cfg := r.ConsensusConfig

	recvCh := make(chan consensusEvent, maxChanSize)
	batchTimerMgr := NewTimerManager(recvCh, logger)
	batchTimerMgr.newTimer(Batch, batchTimeout)
	mockCtl := gomock.NewController(t)
	mockNetwork := mock_network.NewMockNetwork(mockCtl)
	mockPrecheck := mock_precheck.NewMockMinPreCheck(mockCtl, validTxsCh)

	txpoolConf := txpool.Config{
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
		txpoolConf.IsTimed = true
		noTxBatchTimeout = 50 * time.Millisecond
	} else {
		txpoolConf.IsTimed = false
		noTxBatchTimeout = cfg.TimedGenBlock.NoTxBatchTimeout.ToDuration()
	}
	batchTimerMgr.newTimer(NoTxBatch, noTxBatchTimeout)
	batchTimerMgr.newTimer(RemoveTx, txpoolConf.ToleranceRemoveTime)
	txpoolInst := txpool.NewTxPool[types.Transaction, *types.Transaction](txpoolConf)
	ctx, cancel := context.WithCancel(context.Background())

	soloNode := &Node{
		config: &common.Config{
			Config: r.ConsensusConfig,
		},
		lastExec:         uint64(0),
		isTimed:          txpoolConf.IsTimed,
		noTxBatchTimeout: noTxBatchTimeout,
		batchTimeout:     cfg.TxPool.BatchTimeout.ToDuration(),
		commitC:          make(chan *common.CommitEvent, maxChanSize),
		blockCh:          make(chan *txpool.RequestHashBatch[types.Transaction, *types.Transaction], maxChanSize),
		txpool:           txpoolInst,
		batchMgr:         batchTimerMgr,
		network:          mockNetwork,
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
