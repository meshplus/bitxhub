package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	bkg "github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/common-nighthawk/go-figure"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub/api/gateway"
	"github.com/meshplus/bitxhub/api/grpc"
	"github.com/meshplus/bitxhub/api/jsonrpc"
	_ "github.com/meshplus/bitxhub/imports"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/executor/oracle/appchain"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/genesis"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/profile"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/internal/router"
	"github.com/meshplus/bitxhub/internal/storages"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/tssmgr"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

type BitXHub struct {
	Ledger        *ledger.Ledger
	BlockExecutor executor.Executor
	ViewExecutor  executor.Executor
	Router        router.Router
	Order         order.Order
	PeerMgr       peermgr.PeerManager
	TssMgr        *tssmgr.TssMgr

	repo   *repo.Repo
	logger logrus.FieldLogger

	Monitor       *profile.Monitor
	Pprof         *profile.Pprof
	LoggerWrapper *loggers.LoggerWrapper
	Gateway       *gateway.Gateway
	Grpc          *grpc.ChainBrokerService
	Jsonrpc       *jsonrpc.ChainBrokerService

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
			return nil, fmt.Errorf("read order config error: %w", err)
		}
		err = ioutil.WriteFile(filepath.Join(repoRoot, "order.toml"), fileData, 0644)
		if err != nil {
			return nil, fmt.Errorf("write order.toml failed: %w", err)
		}
	}

	bxh, err := GenerateBitXHubWithoutOrder(rep)
	if err != nil {
		return nil, fmt.Errorf("generate bitxhub without order failed: %w", err)
	}

	chainMeta := bxh.Ledger.GetChainMeta()

	m := rep.NetworkConfig.GetVpInfos()

	//Get the order constructor according to different order type.
	orderCon, err := agency.GetOrderConstructor(rep.Config.Order.Type)
	if err != nil {
		return nil, fmt.Errorf("get order %s failed: %w", rep.Config.Order.Type, err)
	}

	order, err := orderCon(
		order.WithRepoRoot(orderRoot),
		order.WithStoragePath(repo.GetStoragePath(repoRoot, "order")),
		order.WithOrderType(rep.Config.Order.Type),
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

	err := asym.ConfiguredKeyType(rep.Config.Crypto.Algorithms)
	if err != nil {
		return nil, fmt.Errorf("set configured key type failed: %w", err)
	}

	supportCryptoTypeToName := asym.GetConfiguredKeyType()
	printType := "Supported crypto type:"
	for _, name := range supportCryptoTypeToName {
		printType = fmt.Sprintf("%s%s ", printType, name)
	}
	printType = fmt.Sprintf("%s\n", printType)
	fmt.Println(printType)

	if err := storages.Initialize(repoRoot); err != nil {
		return nil, fmt.Errorf("storages initialize: %w", err)
	}

	bcStorage, err := storages.Get(storages.BlockChain)
	if err != nil {
		return nil, fmt.Errorf("create blockchain storage: %w", err)
	}

	stateStorage, err := ledger.OpenStateDB(repo.GetStoragePath(repoRoot, "ledger"), rep.Config.Ledger.Type)
	if err != nil {
		return nil, fmt.Errorf("create tm-leveldb: %w", err)
	}

	bf, err := blockfile.NewBlockFile(repoRoot, loggers.Logger(loggers.Storage))
	if err != nil {
		return nil, fmt.Errorf("blockfile initialize: %w", err)
	}

	appchainClient := &appchain.Client{}
	if rep.Config.Appchain.Enable {
		appchainClient, err = appchain.NewAppchainClient(filepath.Join(repoRoot, rep.Config.Appchain.EthHeaderPath), repo.GetStoragePath(repoRoot, "appchain_client"), loggers.Logger(loggers.Executor))
		if err != nil {
			return nil, fmt.Errorf("initialize appchain client failed: %w", err)
		}
	}

	// 0. load ledger
	rwLdg, err := ledger.New(rep, bcStorage, stateStorage, bf, nil, loggers.Logger(loggers.Executor))
	if err != nil {
		return nil, fmt.Errorf("create RW ledger: %w", err)
	}

	viewLdg := &ledger.Ledger{
		ChainLedger: rwLdg.ChainLedger,
	}
	if rep.Config.Ledger.Type == "simple" {
		// create read only ledger
		viewLdg.StateLedger, err = ledger.NewSimpleLedger(rep, stateStorage.(storage.Storage), nil, loggers.Logger(loggers.Executor))
		if err != nil {
			return nil, fmt.Errorf("create readonly ledger: %w", err)
		}
	} else {
		viewLdg.StateLedger = rwLdg.StateLedger.(*ledger2.ComplexStateLedger).Copy()
	}

	// 1. create executor and view executor
	viewExec, err := executor.New(viewLdg, loggers.Logger(loggers.Executor), appchainClient, rep.Config, big.NewInt(0))
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

	txExec, err := executor.New(rwLdg, loggers.Logger(loggers.Executor), appchainClient, rep.Config, big.NewInt(int64(rep.Config.Genesis.BvmGasPrice)))
	if err != nil {
		return nil, fmt.Errorf("create BlockExecutor: %w", err)
	}

	peerMgr, err := peermgr.New(rep, loggers.Logger(loggers.P2P), rwLdg)
	if err != nil {
		return nil, fmt.Errorf("create peer manager: %w", err)
	}

	tssMgr := &tssmgr.TssMgr{}
	if rep.Config.Tss.EnableTSS {
		preParams, err := getPreparams(repoRoot)
		if err != nil {
			return nil, fmt.Errorf("get preparams error: %w", err)
		}

		var preParam *bkg.LocalPreParams
		if len(preParams) <= int(rep.NetworkConfig.ID) {
			preParam = nil
		} else {
			preParam = preParams[rep.NetworkConfig.ID-1]
		}

		tssMgr, err = tssmgr.NewTssMgr(rep.Key.Libp2pPrivKey, rep.Config.Tss, rep.NetworkConfig, repoRoot, preParam, peerMgr, loggers.Logger(loggers.TSS))
		if err != nil {
			return nil, fmt.Errorf("create tss manager: %w, %v", err, rep.Config.Tss.PreParamTimeout)
		}
		peerMgr.Tss = tssMgr
	}

	return &BitXHub{
		repo:          rep,
		logger:        logger,
		Ledger:        rwLdg,
		BlockExecutor: txExec,
		ViewExecutor:  viewExec,
		PeerMgr:       peerMgr,
		TssMgr:        tssMgr,
	}, nil
}

