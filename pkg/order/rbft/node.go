package rbft

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/event"
	"github.com/gogo/protobuf/proto"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/order"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ultramesh/rbft"
	"github.com/ultramesh/rbft/rbftpb"
)

type Node struct {
	id      uint64
	n       rbft.Node
	stack   *Stack
	blockC  chan *pb.CommitEvent
	logger  logrus.FieldLogger
	peerMgr orderPeerMgr.OrderPeerManager

	ctx     context.Context
	cancel  context.CancelFunc
	txCache *TxCache
}

func init() {
	agency.RegisterOrderConstructor("rbft", NewNode)
}

func NewNode(opts ...order.Option) (order.Order, error) {
	var options []order.Option
	for i, _ := range opts {
		options = append(options, opts[i])
	}

	config, err := order.GenerateConfig(options...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	store, err := NewStorage(config.StoragePath)
	if err != nil {
		return nil, err
	}
	rbftConfig, err := generateRbftConfig(config.RepoRoot, config)
	if err != nil {
		return nil, fmt.Errorf("generate rbft txpool config: %w", err)
	}
	blockC := make(chan *pb.CommitEvent, 1024)

	ctx, cancel := context.WithCancel(context.Background())
	s, err := NewStack(store, config, blockC, cancel, rbftConfig.IsNew)
	if err != nil {
		return nil, err
	}
	rbftConfig.External = s

	n, err := rbft.NewNode(rbftConfig)
	if err != nil {
		return nil, err
	}
	s.applyConfChange = n.ApplyConfChange

	n.ReportExecuted(&rbftpb.ServiceState{
		Applied: config.Applied,
		Digest:  config.Digest,
	})
	return &Node{
		id:      rbftConfig.ID,
		n:       n,
		logger:  config.Logger,
		stack:   s,
		blockC:  blockC,
		ctx:     ctx,
		cancel:  cancel,
		txCache: newTxCache(0, 0, config.Logger),
		peerMgr: config.PeerMgr,
	}, nil
}

func (n *Node) Start() error {
	if err := retry.Retry(func(attempt uint) error {
		err := n.checkQuorum()
		if err != nil {
			n.logger.Error(err)
			return err
		}
		return nil
	},
		strategy.Wait(1*time.Second),
	); err != nil {
		n.logger.Error(err)
	}

	go n.txCache.listenEvent()
	go func() {
		for {
			select {
			case r := <-n.stack.readyC:
				block := &pb.Block{
					BlockHeader: &pb.BlockHeader{
						Version:   []byte("1.0.0"),
						Number:    r.height,
						Timestamp: r.timestamp,
					},
					Transactions: &pb.Transactions{Transactions: r.txs},
				}
				commitEvent := &pb.CommitEvent{
					Block:     block,
					LocalList: r.localList,
				}
				n.blockC <- commitEvent

			case txSet := <-n.txCache.txSetC:
				_ = n.n.Propose(txSet)

			case txWithResp := <-n.txCache.TxRespC:
				_ = n.n.ProposeSync(txWithResp.Tx, txWithResp.Ch)

			case <-n.ctx.Done():
				n.n.Stop()
				return
			}
		}
	}()

	n.logger.Info("=====Order started=========")
	return n.n.Start()
}

func (n *Node) Stop() {
	n.cancel()
	if n.txCache.close != nil {
		close(n.txCache.close)
	}
}

func (n *Node) Prepare(tx pb.Transaction) error {
	if err := n.Ready(); err != nil {
		return err
	}
	if n.txCache.IsFull() && n.n.Status().Status == rbft.PoolFull {
		return errors.New("transaction cache are full, we will drop this transaction")
	}

	txWithResp := &TxWithResp{
		Tx: tx,
		Ch: make(chan bool),
	}
	n.txCache.TxRespC <- txWithResp
	n.txCache.recvTxC <- tx

	<-txWithResp.Ch
	return nil
}

func (n *Node) Commit() chan *pb.CommitEvent {
	return n.blockC
}

func (n *Node) Step(msg []byte) error {
	m := &rbftpb.ConsensusMessage{}
	if err := proto.Unmarshal(msg, m); err != nil {
		return err
	}

	n.n.Step(m)

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

func (n *Node) GetPendingTxByHash(hash *types.Hash) pb.Transaction {
	return n.n.GetPendingTxByHash(hash)
}

func (n *Node) DelNode(delID uint64) error {
	cc := &rbftpb.ConfChange{
		NodeID: delID,
		Type:   rbftpb.ConfChangeType_ConfChangeRemoveNode,
	}
	if err := n.n.ProposeConfChange(cc); err != nil {
		n.logger.Errorf("ProposeConfChange for delete vp failed, err: %s", err.Error())
		return err
	}
	return nil
}

func (n *Node) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {
	if n.stack.stateUpdating && n.stack.stateUpdateHeight != height {
		return
	}

	if n.stack.stateUpdating {
		state := &rbftpb.ServiceState{
			Applied: height,
			Digest:  blockHash.String(),
		}
		n.n.ReportStateUpdated(state)
		n.stack.stateUpdating = false
		return
	}

	if height%10 == 0 {
		n.logger.WithFields(logrus.Fields{
			"height": height,
		}).Info("Report checkpoint")
	}
	state := &rbftpb.ServiceState{
		Applied: height,
		Digest:  blockHash.String(),
	}
	n.n.ReportExecuted(state)
}

func (n *Node) Quorum() uint64 {
	N := uint64(len(n.stack.nodes))
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

func (n *Node) SubscribeTxEvent(events chan<- pb.Transactions) event.Subscription {
	return n.n.SubscribeTxEvent(events)
}

func readConfig(repoRoot string) (*RBFTConfig, error) {
	v := viper.New()
	v.SetConfigFile(filepath.Join(repoRoot, "order.toml"))
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &RBFTConfig{
		TimedGenBlock: defaultTimedConfig(),
	}
	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}

	if err := checkConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

// status2String returns a long description of SystemStatus
func status2String(status rbft.StatusType) string {
	switch status {
	case rbft.Normal:
		return "Normal"
	case rbft.InViewChange:
		return "system is in view change"
	case rbft.InRecovery:
		return "system is in recovery"
	case rbft.InUpdatingN:
		return "system is in updatingN"
	case rbft.PoolFull:
		return "system is too busy"
	case rbft.StateTransferring:
		return "system is in state update"
	case rbft.Pending:
		return "system is in pending state"
	default:
		return "Unknown status"
	}
}
