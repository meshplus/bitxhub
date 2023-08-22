package rbft

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/axiomesh/axiom/internal/order/precheck"
	"github.com/ethereum/go-ethereum/event"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-bft/common/consensus"
	"github.com/axiomesh/axiom-bft/mempool"
	rbfttypes "github.com/axiomesh/axiom-bft/types"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
	network "github.com/axiomesh/axiom-p2p"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/order/rbft/adaptor"
	"github.com/axiomesh/axiom/internal/order/txcache"
	"github.com/axiomesh/axiom/internal/peermgr"
)

const (
	pipeID = "order"
)

type Node struct {
	id                uint64
	n                 rbft.Node[types.Transaction, *types.Transaction]
	memPool           mempool.MemPool[types.Transaction, *types.Transaction]
	stack             *adaptor.RBFTAdaptor
	blockC            chan *types.CommitEvent
	logger            logrus.FieldLogger
	peerMgr           peermgr.PeerManager
	msgPipe           network.Pipe
	receiveMsgLimiter *rate.Limiter

	checkpoint uint64

	ctx        context.Context
	cancel     context.CancelFunc
	txCache    *txcache.TxCache
	txPreCheck precheck.PreCheck

	txFeed event.Feed
}

func NewNode(opts ...order.Option) (order.Order, error) {
	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func newNode(opts ...order.Option) (*Node, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	rbftConfig, mempoolConfig, err := generateRbftConfig(config)
	if err != nil {
		return nil, fmt.Errorf("generate rbft txpool config: %w", err)
	}
	blockC := make(chan *types.CommitEvent, 1024)

	ctx, cancel := context.WithCancel(context.Background())
	rbftAdaptor, err := adaptor.NewRBFTAdaptor(config, blockC, cancel)
	if err != nil {
		return nil, err
	}
	rbftConfig.External = rbftAdaptor

	rbftConfig.RequestPool = mempool.NewMempool[types.Transaction, *types.Transaction](mempoolConfig)

	n, err := rbft.NewNode(rbftConfig)
	if err != nil {
		return nil, err
	}
	rbftAdaptor.SetApplyConfChange(n.ApplyConfChange)

	n.ReportExecuted(&rbfttypes.ServiceState{
		MetaState: &rbfttypes.MetaState{
			Height: config.Applied,
			Digest: config.Digest,
		},
		// TODO: should read from ledger
		Epoch: rbftConfig.EpochInit,
	})

	var receiveMsgLimiter *rate.Limiter
	if config.Config.Limit.Enable {
		receiveMsgLimiter = rate.NewLimiter(rate.Limit(config.Config.Limit.Limit), int(config.Config.Limit.Burst))
	}
	return &Node{
		id:                rbftConfig.ID,
		n:                 n,
		memPool:           rbftConfig.RequestPool,
		logger:            config.Logger,
		stack:             rbftAdaptor,
		blockC:            blockC,
		receiveMsgLimiter: receiveMsgLimiter,
		ctx:               ctx,
		cancel:            cancel,
		txCache:           txcache.NewTxCache(rbftConfig.SetTimeout, uint64(rbftConfig.SetSize), config.Logger),
		txPreCheck:        precheck.NewTxPreCheckMgr(ctx, config.Logger, config.GetAccountBalance),

		peerMgr:    config.PeerMgr,
		checkpoint: config.Config.Rbft.CheckpointPeriod,
	}, nil
}

func (n *Node) Start() error {
	pipe, err := n.peerMgr.CreatePipe(n.ctx, pipeID)
	if err != nil {
		return err
	}
	n.msgPipe = pipe
	n.stack.SetMsgPipe(pipe)

	if err := retry.Retry(func(attempt uint) error {
		err := n.checkQuorum()
		if err != nil {
			return err
		}
		return nil
	},
		strategy.Wait(1*time.Second),
	); err != nil {
		n.logger.Error(err)
	}

	n.txPreCheck.Start()
	go n.txCache.ListenEvent()

	go n.listenValidTxs()
	go n.listenNewTxToSubmit()
	go n.listenExecutedBlockToReport()
	go n.listenBatchMemTxsToBroadcast()
	go n.listenP2PMsg()

	n.logger.Info("=====Order started=========")
	return n.n.Start()
}

func (n *Node) listenValidTxs() {
	for {
		select {
		case <-n.ctx.Done():
			return
		case validTxs := <-n.txPreCheck.CommitValidTxs():
			var requests [][]byte
			for _, tx := range validTxs.Txs {
				raw, err := tx.RbftMarshal()
				if err != nil {
					n.logger.Error(err)
					continue
				}
				requests = append(requests, raw)
			}

			if err := n.n.Propose(&consensus.RequestSet{
				Requests: requests,
				Local:    validTxs.Local,
			}); err != nil {
				n.logger.WithField("err", err).Warn("Propose tx failed")
			}

			// post tx event to websocket
			go n.txFeed.Send(validTxs.Txs)

			// send successful response to api
			if validTxs.Local {
				validTxs.LocalRespCh <- &order.TxResp{Status: true}
			}
		}
	}
}

func (n *Node) listenP2PMsg() {
	for {
		msg := n.msgPipe.Receive(n.ctx)
		if msg == nil {
			return
		}

		m := &pb.Message{}
		if err := m.UnmarshalVT(msg.Data); err != nil {
			n.logger.WithField("err", err).Warn("Unmarshal order message failed")
			continue
		}
		switch m.Type {
		case pb.Message_CONSENSUS:
			if err := n.Step(m.Data); err != nil {
				n.logger.WithField("err", err).Warn("Process order message failed")
				continue
			}

		case pb.Message_PUSH_TXS:
			if n.receiveMsgLimiter != nil && !n.receiveMsgLimiter.Allow() {
				// rate limit exceeded, refuse to process the message
				n.logger.Warn("Node received too many PUSH_TXS messages. Rate limiting in effect")
				continue
			}

			tx := &pb.BytesSlice{}
			if err := tx.UnmarshalVT(m.Data); err != nil {
				n.logger.WithField("err", err).Warn("Unmarshal txs message failed")
				continue
			}
			n.submitTxsFromRemote(tx.Slice)
		}
	}
}

func (n *Node) listenNewTxToSubmit() {
	for {
		select {
		case <-n.ctx.Done():
			return

		case txWithResp := <-n.txCache.TxRespC:
			ev := &order.UncheckedTxEvent{
				EventType: order.LocalTxEvent,
				Event:     txWithResp,
			}
			n.txPreCheck.PostUncheckedTxEvent(ev)
		}
	}
}

func (n *Node) listenExecutedBlockToReport() {
	for {
		select {
		case r := <-n.stack.ReadyC:
			block := &types.Block{
				BlockHeader: &types.BlockHeader{
					Version:   []byte("1.0.0"),
					Number:    r.Height,
					Timestamp: r.Timestamp,
				},
				Transactions: r.TXs,
			}
			commitEvent := &types.CommitEvent{
				Block:     block,
				LocalList: r.LocalList,
			}
			n.blockC <- commitEvent
		case <-n.ctx.Done():
			return
		}
	}
}

func (n *Node) listenBatchMemTxsToBroadcast() {
	for {
		select {
		case txSet := <-n.txCache.TxSetC:
			var requests [][]byte
			for _, tx := range txSet {
				raw, err := tx.RbftMarshal()
				if err != nil {
					n.logger.Error(err)
					continue
				}
				requests = append(requests, raw)
			}

			// broadcast to other node
			err := func() error {
				msg := &pb.BytesSlice{
					Slice: requests,
				}
				data, err := msg.MarshalVT()
				if err != nil {
					return err
				}

				p2pmsg := &pb.Message{
					Type:    pb.Message_PUSH_TXS,
					Data:    data,
					Version: []byte("0.1.0"),
				}

				msgData, err := p2pmsg.MarshalVT()
				if err != nil {
					return err
				}

				return n.msgPipe.Broadcast(context.TODO(), lo.MapToSlice(n.peerMgr.OrderPeers(), func(k uint64, v *types.VpInfo) string {
					return v.Pid
				}), msgData)
			}()
			if err != nil {
				n.logger.Errorf("failed to broadcast mempool txs: %v", err)
			}

		case <-n.ctx.Done():
			return
		}
	}
}

func (n *Node) Stop() {
	n.cancel()
	if n.txCache.CloseC != nil {
		close(n.txCache.CloseC)
	}
	n.n.Stop()
}

func (n *Node) Prepare(tx *types.Transaction) error {
	if err := n.Ready(); err != nil {
		return err
	}
	if n.txCache.IsFull() && n.n.Status().Status == rbft.PoolFull {
		return errors.New("transaction cache are full, we will drop this transaction")
	}

	txWithResp := &order.TxWithResp{
		Tx:     tx,
		RespCh: make(chan *order.TxResp),
	}
	n.txCache.TxRespC <- txWithResp
	n.txCache.RecvTxC <- tx

	resp := <-txWithResp.RespCh
	if !resp.Status {
		return fmt.Errorf(resp.ErrorMsg)
	}
	return nil
}

func (n *Node) submitTxsFromRemote(txs [][]byte) {
	var requests []*types.Transaction
	for _, item := range txs {
		tx := &types.Transaction{}
		if err := tx.RbftUnmarshal(item); err != nil {
			n.logger.Error(err)
			continue
		}
		requests = append(requests, tx)
	}

	ev := &order.UncheckedTxEvent{
		EventType: order.RemoteTxEvent,
		Event:     requests,
	}
	n.txPreCheck.PostUncheckedTxEvent(ev)
}

func (n *Node) Commit() chan *types.CommitEvent {
	return n.blockC
}

func (n *Node) Step(msg []byte) error {
	m := &consensus.ConsensusMessage{}
	if err := m.Unmarshal(msg); err != nil {
		return err
	}
	n.n.Step(context.Background(), m)

	return nil
}

func (n *Node) Ready() error {
	status := n.n.Status().Status
	isNormal := status == rbft.Normal
	if !isNormal {
		return fmt.Errorf("%s", status2String(status))
	}
	return nil
}

func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	return n.n.GetPendingNonceByAccount(account)
}

