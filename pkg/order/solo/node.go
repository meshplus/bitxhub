package solo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/etcdraft"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

type Node struct {
	ID        uint64
	commitC   chan *pb.CommitEvent         // block channel
	logger    logrus.FieldLogger           // logger
	mempool   mempool.MemPool              // transaction pool
	proposeC  chan *raftproto.RequestBatch // proposed listenReadyBlock, input channel
	stateC    chan *mempool.ChainState
	txCache   *mempool.TxCache // cache the transactions received from api
	batchMgr  *etcdraft.BatchTimer
	lastExec  uint64              // the index of the last-applied block
	packSize  int                 // maximum number of transaction packages
	blockTick time.Duration       // block packed period
	peerMgr   peermgr.PeerManager // network manager

	ctx    context.Context
	cancel context.CancelFunc
	sync.RWMutex
}

func (n *Node) Start() error {
	go n.txCache.ListenEvent()
	go n.listenReadyBlock()
	return nil
}

func (n *Node) Stop() {
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
	n.txCache.RecvTxC <- tx
	return nil
}

func (n *Node) Commit() chan *pb.CommitEvent {
	return n.commitC
}

func (n *Node) Step(msg []byte) error {
	return nil
}

func (n *Node) Ready() error {
	return nil
}

func (n *Node) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {
	state := &mempool.ChainState{
		Height:     height,
		BlockHash:  blockHash,
		TxHashList: txHashList,
	}
	n.stateC <- state
}

func (n *Node) Quorum() uint64 {
	return 1
}

func NewNode(opts ...order.Option) (order.Order, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("new leveldb: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	batchTimeout, memConfig, err := generateSoloConfig(config.RepoRoot)
	mempoolConf := &mempool.Config{
		ID:              config.ID,
		ChainHeight:     config.Applied,
		Logger:          config.Logger,
		StoragePath:     config.StoragePath,
		GetAccountNonce: config.GetAccountNonce,

		BatchSize:      memConfig.BatchSize,
		PoolSize:       memConfig.PoolSize,
		TxSliceSize:    memConfig.TxSliceSize,
		TxSliceTimeout: memConfig.TxSliceTimeout,
	}
	batchC := make(chan *raftproto.RequestBatch)
	mempoolInst, err := mempool.NewMempool(mempoolConf)
	if err != nil {
		return nil, fmt.Errorf("create mempool instance: %w", err)
	}
	txCache := mempool.NewTxCache(mempoolConf.TxSliceTimeout, mempoolConf.TxSliceSize, config.Logger)
	batchTimerMgr := etcdraft.NewTimer(batchTimeout, config.Logger)
	soloNode := &Node{
		ID:       config.ID,
		commitC:  make(chan *pb.CommitEvent, 1024),
		stateC:   make(chan *mempool.ChainState),
		lastExec: config.Applied,
		mempool:  mempoolInst,
		txCache:  txCache,
		batchMgr: batchTimerMgr,
		peerMgr:  config.PeerMgr,
		proposeC: batchC,
		logger:   config.Logger,
		ctx:      ctx,
		cancel:   cancel,
	}
	soloNode.logger.Infof("SOLO lastExec = %d", soloNode.lastExec)
	soloNode.logger.Infof("SOLO batch timeout = %v", batchTimeout)
	return soloNode, nil
}

// Schedule to collect txs to the listenReadyBlock channel
func (n *Node) listenReadyBlock() {
	go func() {
		for {
			select {
			case proposal := <-n.proposeC:
				n.logger.WithFields(logrus.Fields{
					"proposal_height": proposal.Height,
					"tx_count":        len(proposal.TxList),
				}).Debugf("Receive proposal from mempool")

				if proposal.Height != n.lastExec+1 {
					n.logger.Warningf("Expects to execute seq=%d, but get seq=%d, ignore it", n.lastExec+1, proposal.Height)
					return
				}
				n.logger.Infof("======== Call execute, height=%d", proposal.Height)
				block := &pb.Block{
					BlockHeader: &pb.BlockHeader{
						Version:   []byte("1.0.0"),
						Number:    proposal.Height,
						Timestamp: time.Now().UnixNano(),
					},
					Transactions: proposal.TxList,
				}
				localList := make([]bool, len(proposal.TxList))
				for i := 0; i < len(proposal.TxList); i++ {
					localList[i] = true
				}
				executeEvent := &pb.CommitEvent{
					Block:     block,
					LocalList: localList,
				}
				n.commitC <- executeEvent
				n.lastExec++
			}
		}
	}()

	for {
		select {
		case <-n.ctx.Done():
			n.logger.Info("----- Exit listen ready block loop -----")
			return

		case txSet := <-n.txCache.TxSetC:
			// start batch timer when this node receives the first transaction
			if !n.batchMgr.IsBatchTimerActive() {
				n.batchMgr.StartBatchTimer()
			}
			if batch := n.mempool.ProcessTransactions(txSet.TxList, true, true); batch != nil {
				n.batchMgr.StopBatchTimer()
				n.proposeC <- batch
			}

		case state := <-n.stateC:
			if state.Height%10 == 0 {
				n.logger.WithFields(logrus.Fields{
					"height": state.Height,
					"hash":   state.BlockHash.String(),
				}).Info("Report checkpoint")
			}
			n.mempool.CommitTransactions(state)

		case <-n.batchMgr.BatchTimeoutEvent():
			n.batchMgr.StopBatchTimer()
			n.logger.Debug("Batch timer expired, try to create a batch")
			if n.mempool.HasPendingRequest() {
				if batch := n.mempool.GenerateBlock(); batch != nil {
					n.postProposal(batch)
				}
			} else {
				n.logger.Debug("The length of priorityIndex is 0, skip the batch timer")
			}
		}
	}
}

func (n *Node) postProposal(batch *raftproto.RequestBatch) {
	n.proposeC <- batch
	n.batchMgr.StartBatchTimer()
}
