package solo

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Node struct {
	ID uint64
	sync.RWMutex
	height             uint64             // current block height
	commitC            chan *pb.Block     // block channel
	logger             logrus.FieldLogger // logger
	reqLookUp          *order.ReqLookUp   // bloom filter
	getTransactionFunc func(hash *types.Hash) (*pb.Transaction, error)
	mempool            mempool.MemPool       // transaction pool
	proposeC           chan *raftproto.Ready // proposed listenReadyBlock, input channel

	packSize  int           // maximum number of transaction packages
	blockTick time.Duration // block packed period

	ctx    context.Context
	cancel context.CancelFunc
}

func (n *Node) Start() error {
	go n.listenReadyBlock()
	if err := n.mempool.Start(); err != nil {
		return err
	}
	n.mempool.UpdateLeader(n.ID)
	return nil
}

func (n *Node) Stop() {
	n.mempool.Stop()
	n.cancel()
}

func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	return n.mempool.GetPendingNonceByAccount(account)
}

func (n *Node) DelNode(delID uint64) error {
	return nil
}

func (n *Node) Prepare(tx *pb.Transaction) error {
	if err := n.Ready(); err != nil {
		return err
	}
	return n.mempool.RecvTransaction(tx)
}

func (n *Node) Commit() chan *pb.Block {
	return n.commitC
}

func (n *Node) Step(ctx context.Context, msg []byte) error {
	return nil
}

func (n *Node) Ready() error {
	return nil
}

func (n *Node) ReportState(height uint64, hash *types.Hash) {
	if err := n.reqLookUp.Build(); err != nil {
		n.logger.Errorf("bloom filter persistence error：", err)
	}

	if height%10 == 0 {
		n.logger.WithFields(logrus.Fields{
			"height": height,
			"hash":   hash.String(),
		}).Info("Report checkpoint")
	}
}

func (n *Node) Quorum() uint64 {
	return 1
}

func NewNode(opts ...order.Option) (order.Order, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}
	storage, err := leveldb.New(config.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("new leveldb: %w", err)
	}
	reqLookUp, err := order.NewReqLookUp(storage, config.Logger)
	if err != nil {
		return nil, fmt.Errorf("new bloom filter: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	mockCtl := gomock.NewController(&testing.T{})
	peerMgr := mock_peermgr.NewMockPeerManager(mockCtl)
	peerMgr.EXPECT().Peers().Return(map[uint64]*pb.VpInfo{}).AnyTimes()
	memConfig, err := generateMempoolConfig(config.RepoRoot)
	mempoolConf := &mempool.Config{
		ID:                 config.ID,
		ChainHeight:        config.Applied,
		GetTransactionFunc: config.GetTransactionFunc,
		PeerMgr:            peerMgr,
		Logger:             config.Logger,

		BatchSize:      memConfig.BatchSize,
		BatchTick:      memConfig.BatchTick,
		PoolSize:       memConfig.PoolSize,
		TxSliceSize:    memConfig.TxSliceSize,
		FetchTimeout:   memConfig.FetchTimeout,
		TxSliceTimeout: memConfig.TxSliceTimeout,
	}
	batchC := make(chan *raftproto.Ready, 10)
	mempoolInst := mempool.NewMempool(mempoolConf, storage, batchC)
	return &Node{
		ID:                 config.ID,
		height:             config.Applied,
		commitC:            make(chan *pb.Block, 1024),
		reqLookUp:          reqLookUp,
		getTransactionFunc: config.GetTransactionFunc,
		mempool:            mempoolInst,
		proposeC:           batchC,
		logger:             config.Logger,
		ctx:                ctx,
		cancel:             cancel,
	}, nil
}

// Schedule to collect txs to the listenReadyBlock channel
func (n *Node) listenReadyBlock() {
	for {
		select {
		case <-n.ctx.Done():
			n.logger.Info("----- Exit listen ready block loop -----")
			return
		case proposal := <-n.proposeC:
			n.logger.WithFields(logrus.Fields{
				"proposal_height": proposal.Height,
				"tx_count":        len(proposal.TxHashes),
			}).Debugf("Receive proposal from mempool")
			// collect txs from proposalC
			_, txs := n.mempool.GetBlockByHashList(proposal)
			n.height++

			block := &pb.Block{
				BlockHeader: &pb.BlockHeader{
					Version:   []byte("1.0.0"),
					Number:    n.height,
					Timestamp: time.Now().UnixNano(),
				},
				Transactions: txs,
			}
			n.mempool.CommitTransactions(proposal)
			n.mempool.IncreaseChainHeight()
			n.commitC <- block
		}
	}
}

func generateMempoolConfig(repoRoot string) (*MempoolConfig, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	mempoolConf := &MempoolConfig{}
	mempoolConf.BatchSize = readConfig.RAFT.MempoolConfig.BatchSize
	mempoolConf.PoolSize = readConfig.RAFT.MempoolConfig.PoolSize
	mempoolConf.TxSliceSize = readConfig.RAFT.MempoolConfig.TxSliceSize
	mempoolConf.BatchTick = readConfig.RAFT.MempoolConfig.BatchTick
	mempoolConf.FetchTimeout = readConfig.RAFT.MempoolConfig.FetchTimeout
	mempoolConf.TxSliceTimeout = readConfig.RAFT.MempoolConfig.TxSliceTimeout
	return mempoolConf, nil
}

func readConfig(repoRoot string) (*RAFTConfig, error) {
	v := viper.New()
	v.SetConfigFile(filepath.Join(repoRoot, "order.toml"))
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &RAFTConfig{}

	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
