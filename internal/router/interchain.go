package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/bitxhub/bitxid"
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

const blockChanNumber = 1024

type InterchainRouter struct {
	logger             logrus.FieldLogger
	repo               *repo.Repo
	piers              sync.Map
	unionPiers         sync.Map
	subscriptions      sync.Map
	unionSubscriptions sync.Map
	count              atomic.Int64
	ledger             ledger.Ledger
	peerMgr            peermgr.PeerManager
	quorum             uint64

	ctx    context.Context
	cancel context.CancelFunc
}

func New(logger logrus.FieldLogger, repo *repo.Repo, ledger ledger.Ledger, peerMgr peermgr.PeerManager, quorum uint64) (*InterchainRouter, error) {
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

func (router *InterchainRouter) AddPier(key bitxid.DID, pierID string, isUnion bool) (chan *pb.InterchainTxWrappers, error) {
	c := make(chan *pb.InterchainTxWrappers, blockChanNumber)
	if isUnion {
		raw, _ := router.unionPiers.LoadOrStore(string(key), event.Feed{})
		//if !ok {
		//	return nil, fmt.Errorf("did %s for subscription is not exist", key)
		//}
		subBus := raw.(event.Feed)
		sub := subBus.Subscribe(c)
		router.unionSubscriptions.Store(pierID, sub)
	} else {
		raw, _ := router.piers.LoadOrStore(string(key), event.Feed{})
		//if !ok {
		//	return nil, fmt.Errorf("did %s for subscription is not exist", key)
		//}
		subBus := raw.(event.Feed)
		sub := subBus.Subscribe(c)
		router.subscriptions.Store(pierID, sub)
	}

	router.count.Inc()
	router.logger.WithFields(logrus.Fields{
		"id":       key,
		"is_union": isUnion,
	}).Infof("Add pier")

	return c, nil
}

func (router *InterchainRouter) RemovePier(key bitxid.DID, pierID string, isUnion bool) {
	unsubscribeAndDel := func(r sync.Map) {
		raw, ok := r.Load(pierID)
		if !ok {
			return
		}
		sub := raw.(event.Subscription)
		sub.Unsubscribe()
		r.Delete(pierID)
	}
	if isUnion {
		unsubscribeAndDel(router.unionSubscriptions)
	} else {
		unsubscribeAndDel(router.subscriptions)
	}

	router.count.Dec()
}

func (router *InterchainRouter) PutBlockAndMeta(block *pb.Block, meta *pb.InterchainMeta) {
	if router.count.Load() == 0 {
		return
	}

	ret := router.classify(block, meta)
	router.piers.Range(func(k, value interface{}) bool {
		key := k.(string)
		w := value.(event.Feed)
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

	interchainTxWrappers := router.generateUnionInterchainTxWrappers(ret, block, meta)
	router.unionPiers.Range(func(k, v interface{}) bool {
		w := v.(event.Feed)
		w.Send(interchainTxWrappers)
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

func (router *InterchainRouter) GetInterchainTxWrappers(did string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error {
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
		if ret[did] != nil {
			wrappers = append(wrappers, ret[did])
			ch <- &pb.InterchainTxWrappers{
				InterchainTxWrappers: wrappers,
			}
			continue
		} else {
			_, ok := router.unionPiers.Load(did)
			if !ok {
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

			ch <- router.generateUnionInterchainTxWrappers(ret, block, meta)
		}

	}

	return nil
}

func (router *InterchainRouter) fetchSigns(height uint64) (map[string][]byte, error) {
	// TODO(xcc): fetch block sign from other nodes
	return nil, nil
}

func (router *InterchainRouter) classify(block *pb.Block, meta *pb.InterchainMeta) map[string]*pb.InterchainTxWrapper {
	txsM := make(map[string][]*pb.Transaction)
	hashesM := make(map[string][]types.Hash)

	for k, vs := range meta.Counter {
		var txs []*pb.Transaction
		var hashes []types.Hash
		for _, i := range vs.Slice {
			txs = append(txs, block.Transactions[i])
			hashes = append(hashes, *block.Transactions[i].TransactionHash)
		}
		// k value is the destination did address, so did with same method
		// will be grouped into one set
		fullMethod := bitxid.DID(k).GetMethod()
		txsM[fullMethod] = txs
		hashesM[fullMethod] = hashes
	}

	target := make(map[string]*pb.InterchainTxWrapper)
	for dest, txs := range txsM {
		wrapper := &pb.InterchainTxWrapper{
			Height:            block.BlockHeader.Number,
			TransactionHashes: hashesM[dest],
			Transactions:      txs,
			L2Roots:           meta.L2Roots,
		}
		target[dest] = wrapper
	}

	return target
}
