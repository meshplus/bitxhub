package rbft

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
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
	"github.com/axiomesh/axiom/internal/order/common"
	"github.com/axiomesh/axiom/internal/order/precheck"
	"github.com/axiomesh/axiom/internal/order/rbft/adaptor"
	"github.com/axiomesh/axiom/internal/order/txcache"
	"github.com/axiomesh/axiom/internal/peermgr"
	"github.com/axiomesh/axiom/pkg/repo"
)

const (
	consensusMsgPipeIDPrefix = "consensus_msg_pipe_v1_"
	txsBroadcastMsgPipeID    = "txs_broadcast_msg_pipe_v1"
)

func init() {
	repo.Register(repo.OrderTypeRbft, true)
}

type Node struct {
	config              *common.Config
	n                   rbft.Node[types.Transaction, *types.Transaction]
	stack               *adaptor.RBFTAdaptor
	blockC              chan *common.CommitEvent
	logger              logrus.FieldLogger
	peerMgr             peermgr.PeerManager
	consensusMsgPipes   map[int32]network.Pipe
	txsBroadcastMsgPipe network.Pipe
	receiveMsgLimiter   *rate.Limiter

	ctx        context.Context
	cancel     context.CancelFunc
	txCache    *txcache.TxCache
	txPreCheck precheck.PreCheck

	txFeed event.Feed
}

func NewNode(config *common.Config) (*Node, error) {
	node, err := newNode(config)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func newNode(config *common.Config) (*Node, error) {
	rbftConfig, mempoolConfig := generateRbftConfig(config)
	blockC := make(chan *common.CommitEvent, 1024)

	ctx, cancel := context.WithCancel(context.Background())
	rbftAdaptor, err := adaptor.NewRBFTAdaptor(config, blockC, cancel)
	if err != nil {
		return nil, err
	}

	mp := mempool.NewMempool[types.Transaction, *types.Transaction](mempoolConfig)
	n, err := rbft.NewNode[types.Transaction, *types.Transaction](rbftConfig, rbftAdaptor, mp)
	if err != nil {
		return nil, err
	}

	var receiveMsgLimiter *rate.Limiter
	if config.Config.Limit.Enable {
		receiveMsgLimiter = rate.NewLimiter(rate.Limit(config.Config.Limit.Limit), int(config.Config.Limit.Burst))
	}
	return &Node{
		config:            config,
		n:                 n,
		logger:            config.Logger,
		stack:             rbftAdaptor,
		blockC:            blockC,
		receiveMsgLimiter: receiveMsgLimiter,
		ctx:               ctx,
		cancel:            cancel,
		txCache:           txcache.NewTxCache(rbftConfig.SetTimeout, uint64(rbftConfig.SetSize), config.Logger),
		peerMgr:           config.PeerMgr,
		txPreCheck:        precheck.NewTxPreCheckMgr(ctx, config.Logger, config.GetAccountBalance),
	}, nil
}

func (n *Node) initConsensusMsgPipes() error {
	n.consensusMsgPipes = make(map[int32]network.Pipe, len(consensus.Type_name))
	for id, name := range consensus.Type_name {
		msgPipe, err := n.peerMgr.CreatePipe(n.ctx, consensusMsgPipeIDPrefix+name)
		if err != nil {
			return err
		}
		n.consensusMsgPipes[id] = msgPipe
	}

	n.stack.SetMsgPipes(n.consensusMsgPipes)
	return nil
}

func (n *Node) Start() error {
	err := n.stack.UpdateEpoch()
	if err != nil {
		return err
	}

	n.n.ReportExecuted(&rbfttypes.ServiceState{
		MetaState: &rbfttypes.MetaState{
			Height: n.config.Applied,
			Digest: n.config.Digest,
		},
		Epoch: n.stack.EpochInfo.Epoch,
	})

	if err := n.initConsensusMsgPipes(); err != nil {
		return err
	}

	txsBroadcastMsgPipe, err := n.peerMgr.CreatePipe(n.ctx, txsBroadcastMsgPipeID)
	if err != nil {
		return err
	}
	n.txsBroadcastMsgPipe = txsBroadcastMsgPipe

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

	go n.txPreCheck.Start()
	go n.txCache.ListenEvent()

	go n.listenValidTxs()
	go n.listenNewTxToSubmit()
	go n.listenExecutedBlockToReport()
	go n.listenBatchMemTxsToBroadcast()
	go n.listenConsensusMsg()
	go n.listenTxsBroadcastMsg()

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

			if err := n.n.Propose(validTxs.Txs, validTxs.Local); err != nil {
				n.logger.WithField("err", err).Warn("Propose tx failed")
			}

			// post tx event to websocket
			go n.txFeed.Send(validTxs.Txs)

			// send successful response to api
			if validTxs.Local {
				validTxs.LocalRespCh <- &common.TxResp{Status: true}
			}
		}
	}
}

