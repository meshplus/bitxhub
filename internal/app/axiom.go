package app

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"syscall"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom/api/jsonrpc"
	"github.com/axiomesh/axiom/internal/executor"
	"github.com/axiomesh/axiom/internal/finance"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/internal/ledger/genesis"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/order/rbft"
	"github.com/axiomesh/axiom/internal/order/solo"
	"github.com/axiomesh/axiom/internal/peermgr"
	"github.com/axiomesh/axiom/internal/storages"
	"github.com/axiomesh/axiom/pkg/loggers"
	"github.com/axiomesh/axiom/pkg/profile"
	"github.com/axiomesh/axiom/pkg/repo"
)

type Axiom struct {
	Ledger        *ledger.Ledger
	BlockExecutor executor.Executor
	ViewExecutor  executor.Executor
	Order         order.Order
	PeerMgr       peermgr.PeerManager

	repo   *repo.Repo
	logger logrus.FieldLogger

	Monitor       *profile.Monitor
	Pprof         *profile.Pprof
	LoggerWrapper *loggers.LoggerWrapper
	Jsonrpc       *jsonrpc.ChainBrokerService

	Ctx    context.Context
	Cancel context.CancelFunc
}

func NewAxiom(rep *repo.Repo) (*Axiom, error) {
	repoRoot := rep.Config.RepoRoot
	bxh, err := GenerateAxiomWithoutOrder(rep)
	if err != nil {
		return nil, fmt.Errorf("generate axiom without order failed: %w", err)
	}

	chainMeta := bxh.Ledger.GetChainMeta()

	m := rep.NetworkConfig.GetVpInfos()

	var orderCon func(opt ...order.Option) (order.Order, error)
	// Get the order constructor according to different order type.
	switch rep.Config.Order.Type {
	case repo.OrderTypeSolo:
		orderCon = solo.NewNode
	case repo.OrderTypeRbft:
		orderCon = rbft.NewNode
	default:
		return nil, fmt.Errorf("unsupport order type: %s", rep.Config.Order.Type)
	}

	order, err := orderCon(
		order.WithConfig(rep.OrderConfig),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithStorageType(rep.Config.Ledger.Kv),
		order.WithOrderType(rep.Config.Order.Type),
		order.WithPrivKey(rep.NodeKey),
		order.WithNodes(m),
		order.WithID(rep.NetworkConfig.ID),
		order.WithIsNew(rep.NetworkConfig.New),
		order.WithPeerManager(bxh.PeerMgr),
		order.WithLogger(loggers.Logger(loggers.Order)),
		order.WithApplied(chainMeta.Height),
		order.WithDigest(chainMeta.BlockHash.String()),
		order.WithGetChainMetaFunc(bxh.Ledger.GetChainMeta),
		order.WithGetBlockByHeightFunc(bxh.Ledger.GetBlock),
		order.WithGetAccountNonceFunc(bxh.Ledger.Copy().GetNonce),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize order failed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bxh.Ctx = ctx
	bxh.Cancel = cancel
	bxh.Order = order

	return bxh, nil
}

func GenerateAxiomWithoutOrder(rep *repo.Repo) (*Axiom, error) {
	repoRoot := rep.Config.RepoRoot
	logger := loggers.Logger(loggers.App)

	if err := storages.Initialize(repoRoot, rep.Config.Ledger.Kv); err != nil {
		return nil, fmt.Errorf("storages initialize: %w", err)
	}

	bcStorage, err := storages.Get(storages.BlockChain)
	if err != nil {
		return nil, fmt.Errorf("create blockchain storage: %w", err)
	}

	stateStorage, err := ledger.OpenStateDB(repo.GetStoragePath(repoRoot, "ledger"), rep.Config.Ledger.Kv)
	if err != nil {
		return nil, fmt.Errorf("create stateDB: %w", err)
	}

	bf, err := blockfile.NewBlockFile(repoRoot, loggers.Logger(loggers.Storage))
	if err != nil {
		return nil, fmt.Errorf("blockfile initialize: %w", err)
	}

	// 0. load ledger
	rwLdg, err := ledger.New(rep, bcStorage, stateStorage, bf, nil, loggers.Logger(loggers.Executor))
	if err != nil {
		return nil, fmt.Errorf("create RW ledger: %w", err)
	}

	viewLdg := &ledger.Ledger{
		ChainLedger: rwLdg.ChainLedger,
	}
	// create read only ledger
	viewLdg.StateLedger, err = ledger.NewSimpleLedger(rep, stateStorage, nil, loggers.Logger(loggers.Executor))
	if err != nil {
		return nil, fmt.Errorf("create readonly ledger: %w", err)
	}

	// 1. create executor and view executor
	viewExec, err := executor.New(viewLdg, loggers.Logger(loggers.Executor), rep.Config, func() (*big.Int, error) {
		return big.NewInt(0), nil
	})
	if err != nil {
		return nil, fmt.Errorf("create ViewExecutor: %w", err)
	}

	if rwLdg.ChainLedger.GetChainMeta().Height == 0 {
		if err := genesis.Initialize(&rep.Config.Genesis, rep.NetworkConfig.Nodes, rep.NetworkConfig.N, rwLdg, viewExec); err != nil {
			return nil, err
		}
		logger.WithFields(logrus.Fields{
			"genesis block hash": rwLdg.ChainLedger.GetChainMeta().BlockHash,
		}).Info("Initialize genesis")
	}

	gas := finance.NewGas(rep, viewLdg)
	getGasPrice := func() (*big.Int, error) {
		result := new(big.Int)
		gas, err := gas.GetGasPrice()
		if err != nil {
			return nil, err
		}
		result.SetUint64(gas)
		return result, nil
	}

	txExec, err := executor.New(rwLdg, loggers.Logger(loggers.Executor), rep.Config, getGasPrice)
	if err != nil {
		return nil, fmt.Errorf("create BlockExecutor: %w", err)
	}

	peerMgr, err := peermgr.New(rep, loggers.Logger(loggers.P2P), rwLdg)
	if err != nil {
		return nil, fmt.Errorf("create peer manager: %w", err)
	}

	return &Axiom{
		repo:          rep,
		logger:        logger,
		Ledger:        rwLdg,
		BlockExecutor: txExec,
		ViewExecutor:  viewExec,
		PeerMgr:       peerMgr,
	}, nil
}

func (bxh *Axiom) Start() error {
	if err := bxh.raiseUlimit(2048); err != nil {
		return fmt.Errorf("raise ulimit: %w", err)
	}

	if bxh.repo.Config.Order.Type != repo.OrderTypeSolo {
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

	bxh.start()

	bxh.printLogo()

	return nil
}

func (bxh *Axiom) Stop() error {
	if err := bxh.BlockExecutor.Stop(); err != nil {
		return fmt.Errorf("block executor stop: %w", err)
	}

	if err := bxh.ViewExecutor.Stop(); err != nil {
		return fmt.Errorf("view executor stop: %w", err)
	}

	if bxh.repo.Config.Order.Type != repo.OrderTypeSolo {
		if err := bxh.PeerMgr.Stop(); err != nil {
			return fmt.Errorf("network stop: %w", err)
		}
	}

	bxh.Order.Stop()

	bxh.Cancel()

	bxh.logger.Info("Axiom stopped")

	return nil
}

func (bxh *Axiom) ReConfig(repo *repo.Repo) {
	if repo.Config != nil {
		config := repo.Config
		loggers.ReConfig(config)

		if err := bxh.Jsonrpc.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig json rpc failed: %v", err)
		}

		if err := bxh.Monitor.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig Monitor failed: %v", err)
		}

		if err := bxh.Pprof.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig Pprof failed: %v", err)
		}
	}
}

func (bxh *Axiom) printLogo() {
	for {
		time.Sleep(100 * time.Millisecond)
		err := bxh.Order.Ready()
		if err == nil {
			bxh.logger.WithFields(logrus.Fields{
				"order_type": bxh.repo.Config.Order.Type,
			}).Info("Order is ready")
			fmt.Println()
			fmt.Println("=======================================================")
			fig := figure.NewFigure("Axiom", "slant", true)
			fig.Print()
			fmt.Println()
			fmt.Println("=======================================================")
			fmt.Println()
			return
		}
	}
}

func (bxh *Axiom) raiseUlimit(limitNew uint64) error {
	_, err := fdlimit.Raise(limitNew)
	if err != nil {
		return fmt.Errorf("set limit failed: %w", err)
	}

	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		return fmt.Errorf("getrlimit error: %w", err)
	}

	if limit.Cur != limitNew && limit.Cur != limit.Max {
		return errors.New("failed to raise ulimit")
	}

	bxh.logger.WithFields(logrus.Fields{
		"ulimit": limit.Cur,
	}).Infof("Ulimit raised")

	return nil
}
