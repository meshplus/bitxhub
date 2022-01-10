package solo

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order/etcdraft"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"testing"
	"time"
)

func mockSoloNode(t *testing.T, enableTimed bool) (*Node, error) {
	logger := log.NewWithModule("consensus")
	txCache := mempool.NewTxCache(25*time.Millisecond, uint64(2), logger)
	repoRoot := "./testdata/"
	batchTimeout, memConfig, timedGenBlock, _ := generateSoloConfig(repoRoot)
	batchTimerMgr := etcdraft.NewTimer(batchTimeout, logger)
	mockCtl := gomock.NewController(t)
	mockPeermgr := mock_peermgr.NewMockPeerManager(mockCtl)
	mempoolConf := &mempool.Config{
		ID:           uint64(1),
		IsTimed:      timedGenBlock.Enable,
		BlockTimeout: timedGenBlock.BlockTimeout,
		Logger:       logger,

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
	batchC := make(chan *raftproto.RequestBatch)
	ctx, cancel := context.WithCancel(context.Background())

	soloNode := &Node{
		ID:           uint64(1),
		lastExec:     uint64(0),
		isTimed:      mempoolConf.IsTimed,
		blockTimeout: mempoolConf.BlockTimeout,
		commitC:      make(chan *pb.CommitEvent, 1024),
		stateC:       make(chan *mempool.ChainState),
		mempool:      mempoolInst,
		txCache:      txCache,
		batchMgr:     batchTimerMgr,
		peerMgr:      mockPeermgr,
		proposeC:     batchC,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
	}
	return soloNode, nil
}

func generateTx() pb.Transaction {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	from, _ := privKey.PublicKey().Address()
	tx := &pb.BxhTransaction{
		From:      from,
		To:        types.NewAddressByStr(to),
		Timestamp: time.Now().UnixNano(),
		Nonce:     0,
	}
	_ = tx.Sign(privKey)
	tx.TransactionHash = tx.Hash()
	return tx
}