func (n *Node) listenConsensusMsg() {
	for _, pipe := range n.consensusMsgPipes {
		pipe := pipe
		go func() {
			for {
				msg := pipe.Receive(n.ctx)
				if msg == nil {
					return
				}

				if err := n.Step(msg.Data); err != nil {
					n.logger.WithField("err", err).Warn("Process order message failed")
					continue
				}
			}
		}()
	}
}

func (n *Node) listenTxsBroadcastMsg() {
	for {
		msg := n.txsBroadcastMsgPipe.Receive(n.ctx)
		if msg == nil {
			return
		}

		if n.receiveMsgLimiter != nil && !n.receiveMsgLimiter.Allow() {
			// rate limit exceeded, refuse to process the message
			n.logger.Warn("Node received too many PUSH_TXS messages. Rate limiting in effect")
			continue
		}

		tx := &pb.BytesSlice{}
		if err := tx.UnmarshalVT(msg.Data); err != nil {
			n.logger.WithField("err", err).Warn("Unmarshal txs message failed")
			continue
		}
		n.submitTxsFromRemote(tx.Slice)
	}
}

func (n *Node) listenNewTxToSubmit() {
	for {
		select {
		case <-n.ctx.Done():
			return

		case txWithResp := <-n.txCache.TxRespC:
			ev := &common.UncheckedTxEvent{
				EventType: common.LocalTxEvent,
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
					Epoch:           n.stack.EpochInfo.Epoch,
					Number:          r.Height,
					Timestamp:       r.Timestamp / int64(time.Second),
					ProposerAccount: r.ProposerAccount,
				},
				Transactions: r.Txs,
			}
			commitEvent := &common.CommitEvent{
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

				return n.txsBroadcastMsgPipe.Broadcast(context.TODO(), lo.Map(lo.Flatten([][]*rbft.NodeInfo{n.stack.EpochInfo.ValidatorSet, n.stack.EpochInfo.CandidateSet}), func(item *rbft.NodeInfo, index int) string {
					return item.P2PNodeID
				}), data)
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

	txWithResp := &common.TxWithResp{
		Tx:     tx,
		RespCh: make(chan *common.TxResp),
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

	ev := &common.UncheckedTxEvent{
		EventType: common.RemoteTxEvent,
		Event:     requests,
	}
	n.txPreCheck.PostUncheckedTxEvent(ev)
}

func (n *Node) Commit() chan *common.CommitEvent {
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
	return n.n.GetPendingTxByHash(hash.String())
}

func (n *Node) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {
	if n.stack.StateUpdating && n.stack.StateUpdateHeight != height {
		return
	}

	currentEpoch := n.stack.EpochInfo.Epoch

	// need update cached epoch info
	epochInfo := n.stack.EpochInfo
	if height == (epochInfo.StartBlock + epochInfo.EpochPeriod - 1) {
		err := n.stack.UpdateEpoch()
		if err != nil {
			panic(err)
		}
	}
	if n.stack.StateUpdating {
		state := &rbfttypes.ServiceState{
			MetaState: &rbfttypes.MetaState{
				Height: height,
				Digest: blockHash.String(),
			},
			Epoch: currentEpoch,
		}
		n.n.ReportStateUpdated(state)
		n.stack.StateUpdating = false
		return
	}

	state := &rbfttypes.ServiceState{
		MetaState: &rbfttypes.MetaState{
			Height: height,
			Digest: blockHash.String(),
		},
		Epoch: currentEpoch,
	}
	n.n.ReportExecuted(state)
}

func (n *Node) Quorum() uint64 {
	N := uint64(len(n.stack.EpochInfo.ValidatorSet))
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