func (n *Node) GetPendingTxByHash(hash *types.Hash) *types.Transaction {
	txData := n.n.GetPendingTxByHash(hash.String())
	if txData == nil {
		return nil
	}
	tx := &types.Transaction{}
	err := tx.RbftUnmarshal(txData)
	if err != nil {
		n.logger.Errorf("GetPendingTxByHash unmarshall err: %s", err)
	}
	return tx
}

func (n *Node) DelNode(_ uint64) error {
	return errors.New("unsupported api")
}

func (n *Node) ReportState(height uint64, blockHash *types.Hash, _ []*types.Hash) {
	if n.stack.StateUpdating && n.stack.StateUpdateHeight != height {
		return
	}

	if n.stack.StateUpdating {
		state := &rbfttypes.ServiceState{
			MetaState: &rbfttypes.MetaState{
				Height: height,
				Digest: blockHash.String(),
			},
			Epoch: 1,
		}
		n.n.ReportStateUpdated(state)
		n.stack.StateUpdating = false
		return
	}

	// TODO: read from cfg
	if height%n.checkpoint == 0 {
		n.logger.WithFields(logrus.Fields{
			"height": height,
		}).Info("Report checkpoint")
		n.n.ReportStableCheckpointFinished(height)
	}
	state := &rbfttypes.ServiceState{
		MetaState: &rbfttypes.MetaState{
			Height: height,
			Digest: blockHash.String(),
		},
		Epoch: 1,
	}
	n.n.ReportExecuted(state)
}

