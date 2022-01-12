package solo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/order"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order/etcdraft"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	"github.com/meshplus/bitxhub/pkg/order/mempool"
	"github.com/sirupsen/logrus"
)

type Node struct {
	ID           uint64
	isTimed      bool
	commitC      chan *pb.CommitEvent         // block channel
	logger       logrus.FieldLogger           // logger
	mempool      mempool.MemPool              // transaction pool
	proposeC     chan *raftproto.RequestBatch // proposed listenReadyBlock, input channel
	stateC       chan *mempool.ChainState
	txCache      *mempool.TxCache // cache the transactions received from api
	batchMgr     *etcdraft.BatchTimer
	blockTimeout time.Duration                 // generate block period
	lastExec     uint64                        // the index of the last-applied block
	packSize     int                           // maximum number of transaction packages
	blockTick    time.Duration                 // block packed period
	peerMgr      orderPeerMgr.OrderPeerManager // network manager

	ctx    context.Context
	cancel context.CancelFunc
	sync.RWMutex
}

func (n *Node) GetPendingTxByHash(hash *types.Hash) pb.Transaction {
	return n.mempool.GetTransaction(hash)
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
	n.txCache.RecvTxC <- tx
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

func init() {
	agency.RegisterOrderConstructor("solo", NewNode)
}

func NewNode(opts ...order.Option) (order.Order, error) {
	var options []order.Option
	for i := range opts {
		options = append(options, opts[i])
	}

	config, err := order.GenerateConfig(options...)
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
	batchC := make(chan *raftproto.RequestBatch)
	mempoolInst, err := mempool.NewMempool(mempoolConf)
	if err != nil {
		return nil, fmt.Errorf("create mempool instance: %w", err)
	}
	txCache := mempool.NewTxCache(mempoolConf.TxSliceTimeout, mempoolConf.TxSliceSize, config.Logger)
	batchTimerMgr := etcdraft.NewTimer(batchTimeout, config.Logger)

	soloNode := &Node{
		ID:           config.ID,
		isTimed:      mempoolConf.IsTimed,
		blockTimeout: mempoolConf.BlockTimeout,
		commitC:      make(chan *pb.CommitEvent, 1024),
		stateC:       make(chan *mempool.ChainState),
		lastExec:     config.Applied,
		mempool:      mempoolInst,
		txCache:      txCache,
		batchMgr:     batchTimerMgr,
		peerMgr:      config.PeerMgr,
		proposeC:     batchC,
		logger:       config.Logger,
	}
	soloNode.logger.Infof("SOLO lastExec = %d", soloNode.lastExec)
	soloNode.logger.Infof("SOLO batch timeout = %v", batchTimeout)
	return soloNode, nil
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
			case proposal := <-n.proposeC:
				n.logger.WithFields(logrus.Fields{
					"proposal_height": proposal.Height,
					"tx_count":        len(proposal.TxList.Transactions),
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
						Timestamp: proposal.Timestamp,
					},
					Transactions: proposal.TxList,
				}
				localList := make([]bool, len(proposal.TxList.Transactions))
				for i := 0; i < len(proposal.TxList.Transactions); i++ {
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
			if !n.isTimed {
				// start batch timer when this node receives the first transaction
				if !n.batchMgr.IsBatchTimerActive() {
					n.batchMgr.StartBatchTimer()
				}
				if batch := n.mempool.ProcessTransactions(txSet.Transactions, true, true); batch != nil {
					n.batchMgr.StopBatchTimer()
					n.proposeC <- batch
				}
			} else {
				n.mempool.ProcessTransactions(txSet.Transactions, true, true)
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

func (n *Node) postProposal(batch *raftproto.RequestBatch) {
	n.proposeC <- batch
}
