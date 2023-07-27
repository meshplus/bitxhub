package solo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/meshplus/bitxhub/pkg/order/mempool/proto"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

type Node struct {
	ID           uint64
	isTimed      bool
	commitC      chan *pb.CommitEvent // block channel
	logger       logrus.FieldLogger   // logger
	mempool      mempool.MemPool      // transaction pool
	recvCh       chan consensusEvent  // receive message from consensus engine
	stateC       chan *mempool.ChainState
	txCache      *mempool.TxCache // cache the transactions received from api
	batchMgr     *BatchTimer
	blockTimeout time.Duration            // generate block period
	lastExec     uint64                   // the index of the last-applied block
	peerMgr      peermgr.OrderPeerManager // network manager

	ctx    context.Context
	cancel context.CancelFunc
	sync.RWMutex
}

func (n *Node) GetPendingTxByHash(hash *types.Hash) pb.Transaction {
	getTxReq := &mempool.GetTxReq{
		Hash: hash,
		Tx:   make(chan pb.Transaction),
	}
	n.recvCh <- getTxReq

	return <-getTxReq.Tx
}

func (n *Node) Start() error {
	n.ctx, n.cancel = context.WithCancel(context.Background())
	go n.txCache.ListenEvent(n.ctx)
	go n.listenReadyBlock()
	return nil
}

func (n *Node) Stop() {
	n.cancel()
	n.logger.Info("consensus stopped")
}

func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	return n.mempool.GetPendingNonceByAccount(account)
}

func (n *Node) DelNode(uint64) error {
	return nil
}

func (n *Node) Prepare(tx pb.Transaction) error {
	if err := n.Ready(); err != nil {
		return fmt.Errorf("node get ready failed: %w", err)
	}
	txWithResp := &mempool.TxWithResp{
		Tx: tx,
		Ch: make(chan bool),
	}
	n.txCache.TxRespC <- txWithResp
	<-txWithResp.Ch
	return nil
}

func (n *Node) Commit() chan *pb.CommitEvent {
	return n.commitC
}

func (n *Node) Step([]byte) error {
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

func (n *Node) SubscribeTxEvent(ch chan<- pb.Transactions) event.Subscription {
	return n.mempool.SubscribeTxEvent(ch)
}

func NewNode(opts ...order.Option) (order.Order, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}
	batchTimeout, memConfig, timedGenBlock, err := generateSoloConfig(config.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("generate solo txpool config: %w", err)
	}
	mempoolConf := &mempool.Config{
		ID:              config.ID,
		IsTimed:         timedGenBlock.Enable,
		BlockTimeout:    timedGenBlock.BlockTimeout,
		ChainHeight:     config.Applied,
		Logger:          config.Logger,
		StoragePath:     config.StoragePath,
		GetAccountNonce: config.GetAccountNonce,

		BatchSize:      memConfig.BatchSize,
		PoolSize:       memConfig.PoolSize,
		TxSliceSize:    memConfig.TxSliceSize,
		TxSliceTimeout: memConfig.TxSliceTimeout,
	}
	mempoolInst, err := mempool.NewMempool(mempoolConf)
	if err != nil {
		return nil, fmt.Errorf("create mempool instance: %w", err)
	}
	txCache := mempool.NewTxCache(mempoolConf.TxSliceTimeout, mempoolConf.TxSliceSize, config.Logger)
	batchTimerMgr := NewTimer(batchTimeout, config.Logger)

	soloNode := &Node{
		ID:           config.ID,
		isTimed:      mempoolConf.IsTimed,
		blockTimeout: mempoolConf.BlockTimeout,
		commitC:      make(chan *pb.CommitEvent, 1024),
		stateC:       make(chan *mempool.ChainState),
		recvCh:       make(chan consensusEvent),
		lastExec:     config.Applied,
		mempool:      mempoolInst,
		txCache:      txCache,
		batchMgr:     batchTimerMgr,
		peerMgr:      config.PeerMgr,
		logger:       config.Logger,
	}
	soloNode.logger.Infof("SOLO lastExec = %d", soloNode.lastExec)
	soloNode.logger.Infof("SOLO batch timeout = %v", batchTimeout)
	return soloNode, nil
}

func (n *Node) SubmitTxsFromRemote(tsx [][]byte) error {
	return nil
}

// Schedule to collect txs to the listenReadyBlock channel
func (n *Node) listenReadyBlock() {
	var blockTicker <-chan time.Time
	if n.isTimed && blockTicker == nil {
		blockTicker = time.Tick(n.blockTimeout)
	}
	go func() {
		for {
			select {
			case ev := <-n.recvCh:
				switch e := ev.(type) {
				case *proto.RequestBatch:
					n.logger.WithFields(logrus.Fields{
						"proposal_height": e.Height,
						"tx_count":        len(e.TxList.Transactions),
					}).Debugf("Receive proposal from mempool")

					if e.Height != n.lastExec+1 {
						n.logger.Warningf("Expects to execute seq=%d, but get seq=%d, ignore it", n.lastExec+1, e.Height)
						continue
					}
					n.logger.Infof("======== Call execute, height=%d", e.Height)
					block := &pb.Block{
						BlockHeader: &pb.BlockHeader{
							Version:   []byte("1.0.0"),
							Number:    e.Height,
							Timestamp: e.Timestamp,
						},
						Transactions: e.TxList,
					}
					localList := make([]bool, len(e.TxList.Transactions))
					for i := 0; i < len(e.TxList.Transactions); i++ {
						localList[i] = true
					}
					executeEvent := &pb.CommitEvent{
						Block:     block,
						LocalList: localList,
					}
					n.commitC <- executeEvent
					n.lastExec++

				case *mempool.GetTxReq:
					e.Tx <- n.mempool.GetTransaction(e.Hash)
				default:
					n.logger.Errorf("Can't recognize event type of %v.", e)
				}
			}
		}
	}()

	for {

		select {
		case <-n.ctx.Done():
			n.logger.Info("----- Exit listen ready block loop -----")
			return

		case txWithResp := <-n.txCache.TxRespC:
			if !n.isTimed {
				// start batch timer when this node receives the first transaction
				if !n.batchMgr.IsBatchTimerActive() {
					n.batchMgr.StartBatchTimer()
				}
				if batch := n.mempool.ProcessTransactions([]pb.Transaction{txWithResp.Tx}, true, true); batch != nil {
					n.batchMgr.StopBatchTimer()
					n.recvCh <- batch
				}
			} else {
				n.mempool.ProcessTransactions([]pb.Transaction{txWithResp.Tx}, true, true)
			}
			txWithResp.Ch <- true
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
					n.batchMgr.StartBatchTimer()
				}
			} else {
				n.logger.Debug("The length of priorityIndex is 0, skip the batch timer")
			}

		case <-blockTicker:
			if !n.mempool.HasPendingRequest() {
				n.logger.Debug("start create empty block")
			}
			batch := n.mempool.GenerateBlock()
			n.postProposal(batch)
		}
	}
}

func (n *Node) postProposal(batch *proto.RequestBatch) {
	n.recvCh <- batch
}
