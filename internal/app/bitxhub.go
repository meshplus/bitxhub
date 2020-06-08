package app

import (
	"context"
	"fmt"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/genesis"
	"github.com/meshplus/bitxhub/internal/loggers"
	orderplg "github.com/meshplus/bitxhub/internal/plugins"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/internal/router"
	"github.com/meshplus/bitxhub/internal/storages"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/order/etcdraft"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

type BitXHub struct {
	Ledger   ledger.Ledger
	Executor executor.Executor
	Router   router.Router
	Order    order.Order
	PeerMgr  peermgr.PeerManager

	repo   *repo.Repo
	logger logrus.FieldLogger

	ctx    context.Context
	cancel context.CancelFunc
}

func NewBitXHub(rep *repo.Repo) (*BitXHub, error) {
	repoRoot := rep.Config.RepoRoot

	bxh, err := generateBitXHubWithoutOrder(rep)
	if err != nil {
		return nil, err
	}

	chainMeta := bxh.Ledger.GetChainMeta()

	m := make(map[uint64]types.Address)

	if !rep.Config.Solo {
		for i, node := range rep.NetworkConfig.Nodes {
			m[node.ID] = types.String2Address(rep.Genesis.Addresses[i])
		}
	}

	order, err := orderplg.New(
		order.WithRepoRoot(repoRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithPluginPath(rep.Config.Plugin),
		order.WithNodes(m),
		order.WithID(rep.NetworkConfig.ID),
		order.WithPeerManager(bxh.PeerMgr),
		order.WithLogger(loggers.Logger(loggers.Order)),
		order.WithApplied(chainMeta.Height),
		order.WithDigest(chainMeta.BlockHash.Hex()),
		order.WithGetChainMetaFunc(bxh.Ledger.GetChainMeta),
		order.WithGetTransactionFunc(bxh.Ledger.GetTransaction),
	)
	if err != nil {
		return nil, err
	}

	r, err := router.New(loggers.Logger(loggers.Router), rep, bxh.Ledger, bxh.PeerMgr, order.Quorum())
	if err != nil {
		return nil, fmt.Errorf("create InterchainRouter: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bxh.ctx = ctx
	bxh.cancel = cancel
	bxh.Order = order
	bxh.Router = r

	return bxh, nil
}

func generateBitXHubWithoutOrder(rep *repo.Repo) (*BitXHub, error) {
	repoRoot := rep.Config.RepoRoot
	logger := loggers.Logger(loggers.App)

	if err := storages.Initialize(repoRoot); err != nil {
		return nil, fmt.Errorf("storages initialize: %w", err)
	}

	bcStorage, err := storages.Get(storages.BlockChain)
	if err != nil {
		return nil, fmt.Errorf("create blockchain storage: %w", err)
	}

	// 0. load ledger
	ldg, err := ledger.New(repoRoot, bcStorage, loggers.Logger(loggers.Executor))
	if err != nil {
		return nil, fmt.Errorf("create ledger: %w", err)
	}

	if ldg.GetChainMeta().Height == 0 {
		if err := genesis.Initialize(rep.Genesis, ldg); err != nil {
			return nil, err
		}
		logger.Info("Initialize genesis")
	}

	// 1. create executor
	exec, err := executor.New(ldg, loggers.Logger(loggers.Executor))
	if err != nil {
		return nil, fmt.Errorf("create BlockExecutor: %w", err)
	}

	peerMgr := &peermgr.Swarm{}
	if !rep.Config.Solo {
		peerMgr, err = peermgr.New(rep, loggers.Logger(loggers.P2P), ldg)
		if err != nil {
			return nil, fmt.Errorf("create peer manager: %w", err)
		}
	}

	return &BitXHub{
		repo:     rep,
		logger:   logger,
		Ledger:   ldg,
		Executor: exec,
		PeerMgr:  peerMgr,
	}, nil
}

func NewTesterBitXHub(rep *repo.Repo) (*BitXHub, error) {
	repoRoot := rep.Config.RepoRoot

	bxh, err := generateBitXHubWithoutOrder(rep)
	if err != nil {
		return nil, err
	}

	chainMeta := bxh.Ledger.GetChainMeta()

	m := make(map[uint64]types.Address)

	if !rep.Config.Solo {
		for i, node := range rep.NetworkConfig.Nodes {
			m[node.ID] = types.String2Address(rep.Genesis.Addresses[i])
		}
	}

	order, err := etcdraft.NewNode(
		order.WithRepoRoot(repoRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithPluginPath(rep.Config.Plugin),
		order.WithNodes(m),
		order.WithID(rep.NetworkConfig.ID),
		order.WithPeerManager(bxh.PeerMgr),
		order.WithLogger(loggers.Logger(loggers.Order)),
		order.WithApplied(chainMeta.Height),
		order.WithDigest(chainMeta.BlockHash.Hex()),
		order.WithGetChainMetaFunc(bxh.Ledger.GetChainMeta),
		order.WithGetTransactionFunc(bxh.Ledger.GetTransaction),
	)

	if err != nil {
		return nil, err
	}

	r, err := router.New(loggers.Logger(loggers.Router), rep, bxh.Ledger, bxh.PeerMgr, order.Quorum())
	if err != nil {
		return nil, fmt.Errorf("create InterchainRouter: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bxh.ctx = ctx
	bxh.cancel = cancel
	bxh.Order = order
	bxh.Router = r

	return bxh, nil
}

func (bxh *BitXHub) Start() error {
	if !bxh.repo.Config.Solo {
		if err := bxh.PeerMgr.Start(); err != nil {
			return fmt.Errorf("peer manager start: %w", err)
		}
	}

	if err := bxh.Order.Start(); err != nil {
		return fmt.Errorf("order start: %w", err)
	}

	if err := bxh.Executor.Start(); err != nil {
		return fmt.Errorf("executor start: %w", err)
	}

	if err := bxh.Router.Start(); err != nil {
		return fmt.Errorf("router start: %w", err)
	}

	bxh.start()

	bxh.printLogo()

	return nil
}

func (bxh *BitXHub) Stop() error {
	if err := bxh.Executor.Stop(); err != nil {
		return fmt.Errorf("executor stop: %w", err)
	}

	if err := bxh.Router.Stop(); err != nil {
		return fmt.Errorf("InterchainRouter stop: %w", err)
	}

	if !bxh.repo.Config.Solo {
		if err := bxh.PeerMgr.Stop(); err != nil {
			return fmt.Errorf("network stop: %w", err)
		}
	}

	bxh.Order.Stop()

	bxh.cancel()

	bxh.logger.Info("Bitxhub stopped")

	return nil
}

func (bxh *BitXHub) printLogo() {
	for {
		time.Sleep(100 * time.Millisecond)
		if bxh.Order.Ready() {
			bxh.logger.WithFields(logrus.Fields{
				"plugin_path": bxh.repo.Config.Order.Plugin,
			}).Info("Order is ready")
			fmt.Println()
			fmt.Println("=======================================================")
			fig := figure.NewFigure("BitXHub", "slant", true)
			fig.Print()
			fmt.Println()
			fmt.Println("=======================================================")
			fmt.Println()
			return
		}
	}
}
