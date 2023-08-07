package solo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/order/mempool"
	"github.com/axiomesh/axiom/internal/peermgr"
)

// consensusEvent is a type meant to clearly convey that the return type or parameter to a function will be supplied to/from an events.Manager
type consensusEvent any

type Node struct {
	ID               uint64
	isTimed          bool
	commitC          chan *types.CommitEvent // block channel
	logger           logrus.FieldLogger      // logger
	mempool          mempool.MemPool         // transaction pool
	recvCh           chan consensusEvent     // receive message from consensus engine
	batchTimeoutCh   chan batchTimeoutEvent  // receive batch timeout message from consensus engine
	stateC           chan *mempool.ChainState
	txCache          *mempool.TxCache // cache the transactions received from api
	batchMgr         *timerManager
	noTxBatchTimeout time.Duration       // generate no-tx block period
	batchTimeout     time.Duration       // generate block period
	lastExec         uint64              // the index of the last-applied block
	peerMgr          peermgr.PeerManager // network manager

	ctx    context.Context
	cancel context.CancelFunc
	sync.RWMutex
}

func (n *Node) GetPendingTxByHash(hash *types.Hash) *types.Transaction {
	getTxReq := &mempool.GetTxReq{
		Hash: hash,
		Tx:   make(chan *types.Transaction),
	}
	n.recvCh <- getTxReq

	return <-getTxReq.Tx
}

