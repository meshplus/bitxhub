package router

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

var _ Router = (*InterchainRouter)(nil)

const blockChanNumber = 1024

type InterchainRouter struct {
	logger  logrus.FieldLogger
	repo    *repo.Repo
	piers   sync.Map
	count   uint64
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
	router.logger.Infof("router module started")

	return nil
}

func (router *InterchainRouter) Stop() error {
	router.cancel()

	router.logger.Infof("router module stopped")

	return nil
}

func (router *InterchainRouter) AddPier(key string) (chan *pb.MerkleWrapper, error) {
	c := make(chan *pb.MerkleWrapper, blockChanNumber)
	router.piers.Store(key, c)
	router.count++
	router.logger.WithFields(logrus.Fields{
		"id": key,
	}).Infof("Add pier")

	return c, nil
}

func (router *InterchainRouter) RemovePier(key string) {
	router.piers.Delete(key)
	router.count--
}

func (router *InterchainRouter) PutBlock(block *pb.Block) {
	if router.count == 0 {
		return
	}

	signed, err := router.fetchSigns(block.BlockHeader.Number)
	if err != nil {
		router.logger.Errorf("fetch signs: %w", err)
	}

	ret := router.classify(block)

	router.piers.Range(func(k, value interface{}) bool {
		key := k.(string)
		w := value.(chan *pb.MerkleWrapper)
		_, ok := ret[key]
		if ok {
			ret[key].Signatures = signed
			w <- ret[key]
			return true
		}

		w <- &pb.MerkleWrapper{
			BlockHeader: block.BlockHeader,
			BlockHash:   block.BlockHash,
			Signatures:  signed,
		}

		return true
	})
}

func (router *InterchainRouter) GetMerkleWrapper(pid string, begin, end uint64, ch chan<- *pb.MerkleWrapper) error {
	for i := begin; i <= end; i++ {
		block, err := router.ledger.GetBlock(i)
		if err != nil {
			return fmt.Errorf("get block: %w", err)
		}

		signed, err := router.fetchSigns(i)
		if err != nil {
			return fmt.Errorf("fetch signs: %w", err)
		}

		ret := router.classify(block)
		if ret[pid] != nil {
			ret[pid].Signatures = signed
			ch <- ret[pid]
			continue
		}

		ch <- &pb.MerkleWrapper{
			BlockHeader: block.BlockHeader,
			BlockHash:   block.BlockHash,
			Signatures:  signed,
		}
	}

	return nil
}

func (router *InterchainRouter) fetchSigns(height uint64) (map[string][]byte, error) {
	// TODO(xcc): fetch block sign from other nodes
	return nil, nil
}

func (router *InterchainRouter) classify(block *pb.Block) map[string]*pb.MerkleWrapper {
	hashes := make([]types.Hash, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		hashes = append(hashes, tx.TransactionHash)
	}

	if block.BlockHeader.InterchainIndex == nil {
		return make(map[string]*pb.MerkleWrapper)
	}
	idx := make(map[string][]uint64)
	m := make(map[string][]*pb.Transaction)
	err := json.Unmarshal(block.BlockHeader.InterchainIndex, &idx)
	if err != nil {
		panic(err)
	}

	for k, vs := range idx {
		var txs []*pb.Transaction
		for _, i := range vs {
			txs = append(txs, block.Transactions[i])
		}
		m[k] = txs
	}

	target := make(map[string]*pb.MerkleWrapper)
	for dest, txs := range m {
		wrapper := &pb.MerkleWrapper{
			BlockHeader:       block.BlockHeader,
			TransactionHashes: hashes,
			Transactions:      txs,
			BlockHash:         block.BlockHash,
		}
		target[dest] = wrapper
	}

	return target
}
