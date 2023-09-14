package app

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/sirupsen/logrus"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-kit/storage/blockfile"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/api/jsonrpc"
	"github.com/axiomesh/axiom/internal/executor"
	"github.com/axiomesh/axiom/internal/executor/executor_dev"
	"github.com/axiomesh/axiom/internal/executor/system/base"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/internal/ledger/genesis"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/axiomesh/axiom/internal/order/common"
	"github.com/axiomesh/axiom/internal/peermgr"
	"github.com/axiomesh/axiom/internal/storages"
	"github.com/axiomesh/axiom/pkg/loggers"
	"github.com/axiomesh/axiom/pkg/profile"
	"github.com/axiomesh/axiom/pkg/repo"
)

type Axiom struct {
	ViewLedger    *ledger.Ledger
	BlockExecutor executor.Executor
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

func NewAxiom(rep *repo.Repo, ctx context.Context, cancel context.CancelFunc) (*Axiom, error) {
	repoRoot := rep.Config.RepoRoot
	axm, err := GenerateAxiomWithoutOrder(rep)
	if err != nil {
		return nil, fmt.Errorf("generate axiom without order failed: %w", err)
	}
	axm.Ctx = ctx
	axm.Cancel = cancel

	chainMeta := axm.ViewLedger.ChainLedger.GetChainMeta()

	var getNonceFunc func(address *types.Address) uint64
	if rep.Config.Order.Type == repo.OrderTypeSoloDev {
		getNonceFunc = axm.ViewLedger.StateLedger.GetNonce
	} else {
		getNonceFunc = axm.ViewLedger.Copy().StateLedger.GetNonce
	}

	order, err := order.New(
		rep.Config.Order.Type,
		common.WithConfig(rep.OrderConfig),
		common.WithSelfAccountAddress(rep.NodeAddress),
		common.WithGenesisEpochInfo(rep.Config.Genesis.EpochInfo.Clone()),
		common.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		common.WithStorageType(rep.Config.Ledger.Kv),
		common.WithOrderType(rep.Config.Order.Type),
		common.WithPrivKey(rep.NodeKey),
		common.WithPeerManager(axm.PeerMgr),
		common.WithLogger(loggers.Logger(loggers.Order)),
		common.WithApplied(chainMeta.Height),
		common.WithDigest(chainMeta.BlockHash.String()),
		common.WithGenesisDigest(axm.ViewLedger.ChainLedger.GetBlockHash(1).String()),
		common.WithGetChainMetaFunc(axm.ViewLedger.ChainLedger.GetChainMeta),
		common.WithGetAccountBalanceFunc(axm.ViewLedger.StateLedger.GetBalance),
		common.WithGetAccountNonceFunc(getNonceFunc),
		common.WithGetEpochInfoFromEpochMgrContractFunc(func(epoch uint64) (*rbft.EpochInfo, error) {
			return base.GetEpochInfo(axm.ViewLedger.StateLedger, epoch)
		}),
		common.WithGetCurrentEpochInfoFromEpochMgrContractFunc(func() (*rbft.EpochInfo, error) {
			return base.GetCurrentEpochInfo(axm.ViewLedger.StateLedger)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize order failed: %w", err)
	}

	axm.Order = order

	if err := axm.raiseUlimit(rep.Config.Ulimit); err != nil {
		return nil, fmt.Errorf("raise ulimit: %w", err)
	}

	return axm, nil
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

	if rwLdg.ChainLedger.GetChainMeta().Height == 0 {
		if err := genesis.Initialize(&rep.Config.Genesis, rwLdg); err != nil {
			return nil, err
		}
		logger.WithFields(logrus.Fields{
			"genesis block hash": rwLdg.ChainLedger.GetChainMeta().BlockHash,
		}).Info("Initialize genesis")
	}

	var txExec executor.Executor
	log := loggers.Logger(loggers.Executor)
	if rep.Config.Executor.Type == repo.ExecTypeDev {
		txExec, err = executor_dev.New(log)
	} else {
		txExec, err = executor.New(rwLdg, log, rep)
	}
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
		ViewLedger:    rwLdg.Copy(),
		BlockExecutor: txExec,
		PeerMgr:       peerMgr,
	}, nil
}

func (axm *Axiom) Start() error {
	var err error
	// read current epoch info from ledger
	axm.repo.EpochInfo, err = base.GetCurrentEpochInfo(axm.ViewLedger.StateLedger)
	if err != nil {
		return err
	}

	if repo.SupportMultiNode[axm.repo.Config.Order.Type] {
		if err := axm.PeerMgr.Start(); err != nil {
			return fmt.Errorf("peer manager start: %w", err)
		}
	}

	if err := axm.Order.Start(); err != nil {
		return fmt.Errorf("order start: %w", err)
	}

	if err := axm.BlockExecutor.Start(); err != nil {
		return fmt.Errorf("block executor start: %w", err)
	}

	axm.start()

	axm.printLogo()

	return nil
}

func (axm *Axiom) Stop() error {
	if err := axm.BlockExecutor.Stop(); err != nil {
		return fmt.Errorf("block executor stop: %w", err)
	}

	if axm.repo.Config.Order.Type != repo.OrderTypeSolo {
		if err := axm.PeerMgr.Stop(); err != nil {
			return fmt.Errorf("network stop: %w", err)
		}
	}

	axm.Order.Stop()

	axm.Cancel()

	axm.logger.Info("Axiom stopped")

	return nil
}

func (axm *Axiom) ReConfig(repo *repo.Repo) {
	if repo.Config != nil {
		config := repo.Config
		loggers.ReConfig(config)

		if err := axm.Jsonrpc.ReConfig(config); err != nil {
			axm.logger.Errorf("reconfig json rpc failed: %v", err)
		}

		if err := axm.Monitor.ReConfig(config); err != nil {
			axm.logger.Errorf("reconfig Monitor failed: %v", err)
		}

		if err := axm.Pprof.ReConfig(config); err != nil {
			axm.logger.Errorf("reconfig Pprof failed: %v", err)
		}
	}
}

func (axm *Axiom) printLogo() {
	for {
		time.Sleep(100 * time.Millisecond)
		err := axm.Order.Ready()
		if err == nil {
			axm.logger.WithFields(logrus.Fields{
				"order_type": axm.repo.Config.Order.Type,
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

func (axm *Axiom) raiseUlimit(limitNew uint64) error {
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

	axm.logger.WithFields(logrus.Fields{
		"ulimit": limit.Cur,
	}).Infof("Ulimit raised")

	return nil
}