func (n *Node) Start() error {
	n.ctx, n.cancel = context.WithCancel(context.Background())
	n.logger.Info("consensus started")
	if n.isTimed {
		n.batchMgr.startTimer(NoTxBatch)
	}
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

func (n *Node) Prepare(tx *types.Transaction) error {
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

func (n *Node) Commit() chan *types.CommitEvent {
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

func (n *Node) SubscribeTxEvent(ch chan<- []*types.Transaction) event.Subscription {
	return n.mempool.SubscribeTxEvent(ch)
}

func NewNode(opts ...order.Option) (order.Order, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}
	mempoolConf := &mempool.Config{
		ID:               config.ID,
		IsTimed:          config.Config.TimedGenBlock.Enable,
		NoTxBatchTimeout: config.Config.TimedGenBlock.NoTxBatchTimeout.ToDuration(),
		ChainHeight:      config.Applied,
		Logger:           config.Logger,
		StoragePath:      config.StoragePath,
		GetAccountNonce:  config.GetAccountNonce,

		BatchSize:      config.Config.Solo.Mempool.BatchSize,
		PoolSize:       config.Config.Solo.Mempool.PoolSize,
		TxSliceSize:    config.Config.Solo.Mempool.TxSliceSize,
		TxSliceTimeout: config.Config.Solo.Mempool.TxSliceTimeout.ToDuration(),
	}
	mempoolInst := mempool.NewMempool(mempoolConf)
	txCache := mempool.NewTxCache(mempoolConf.TxSliceTimeout, mempoolConf.TxSliceSize, config.Logger)
	// init batch timer manager
	batchTimeoutCh := make(chan batchTimeoutEvent, 1024)
	batchTimerMgr := NewTimerManager(batchTimeoutCh, config.Logger)
	batchTimerMgr.newTimer(Batch, config.Config.Solo.Mempool.BatchTimeout.ToDuration())
	batchTimerMgr.newTimer(NoTxBatch, mempoolConf.NoTxBatchTimeout)

	soloNode := &Node{
		ID:               config.ID,
		isTimed:          mempoolConf.IsTimed,
		noTxBatchTimeout: mempoolConf.NoTxBatchTimeout,
		batchTimeout:     config.Config.Solo.Mempool.BatchTimeout.ToDuration(),
		commitC:          make(chan *types.CommitEvent, 1024),
		stateC:           make(chan *mempool.ChainState),
		recvCh:           make(chan consensusEvent),
		batchTimeoutCh:   batchTimeoutCh,
		lastExec:         config.Applied,
		mempool:          mempoolInst,
		txCache:          txCache,
		batchMgr:         batchTimerMgr,
		peerMgr:          config.PeerMgr,
		logger:           config.Logger,
	}
	soloNode.logger.Infof("SOLO lastExec = %d", soloNode.lastExec)
	soloNode.logger.Infof("SOLO batch timeout = %v", config.Config.Solo.Mempool.BatchTimeout.ToDuration())
	return soloNode, nil
}

func (n *Node) SubmitTxsFromRemote(tsx [][]byte) error {
	return nil
}

// Schedule to collect txs to the listenReadyBlock channel
func (n *Node) listenReadyBlock() {
	go func() {
		// nolint
		for {
			select {
			case ev := <-n.recvCh:
				switch e := ev.(type) {
				case *mempool.RequestBatch:
					n.logger.WithFields(logrus.Fields{
						"proposal_height": e.Height,
						"tx_count":        len(e.TxList),
					}).Debugf("Receive proposal from mempool")

					if e.Height != n.lastExec+1 {
						n.logger.Warningf("Expects to execute seq=%d, but get seq=%d, ignore it", n.lastExec+1, e.Height)
						continue
					}
					n.logger.Infof("======== Call execute, height=%d", e.Height)
					block := &types.Block{
						BlockHeader: &types.BlockHeader{
							Version:   []byte("1.0.0"),
							Number:    e.Height,
							Timestamp: e.Timestamp,
						},
						Transactions: e.TxList,
					}
					localList := make([]bool, len(e.TxList))
					for i := 0; i < len(e.TxList); i++ {
						localList[i] = true
					}
					executeEvent := &types.CommitEvent{
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
			// stop no-tx batch timer when this node receives the first transaction
			n.batchMgr.stopTimer(NoTxBatch)
			// start batch timer when this node receives the first transaction
			if !n.batchMgr.isTimerActive(Batch) {
				n.batchMgr.startTimer(Batch)
			}
			if batch := n.mempool.ProcessTransactions([]*types.Transaction{txWithResp.Tx}, true, true); batch != nil {
				n.batchMgr.stopTimer(Batch)
				n.recvCh <- batch
				// start no-tx batch timer when this node handle the last transaction
				if n.isTimed && !n.mempool.HasPendingRequest() {
					if !n.batchMgr.isTimerActive(NoTxBatch) {
						n.batchMgr.startTimer(NoTxBatch)
					}
				}
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

		case event := <-n.batchTimeoutCh:
			switch event {
			case Batch:
				n.batchMgr.stopTimer(Batch)
				n.logger.Debug("Batch timer expired, try to create a batch")
				if n.mempool.HasPendingRequest() {
					if batch := n.mempool.GenerateBlock(); batch != nil {
						n.postProposal(batch)
						n.batchMgr.startTimer(Batch)

						// check if there is no tx in the mempool, start the no tx batch timer
						if n.isTimed && !n.mempool.HasPendingRequest() {
							if !n.batchMgr.isTimerActive(NoTxBatch) {
								n.batchMgr.startTimer(NoTxBatch)
							}
						}
					}
				} else {
					n.logger.Debug("The length of priorityIndex is 0, skip the batch timer")
				}
			case NoTxBatch:
				if !n.isTimed {
					n.logger.Warningf("The node is not support the no-tx batch, skip the timer")
					continue
				}
				if !n.mempool.HasPendingRequest() {
					n.batchMgr.stopTimer(NoTxBatch)
					n.logger.Debug("start create empty block")
					batch := n.mempool.GenerateBlock()
					n.postProposal(batch)
					if !n.batchMgr.isTimerActive(NoTxBatch) {
						n.batchMgr.startTimer(NoTxBatch)
					}
				}
			}
		}
	}
}

func (n *Node) postProposal(batch *mempool.RequestBatch) {
	n.recvCh <- batch
}