func (n *Node) Quorum() uint64 {
	N := uint64(len(n.stack.Nodes))
	f := (N - 1) / 3
	return (N + f + 2) / 2
}

func (n *Node) checkQuorum() error {
	n.logger.Infof("=======Quorum = %d, connected peers = %d", n.Quorum(), n.peerMgr.CountConnectedPeers()+1)
	if n.peerMgr.CountConnectedPeers()+1 < n.Quorum() {
		return errors.New("the number of connected Peers don't reach Quorum")
	}
	return nil
}

func (n *Node) SubscribeTxEvent(events chan<- []*types.Transaction) event.Subscription {
	return n.txFeed.Subscribe(events)
}

// status2String returns a long description of SystemStatus
func status2String(status rbft.StatusType) string {
	switch status {
	case rbft.Normal:
		return "Normal"
	case rbft.InConfChange:
		return "system is in conf change"
	case rbft.InViewChange:
		return "system is in view change"
	case rbft.InRecovery:
		return "system is in recovery"
	case rbft.StateTransferring:
		return "system is in state update"
	case rbft.PoolFull:
		return "system is too busy"
	case rbft.Pending:
		return "system is in pending state"
	case rbft.Stopped:
		return "system is stopped"
	default:
		return fmt.Sprintf("Unknown status: %d", status)
	}
}
