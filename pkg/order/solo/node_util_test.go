package solo

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
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
		commitC:          make(chan *pb.CommitEvent, 1024),
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

func generateTx() pb.Transaction {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	from, _ := privKey.PublicKey().Address()
	tx := &pb.BxhTransaction{
		From:      from,
		To:        types.NewAddressByStr(to),
		Timestamp: time.Now().Unix(),
		Nonce:     0,
	}
	_ = tx.Sign(privKey)
	tx.TransactionHash = tx.Hash()
	return tx
}
