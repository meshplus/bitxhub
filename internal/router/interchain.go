package router

import (
	"context"
	"fmt"
	"sync"

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
	logger  logrus.FieldLogger
	repo    *repo.Repo
	piers   sync.Map
	count   atomic.Int64
	ledger  ledger.Ledger
	peerMgr peermgr.PeerManager
	quorum  uint64

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

func (router *InterchainRouter) AddPier(key string) (chan *pb.InterchainTxWrapper, error) {
	c := make(chan *pb.InterchainTxWrapper, blockChanNumber)
	router.piers.Store(key, c)
	router.count.Inc()
	router.logger.WithFields(logrus.Fields{
		"id": key,
	}).Infof("Add pier")

	return c, nil
}

func (router *InterchainRouter) RemovePier(key string) {
	router.piers.Delete(key)
	router.count.Dec()
}

func (router *InterchainRouter) PutBlockAndMeta(block *pb.Block, meta *pb.InterchainMeta) {
	if router.count.Load() == 0 {
		return
	}

	ret := router.classify(block, meta)

	router.piers.Range(func(k, value interface{}) bool {
		key := k.(string)
		w := value.(chan *pb.InterchainTxWrapper)
		_, ok := ret[key]
		if ok {
			w <- ret[key]
			return true
		}

		// empty interchain tx in this block
		w <- &pb.InterchainTxWrapper{
			Height:  block.Height(),
			L2Roots: meta.L2Roots,
		}

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

func (router *InterchainRouter) GetInterchainTxWrapper(pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrapper) error {
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
		if ret[pid] != nil {
			ch <- ret[pid]
			continue
		}

		// empty interchain tx in this block
		ch <- &pb.InterchainTxWrapper{
			Height:  block.Height(),
			L2Roots: meta.L2Roots,
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
			hashes = append(hashes, block.Transactions[i].TransactionHash)
		}
		txsM[k] = txs
		hashesM[k] = hashes
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
