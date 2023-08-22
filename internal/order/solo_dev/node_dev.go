package solo_dev

import (
	"fmt"
	"sync"
	"time"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/pkg/repo"
	"github.com/ethereum/go-ethereum/event"
	"github.com/sirupsen/logrus"
)

var _ order.Order = (*NodeDev)(nil)

const checkpoint = 10

type GetAccountNonceFunc func(address *types.Address) uint64

func init() {
	repo.Register(repo.OrderTypeSoloDev, false)
}

type NodeDev struct {
	persistDoneC    chan struct{}           // signal of tx had been persisted
	commitC         chan *types.CommitEvent // block channel
	lastExec        uint64                  // the index of the last-applied block
	mutex           sync.Mutex
	logger          logrus.FieldLogger // logger
	GetAccountNonce GetAccountNonceFunc
	txFeed          event.Feed
}

func NewNode(opts ...order.Option) (order.Order, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	return &NodeDev{
		persistDoneC:    make(chan struct{}),
		commitC:         make(chan *types.CommitEvent),
		lastExec:        config.Applied,
		logger:          config.Logger,
		GetAccountNonce: config.GetAccountNonce,
	}, nil
}

func (n *NodeDev) Start() error {
	n.logger.Info("consensus dev started")
	return nil
}

func (n *NodeDev) Stop() {
	n.logger.Info("consensus dev stopped")
}

func (n *NodeDev) Prepare(tx *types.Transaction) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	block := &types.Block{
		BlockHeader: &types.BlockHeader{
			Version:   []byte("1.0.0"),
			Number:    n.lastExec + 1,
			Timestamp: time.Now().Unix(),
		},
		Transactions: []*types.Transaction{tx},
	}
	n.commitC <- &types.CommitEvent{
		Block:     block,
		LocalList: []bool{true},
	}
	n.lastExec++
	// ensure this tx had been persist
	<-n.persistDoneC
	return nil
}

func (n *NodeDev) SubmitTxsFromRemote(_ [][]byte) error {
	return nil
}

func (n *NodeDev) Commit() chan *types.CommitEvent {
	return n.commitC
}

func (n *NodeDev) Step(_ []byte) error {
	return nil
}

func (n *NodeDev) Ready() error {
	return nil
}

func (n *NodeDev) ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash) {
	if height%checkpoint == 0 {
		n.logger.WithFields(logrus.Fields{
			"height": height,
			"hash":   blockHash,
			"txs":    txHashList,
		}).Info("Report checkpoint")
	}
	n.logger.Debugf("ReportState", height, blockHash, txHashList)
	n.persistDoneC <- struct{}{}
}

func (n *NodeDev) Quorum() uint64 {
	return 1
}

func (n *NodeDev) GetPendingNonceByAccount(account string) uint64 {
	nonce := n.GetAccountNonce(types.NewAddressByStr(account))
	n.logger.Debugf("GetPendingNonceByAccount", nonce)
	return nonce
}

func (n *NodeDev) GetPendingTxByHash(_ *types.Hash) *types.Transaction {
	return nil
}

func (n *NodeDev) DelNode(_ uint64) error {
	return nil
}

func (n *NodeDev) SubscribeTxEvent(events chan<- []*types.Transaction) event.Subscription {
	return n.txFeed.Subscribe(events)
}
