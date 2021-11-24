package app

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"syscall"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub/api/gateway"
	"github.com/meshplus/bitxhub/api/grpc"
	_ "github.com/meshplus/bitxhub/imports"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/genesis"
	"github.com/meshplus/bitxhub/internal/loggers"
	orderplg "github.com/meshplus/bitxhub/internal/plugins"
	"github.com/meshplus/bitxhub/internal/profile"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/internal/router"
	"github.com/meshplus/bitxhub/internal/storages"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

type BitXHub struct {
	Ledger        ledger.Ledger
	BlockExecutor executor.Executor
	ViewExecutor  executor.Executor
	Router        router.Router
	Order         order.Order
	PeerMgr       peermgr.PeerManager

	Monitor       *profile.Monitor
	Pprof         *profile.Pprof
	LoggerWrapper *loggers.LoggerWrapper
	Gateway       *gateway.Gateway
	Grpc          *grpc.ChainBrokerService

	repo   *repo.Repo
	logger logrus.FieldLogger

	Ctx    context.Context
	Cancel context.CancelFunc
}

func NewBitXHub(rep *repo.Repo, orderPath string) (*BitXHub, error) {
	repoRoot := rep.Config.RepoRoot
	var orderRoot string
	if len(orderPath) == 0 {
		orderRoot = repoRoot
	} else {
		orderRoot = filepath.Dir(orderPath)
		fileData, err := ioutil.ReadFile(orderPath)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(filepath.Join(repoRoot, "order.toml"), fileData, 0644)
		if err != nil {
			return nil, err
		}
	}

	bxh, err := GenerateBitXHubWithoutOrder(rep)
	if err != nil {
		return nil, err
	}

	chainMeta := bxh.Ledger.GetChainMeta()

	m := rep.NetworkConfig.GetVpInfos()

	order, err := orderplg.New(
		order.WithRepoRoot(orderRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithPluginPath(rep.Config.Plugin),
		order.WithNodes(m),
		order.WithID(rep.NetworkConfig.ID),
		order.WithIsNew(rep.NetworkConfig.New),
		order.WithPeerManager(bxh.PeerMgr),
		order.WithLogger(loggers.Logger(loggers.Order)),
		order.WithApplied(chainMeta.Height),
		order.WithDigest(chainMeta.BlockHash.String()),
		order.WithGetChainMetaFunc(bxh.Ledger.GetChainMeta),
		order.WithGetBlockByHeightFunc(bxh.Ledger.GetBlock),
		order.WithGetAccountNonceFunc(bxh.Ledger.GetNonce),
	)
	if err != nil {
		return nil, err
	}

	r, err := router.New(loggers.Logger(loggers.Router), rep, bxh.Ledger, bxh.PeerMgr, order.Quorum())
	if err != nil {
		return nil, fmt.Errorf("create InterchainRouter: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bxh.Ctx = ctx
	bxh.Cancel = cancel
	bxh.Order = order
	bxh.Router = r

	return bxh, nil
}

func GenerateBitXHubWithoutOrder(rep *repo.Repo) (*BitXHub, error) {
	repoRoot := rep.Config.RepoRoot
	logger := loggers.Logger(loggers.App)

	if err := storages.Initialize(repoRoot); err != nil {
		return nil, fmt.Errorf("storages initialize: %w", err)
	}

	bcStorage, err := storages.Get(storages.BlockChain)
	if err != nil {
		return nil, fmt.Errorf("create blockchain storage: %w", err)
	}

	ldb, err := leveldb.New(repo.GetStoragePath(repoRoot, "ledger"))
	if err != nil {
		return nil, fmt.Errorf("create tm-leveldb: %w", err)
	}

	bf, err := blockfile.NewBlockFile(repoRoot, loggers.Logger(loggers.Storage))
	if err != nil {
		return nil, fmt.Errorf("blockfile initialize: %w", err)
	}

	// 0. load ledger
	rwLdg, err := ledger.New(rep, bcStorage, ldb, bf, nil, loggers.Logger(loggers.Executor))
	if err != nil {
		return nil, fmt.Errorf("create RW ledger: %w", err)
	}

	if rwLdg.GetChainMeta().Height == 0 {
		if err := genesis.Initialize(&rep.Config.Genesis, rwLdg); err != nil {
			return nil, err
		}
		logger.Info("Initialize genesis")
	}

	// create read only ledger
	viewLdg, err := ledger.New(rep, bcStorage, ldb, bf, rwLdg.AccountCache(), loggers.Logger(loggers.Executor))
	if err != nil {
		return nil, fmt.Errorf("create readonly ledger: %w", err)
	}

	// 1. create executor and view executor
	txExec, err := executor.New(rwLdg, loggers.Logger(loggers.Executor), rep.Config.Executor.Type)
	if err != nil {
		return nil, fmt.Errorf("create BlockExecutor: %w", err)
	}

	viewExec, err := executor.New(viewLdg, loggers.Logger(loggers.Executor), rep.Config.Executor.Type)
	if err != nil {
		return nil, fmt.Errorf("create ViewExecutor: %w", err)
	}

	peerMgr, err := peermgr.New(rep, loggers.Logger(loggers.P2P), rwLdg)
	if err != nil {
		return nil, fmt.Errorf("create peer manager: %w", err)
	}

	return &BitXHub{
		repo:          rep,
		logger:        logger,
		Ledger:        rwLdg,
		BlockExecutor: txExec,
		ViewExecutor:  viewExec,
		PeerMgr:       peerMgr,
	}, nil
}

func (bxh *BitXHub) Start() error {

	if err := bxh.raiseUlimit(2048); err != nil {
		return fmt.Errorf("raise ulimit: %w", err)
	}

	if !bxh.repo.Config.Solo {
		if err := bxh.PeerMgr.Start(); err != nil {
			return fmt.Errorf("peer manager start: %w", err)
		}
	}

	if err := bxh.Order.Start(); err != nil {
		return fmt.Errorf("order start: %w", err)
	}

	if err := bxh.BlockExecutor.Start(); err != nil {
		return fmt.Errorf("block executor start: %w", err)
	}

	if err := bxh.ViewExecutor.Start(); err != nil {
		return fmt.Errorf("view executor start: %w", err)
	}

	if err := bxh.Router.Start(); err != nil {
		return fmt.Errorf("router start: %w", err)
	}

	bxh.start()

	bxh.printLogo()

	return nil
}

func (bxh *BitXHub) Stop() error {
	if err := bxh.BlockExecutor.Stop(); err != nil {
		return fmt.Errorf("block executor stop: %w", err)
	}

	if err := bxh.ViewExecutor.Stop(); err != nil {
		return fmt.Errorf("view executor stop: %w", err)
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

	bxh.Cancel()

	bxh.logger.Info("Bitxhub stopped")

	return nil
}

func (bxh *BitXHub) ReConfig(repo *repo.Repo) {
	if repo.Config != nil {
		config := repo.Config
		loggers.ReConfig(config)

		if err := bxh.Grpc.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig grpc failed: %v", err)
		}

		if err := bxh.Gateway.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig gateway failed: %v", err)
		}

		if err := bxh.PeerMgr.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig PeerMgr failed: %v", err)
		}

		if err := bxh.Monitor.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig Monitor failed: %v", err)
		}

		if err := bxh.Pprof.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig Pprof failed: %v", err)
		}
	}
	if repo.NetworkConfig != nil {
		config := repo.NetworkConfig
		if err := bxh.PeerMgr.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig PeerMgr failed: %v", err)
		}
	}
}

func (bxh *BitXHub) printLogo() {
	for {
		time.Sleep(100 * time.Millisecond)
		err := bxh.Order.Ready()
		if err == nil {
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

func (bxh *BitXHub) raiseUlimit(limitNew uint64) error {
	_, err := fdlimit.Raise(limitNew)
	if err != nil {
		return err
	}

	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		return err
	}

	if limit.Cur != limitNew && limit.Cur != limit.Max {
		return fmt.Errorf("failed to raise ulimit")
	}

	bxh.logger.WithFields(logrus.Fields{
		"ulimit": limit.Cur,
	}).Infof("Ulimit raised")

	return nil
}

func (bxh *BitXHub) GetPrivKey() *repo.Key {
	return bxh.repo.Key
}
