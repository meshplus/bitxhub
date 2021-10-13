package router

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/cbergoon/merkletree"
	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

var _ Router = (*InterchainRouter)(nil)

const (
	blockChanNumber = 1024
)

type InterchainRouter struct {
	logger             logrus.FieldLogger
	repo               *repo.Repo
	piers              sync.Map
	unionPiers         sync.Map
	subscriptions      sync.Map
	unionSubscriptions sync.Map
	count              atomic.Int64
	ledger             *ledger.Ledger
	peerMgr            peermgr.PeerManager
	quorum             uint64

	ctx    context.Context
	cancel context.CancelFunc
}

func New(logger logrus.FieldLogger, repo *repo.Repo, ledger *ledger.Ledger, peerMgr peermgr.PeerManager, quorum uint64) (*InterchainRouter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	return &InterchainRouter{
		logger:  logger,
		ledger:  ledger,
		peerMgr: peerMgr,
		quorum:  quorum,
		repo:    repo,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (router *InterchainRouter) Start() error {
	router.logger.Infof("Router module started")
	return nil
}

func (router *InterchainRouter) Stop() error {
	router.cancel()

	router.logger.Infof("Router module stopped")

	return nil
}

func (router *InterchainRouter) AddPier(pierID string) (chan *pb.InterchainTxWrappers, error) {
	c := make(chan *pb.InterchainTxWrappers, blockChanNumber)
	raw, _ := router.piers.LoadOrStore(pierID, &event.Feed{})
	subBus := raw.(*event.Feed)
	sub := subBus.Subscribe(c)
	router.subscriptions.Store(pierID, sub)

	router.count.Inc()
	router.logger.WithFields(logrus.Fields{
		"pierID": pierID,
	}).Infof("Add pier")

	return c, nil
}

func (router *InterchainRouter) RemovePier(pierID string) {
	unsubscribeAndDel := func(r sync.Map) {
		raw, ok := r.Load(pierID)
		if !ok {
			return
		}
		sub := raw.(event.Subscription)
		sub.Unsubscribe()
		r.Delete(pierID)
	}
	unsubscribeAndDel(router.subscriptions)

	router.count.Dec()
}

func (router *InterchainRouter) PutBlockAndMeta(block *pb.Block, meta *pb.InterchainMeta) {
	if router.count.Load() == 0 {
		return
	}

	ret := router.classify(block, meta)
	router.piers.Range(func(k, value interface{}) bool {
		key := k.(string)
		w := value.(*event.Feed)
		wrappers := make([]*pb.InterchainTxWrapper, 0)
		_, ok := ret[key]
		if ok {
			wrappers = append(wrappers, ret[key])
			w.Send(&pb.InterchainTxWrappers{
				InterchainTxWrappers: wrappers,
			})
			return true
		}

		// empty interchain tx in this block
		emptyWrapper := &pb.InterchainTxWrapper{
			Height:  block.Height(),
			L2Roots: meta.L2Roots,
		}
		wrappers = append(wrappers, emptyWrapper)
		w.Send(&pb.InterchainTxWrappers{
			InterchainTxWrappers: wrappers,
		})

		return true
	})
}

func (router *InterchainRouter) GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error {
	defer close(ch)

	for i := begin; i <= end; i++ {
		block, err := router.ledger.GetBlock(i)
		if err != nil {
			return fmt.Errorf("get block: %w", err)
		}

		// TODO: fetch signatures to block header
		ch <- block.GetBlockHeader()
	}

	return nil
}

func (router *InterchainRouter) GetInterchainTxWrappers(appchainID string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error {
	defer close(ch)

	for i := begin; i <= end; i++ {
		block, err := router.ledger.GetBlock(i)
		if err != nil {
			return fmt.Errorf("get block: %w", err)
		}

		meta, err := router.ledger.GetInterchainMeta(i)
		if err != nil {
			return fmt.Errorf("get interchain meta data: %w", err)
		}

		ret := router.classify(block, meta)
		wrappers := make([]*pb.InterchainTxWrapper, 0)
		if ret[appchainID] != nil {
			wrappers = append(wrappers, ret[appchainID])
			ch <- &pb.InterchainTxWrappers{
				InterchainTxWrappers: wrappers,
			}
			continue
		} else {
			// empty interchain tx in this block
			emptyWrapper := &pb.InterchainTxWrapper{
				Height:  block.Height(),
				L2Roots: meta.L2Roots,
			}
			wrappers = append(wrappers, emptyWrapper)
			ch <- &pb.InterchainTxWrappers{
				InterchainTxWrappers: wrappers,
			}
			continue
		}

	}

	return nil
}

func (router *InterchainRouter) fetchSigns(height uint64) (map[string][]byte, error) {
	// TODO(xcc): fetch block sign from other nodes
	return nil, nil
}

func (router *InterchainRouter) classify(block *pb.Block, meta *pb.InterchainMeta) map[string]*pb.InterchainTxWrapper {
	txsM := make(map[string][]*pb.VerifiedTx)

	for k, vs := range meta.Counter {
		var txs []*pb.VerifiedTx
		for _, vi := range vs.Slice {
			tx, _ := block.Transactions.Transactions[vi.Index].(*pb.BxhTransaction)
			txs = append(txs, &pb.VerifiedTx{
				Tx:    tx,
				Valid: vi.Valid,
			})
		}
		txsM[k] = txs
	}

	target := make(map[string]*pb.InterchainTxWrapper)

	for dest, list := range meta.TimeoutCounter {
		wrapper := &pb.InterchainTxWrapper{
			Height:         block.BlockHeader.Number,
			L2Roots:        meta.L2Roots,
			TimeoutL2Roots: meta.TimeoutL2Roots,
			TimeoutIbtps:   list.GetSlice(),
		}
		target[dest] = wrapper
	}

	for dest, list := range meta.MultiTxCounter {
		if wrapper, has := target[dest]; has {
			wrapper.MultiTxIbtps = list.GetSlice()
			target[dest] = wrapper
		} else {
			wrapper := &pb.InterchainTxWrapper{
				Height:       block.BlockHeader.Number,
				L2Roots:      meta.L2Roots,
				MultiTxIbtps: list.GetSlice(),
			}
			target[dest] = wrapper
		}
	}

	for dest, txs := range txsM {
		if wrapper, has := target[dest]; has {
			wrapper.Transactions = txs
			target[dest] = wrapper
		} else {
			wrapper := &pb.InterchainTxWrapper{
				Height:       block.BlockHeader.Number,
				Transactions: txs,
				L2Roots:      meta.L2Roots,
			}
			target[dest] = wrapper
		}
	}

	return target
}

func (router *InterchainRouter) calcTimeoutL2Root(list []string) (types.Hash, error) {
	hashes := make([]merkletree.Content, 0, len(list))
	for _, id := range list {
		hash := sha256.Sum256([]byte(id))
		hashes = append(hashes, types.NewHash(hash[:]))
	}

	tree, err := merkletree.NewTree(hashes)
	if err != nil {
		return types.Hash{}, fmt.Errorf("init merkle tree: %w", err)
	}

	return *types.NewHash(tree.MerkleRoot()), nil
}
