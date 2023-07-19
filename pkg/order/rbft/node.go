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
	rbft "github.com/hyperchain/go-hpc-rbft/v2"
	"github.com/hyperchain/go-hpc-rbft/v2/common/consensus"
	"github.com/hyperchain/go-hpc-rbft/v2/txpool"
	rbfttypes "github.com/hyperchain/go-hpc-rbft/v2/types"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/rbft/adaptor"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	ethtypes "github.com/meshplus/eth-kit/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Node struct {
	id      uint64
	n       rbft.Node[ethtypes.EthTransaction, *ethtypes.EthTransaction]
	txPool  txpool.TxPool[ethtypes.EthTransaction, *ethtypes.EthTransaction]
	stack   *adaptor.RBFTAdaptor
	blockC  chan *pb.CommitEvent
	logger  logrus.FieldLogger
	peerMgr peermgr.OrderPeerManager

	ctx     context.Context
	cancel  context.CancelFunc
	txCache *TxCache

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

	rbftConfig, txpoolConfig, err := generateRbftConfig(config.RepoRoot, config)
	if err != nil {
		return nil, fmt.Errorf("generate rbft txpool config: %w", err)
	}
	blockC := make(chan *pb.CommitEvent, 1024)

	ctx, cancel := context.WithCancel(context.Background())
	rbftAdaptor, err := adaptor.NewRBFTAdaptor(config, blockC, cancel, rbftConfig.IsNew)
	if err != nil {
		return nil, err
	}
	rbftConfig.External = rbftAdaptor

	rbftTXPoolAdaptor, err := adaptor.NewRBFTTXPoolAdaptor()
	if err != nil {
		return nil, err
	}
	rbftConfig.RequestPool = txpool.NewTxPool[ethtypes.EthTransaction, *ethtypes.EthTransaction]("global", rbftTXPoolAdaptor, txpoolConfig)

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
	return &Node{
		id:      rbftConfig.ID,
		n:       n,
		txPool:  rbftConfig.RequestPool,
		logger:  config.Logger,
		stack:   rbftAdaptor,
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
			case r := <-n.stack.ReadyC:
				block := &pb.Block{
					BlockHeader: &pb.BlockHeader{
						Version:   []byte("1.0.0"),
						Number:    r.Height,
						Timestamp: r.Timestamp,
					},
					Transactions: &pb.Transactions{Transactions: r.TXs},
				}
				commitEvent := &pb.CommitEvent{
					Block:     block,
					LocalList: r.LocalList,
				}
				n.blockC <- commitEvent

			case txSet := <-n.txCache.txSetC:
				var requests [][]byte
				for _, item := range txSet {
					if tx, ok := item.(*ethtypes.EthTransaction); ok {
						raw, err := tx.RbftMarshal()
						if err != nil {
							n.logger.Error(err)
							continue
						}
						requests = append(requests, raw)
					}
				}

				// broadcast to other node
				err := func() error {
					msg := &pb.PushTxs{
						Data: requests,
					}
					data, err := msg.Marshal()
					if err != nil {
						return err
					}

					p2pmsg := &pb.Message{
						Type:    pb.Message_PUSH_TXS,
						Data:    data,
						Version: []byte("0.1.0"),
					}

					return n.peerMgr.Broadcast(p2pmsg)
				}()
				if err != nil {
					n.logger.Errorf("failed to broadcast mempool txs: %v", err)
				}

			case txWithResp := <-n.txCache.TxRespC:
				var requests [][]byte
				if tx, ok := txWithResp.Tx.(*ethtypes.EthTransaction); ok {
					raw, err := tx.RbftMarshal()
					if err != nil {
						n.logger.Error(err)
					} else {
						requests = append(requests, raw)
					}
				}

				if len(requests) != 0 {
					_ = n.n.Propose(&consensus.RequestSet{
						Requests: requests,
						Local:    true,
					})
					go n.txFeed.Send(pb.Transactions{Transactions: []pb.Transaction{txWithResp.Tx}})
				}

				txWithResp.Ch <- true

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

func (n *Node) SubmitTxsFromRemote(tsx [][]byte) error {
	var requests []pb.Transaction
	for _, item := range tsx {
		tx := &ethtypes.EthTransaction{}
		if err := tx.RbftUnmarshal(item); err != nil {
			n.logger.Error(err)
			continue
		}
		requests = append(requests, tx)
	}
	go n.txFeed.Send(pb.Transactions{Transactions: requests})

	return n.n.Propose(&consensus.RequestSet{
		Requests: tsx,
		Local:    true,
	})
}

func (n *Node) Commit() chan *pb.CommitEvent {
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

// TODO: implement it
func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	return 0
}

// TODO: implement it
func (n *Node) GetPendingTxByHash(hash *types.Hash) pb.Transaction {
	return nil
}

func (n *Node) DelNode(delID uint64) error {
	return errors.New("unsupported api")
}

func (n *Node) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {
	if n.stack.StateUpdating && n.stack.StateUpdateHeight != height {
		return
	}

	if n.stack.StateUpdating {
		state := &rbfttypes.ServiceState{
			MetaState: &rbfttypes.MetaState{
				Height: height,
				Digest: blockHash.String(),
			},
			Epoch: 0,
		}
		n.n.ReportStateUpdated(state)
		n.stack.StateUpdating = false
		return
	}

	if height%10 == 0 {
		n.logger.WithFields(logrus.Fields{
			"height": height,
		}).Info("Report checkpoint")
	}
	state := &rbfttypes.ServiceState{
		MetaState: &rbfttypes.MetaState{
			Height: height,
			Digest: blockHash.String(),
		},
		Epoch: 0,
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

func (n *Node) SubscribeTxEvent(events chan<- pb.Transactions) event.Subscription {
	return n.txFeed.Subscribe(events)
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