func getPreparams(repoRoot string) ([]*bkg.LocalPreParams, error) {
	const (
		preParamTestFile = "preParam_test.data"
	)
	var preParamArray []*bkg.LocalPreParams
	buf, err := ioutil.ReadFile(path.Join(repoRoot, preParamTestFile))
	if err != nil {
		return nil, fmt.Errorf("get preparams error: %v", err)
	}
	preParamsStr := strings.Split(string(buf), "\n")
	for _, item := range preParamsStr {
		var preParam bkg.LocalPreParams
		val, err := hex.DecodeString(item)
		if err != nil {
			return nil, fmt.Errorf("get preparams error: %v", err)
		}
		err = json.Unmarshal(val, &preParam)
		preParamArray = append(preParamArray, &preParam)
	}
	return preParamArray, nil
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

	if bxh.repo.Config.Tss.EnableTSS {
		bxh.TssMgr.Start(bxh.Order.Quorum() - 1)
		time1 := time.Now()
		bxh.logger.Debugf("...... tss start key gen")
		if err := bxh.TssMgr.Keygen(false); err != nil {
			bxh.logger.Errorf("tss key generate error: %v", err)
			return fmt.Errorf("tss key generate: %w", err)
		}
		timeKeygen := time.Since(time1)
		bxh.logger.Infof("=============================keygen time: %v", timeKeygen)
	}

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

	if bxh.repo.Config.Tss.EnableTSS {
		bxh.TssMgr.Stop()
	}

	bxh.Cancel()

	bxh.logger.Info("Bitxhub stopped")

	return nil
}

func (bxh *BitXHub) ReConfig(repo *repo.Repo) {
	if repo.Config != nil {
		config := repo.Config
		loggers.ReConfig(config)

		if err := bxh.Jsonrpc.ReConfig(config); err != nil {
			bxh.logger.Errorf("reconfig json rpc failed: %v", err)
		}

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
				"order_type": bxh.repo.Config.Order.Type,
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
		return fmt.Errorf("set limit failed: %w", err)
	}

	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		return fmt.Errorf("getrlimit error: %w", err)
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

func (bxh *BitXHub) GetSoloType() bool {
	return bxh.repo.Config.Solo
}
