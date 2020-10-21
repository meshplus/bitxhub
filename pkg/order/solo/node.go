package solo

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/sirupsen/logrus"
)

type Node struct {
	sync.RWMutex
	height             uint64             // current block height
	pendingTxs         *list.List         // pending tx pool
	commitC            chan *pb.Block     // block channel
	logger             logrus.FieldLogger // logger
	reqLookUp          *order.ReqLookUp   // bloom filter
	getTransactionFunc func(hash types.Hash) (*pb.Transaction, error)

	packSize  int           // maximum number of transaction packages
	blockTick time.Duration // block packed period

	ctx    context.Context
	cancel context.CancelFunc
}

func (n *Node) Start() error {
	go n.execute()
	return nil
}

func (n *Node) Stop() {
	n.cancel()
}

func (n *Node) GetPendingNonceByAccount(account string) uint64 {
	// TODO: implement me
	return 0
}

func (n *Node) Prepare(tx *pb.Transaction) error {
	hash := tx.TransactionHash
	if ok := n.reqLookUp.LookUp(hash.Bytes()); ok {
		if tx, _ := n.getTransactionFunc(*hash); tx != nil {
			return nil
		}
	}
	n.pushBack(tx)
	if n.PoolSize() >= n.packSize {
		if r := n.ready(); r != nil {
			n.commitC <- r
		}
	}
	return nil
}

//Current txpool's size
func (n *Node) PoolSize() int {
	n.RLock()
	defer n.RUnlock()
	return n.pendingTxs.Len()
}

func (n *Node) Commit() chan *pb.Block {
	return n.commitC
}

func (n *Node) Step(ctx context.Context, msg []byte) error {
	return nil
}

func (n *Node) Ready() bool {
	return true
}

func (n *Node) ReportState(height uint64, hash types.Hash) {
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
	return &Node{
		height:             config.Applied,
		pendingTxs:         list.New(),
		commitC:            make(chan *pb.Block, 1024),
		packSize:           500,
		blockTick:          500 * time.Millisecond,
		reqLookUp:          reqLookUp,
		getTransactionFunc: config.GetTransactionFunc,
		logger:             config.Logger,
		ctx:                ctx,
		cancel:             cancel,
	}, nil
}

// Schedule to collect txs to the ready channel
func (n *Node) execute() {
	ticker := time.NewTicker(n.blockTick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if r := n.ready(); r != nil {
				n.commitC <- r
			}
		case <-n.ctx.Done():
			n.logger.Infoln("Done txpool execute")
			return
		}
	}

}

func (n *Node) ready() *pb.Block {
	n.Lock()
	defer n.Unlock()
	l := n.pendingTxs.Len()
	if l == 0 {
		return nil
	}
	var size int
	if l > n.packSize {
		size = n.packSize
	} else {
		size = l
	}
	txs := make([]*pb.Transaction, 0, size)
	for i := 0; i < size; i++ {
		front := n.pendingTxs.Front()
		tx := front.Value.(*pb.Transaction)
		txs = append(txs, tx)
		n.pendingTxs.Remove(front)
	}
	n.height++

	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Version:   []byte("1.0.0"),
			Number:    n.height,
			Timestamp: time.Now().UnixNano(),
		},
		Transactions: txs,
	}
	return block
}

func (n *Node) pushBack(value interface{}) *list.Element {
	n.Lock()
	defer n.Unlock()
	return n.pendingTxs.PushBack(value)
}
