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
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	bkg "github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/common-nighthawk/go-figure"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-core/tss"
	"github.com/meshplus/bitxhub-core/tss/conversion"
	"github.com/meshplus/bitxhub-core/tss/keygen"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-model/pb"
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
	TssMgr        tss.Tss

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

	tssMgr := &tss.TssManager{}
	if rep.Config.Tss.EnableTSS {
		preParams, err := getPreparams(repoRoot)
		if err != nil {
			return nil, fmt.Errorf("get preparams error: %w", err)
		}

		tssMgr, err = tss.NewTss(repoRoot, peerMgr, rep.Config.Tss, 0, rep.Key.Libp2pPrivKey, loggers.Logger(loggers.TSS), filepath.Join(repoRoot, rep.Config.Tss.TssConfPath), preParams[rep.NetworkConfig.ID-1])
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
		if err := bxh.keygen(); err != nil {
			bxh.logger.Errorf("tss key generate error: %v", err)
			//return fmt.Errorf("tss key generate: %w", err)
		}
		timeKeygen := time.Since(time1).Milliseconds()
		bxh.logger.Infof("=============================keygen time: %d", timeKeygen)
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

func (bxh *BitXHub) keygen() error {
	myStatus := true
	var poolPkMap = map[string]string{}

	// 1. 获取本地持久化公钥信息
	myPoolPkAddr, _, err := bxh.TssMgr.GetTssPubkey()
	if err != nil {
		myStatus = false
	}
	poolPkMap[strconv.Itoa(int(bxh.repo.NetworkConfig.ID))] = myPoolPkAddr

	// 2. 如果自己有持久化数据，向其他节点请求公钥，验证是否一致
	// 如果自己没有持久化数据，直接keygen
	if myStatus {
		// 2.1 向其他节点获取公钥
		for id, addr := range bxh.fetchTssPkFromOtherPeers() {
			poolPkMap[id] = addr
		}

		// 2.2 检查公钥一致个数
		ok, ids, err := checkPoolPkMap(poolPkMap, bxh.Order.Quorum())
		if err != nil {
			return fmt.Errorf("check pool pk map: %w", err)
		}
		// 2.3 如果检查累计一致的公钥个数达到q，门限签名可以正常使用，将不一致的节点踢出签名组合即可
		// 如果检查不通过，当前的门限签名已经不能正常使用，需要重新keygen
		if ok {
			bxh.logger.WithFields(logrus.Fields{
				"ids": ids,
			}).Infof("delete culprits from localState")
			// 将不一致的参与方踢出tss节点
			if err := bxh.TssMgr.DeleteCulpritsFromLocalState(ids); err != nil {
				return fmt.Errorf("handle culprits: %w", err)
			}
			// 继续使用之前的公钥
			return nil
		}
	}

	// 3. 开始keygen
	var (
		keys = []crypto.PubKey{bxh.repo.Key.Libp2pPrivKey.GetPublic()}
	)

	for _, key := range bxh.fetchPkFromOtherPeers() {
		keys = append(keys, key)
	}

	bxh.logger.WithFields(logrus.Fields{
		"pubkeys": keys,
	}).Infof("tss keygen peer pubkeys")

	_, err = bxh.TssMgr.Keygen(keygen.NewRequest(keys))
	if err != nil {
		return fmt.Errorf("tss key generate: %w", err)
	}

	return nil
}

func checkPoolPkMap(pkAddrMap map[string]string, quorum uint64) (bool, []string, error) {
	freq := make(map[string]int, len(pkAddrMap))
	for _, addr := range pkAddrMap {
		freq[addr]++
	}
	maxFreq := -1
	var pkAddr string
	for key, counter := range freq {
		if counter > maxFreq {
			maxFreq = counter
			pkAddr = key
		}
	}

	if pkAddr == "" {
		return false, nil, nil
	}

	if maxFreq < int(quorum) {
		return false, nil, nil
	}

	ids := []string{}
	for id, addr := range pkAddrMap {
		if addr != pkAddr {
			ids = append(ids, id)
		}
	}

	return true, ids, nil
}

func (bxh *BitXHub) requestPkFromPeer(pid uint64) (crypto.PubKey, error) {
	req := pb.Message{
		Type: pb.Message_FETCH_P2P_PUBKEY,
	}

	resp, err := bxh.PeerMgr.Send(pid, &req)
	if err != nil {
		return nil, fmt.Errorf("send message to %d failed: %w", pid, err)
	}
	if resp == nil || resp.Type != pb.Message_FETCH_P2P_PUBKEY_ACK {
		return nil, fmt.Errorf("invalid fetch p2p pk resp")
	}

	return conversion.GetPubKeyFromPubKeyData(resp.Data)
}

func (bxh *BitXHub) requestTssPkFromPeer(pid uint64) (string, error) {
	req := pb.Message{
		Type: pb.Message_FETCH_TSS_PUBKEY,
	}

	resp, err := bxh.PeerMgr.Send(pid, &req)
	if err != nil {
		return "", fmt.Errorf("send message to %d failed: %w", pid, err)
	}
	if resp == nil || resp.Type != pb.Message_FETCH_TSS_PUBKEY_ACK {
		return "", fmt.Errorf("invalid fetch tss pk resp")
	}

	return string(resp.Data), nil
}

func (bxh *BitXHub) fetchPkFromOtherPeers() map[uint64]crypto.PubKey {
	var (
		result = make(map[uint64]crypto.PubKey)
		wg     = sync.WaitGroup{}
		lock   = sync.Mutex{}
	)

	wg.Add(len(bxh.PeerMgr.OtherPeers()))
	for pid := range bxh.PeerMgr.OtherPeers() {
		go func(pid uint64, result map[uint64]crypto.PubKey, wg *sync.WaitGroup, lock *sync.Mutex) {
			if err := retry.Retry(func(attempt uint) error {
				pk, err := bxh.requestPkFromPeer(pid)
				if err != nil {
					bxh.logger.WithFields(logrus.Fields{
						"pid": pid,
						"err": err.Error(),
					}).Warnf("Get peer pubkey with error")
					return err
				} else {
					lock.Lock()
					result[pid] = pk
					lock.Unlock()
					return nil
				}
			}, strategy.Wait(500*time.Millisecond),
			); err != nil {
				bxh.logger.WithFields(logrus.Fields{
					"pid": pid,
					"err": err.Error(),
				}).Warnf("retry error")
			}

			wg.Done()
		}(pid, result, &wg, &lock)
	}

	wg.Wait()

	return result
}

func (bxh *BitXHub) fetchTssPkFromOtherPeers() map[string]string {
	var (
		result = make(map[string]string)
		wg     = sync.WaitGroup{}
		lock   = sync.Mutex{}
	)

	wg.Add(len(bxh.PeerMgr.OtherPeers()))
	for pid := range bxh.PeerMgr.OtherPeers() {
		go func(pid uint64, result map[string]string, wg *sync.WaitGroup, lock *sync.Mutex) {
			if err := retry.Retry(func(attempt uint) error {
				pkAddr, err := bxh.requestTssPkFromPeer(pid)
				if err != nil {
					bxh.logger.WithFields(logrus.Fields{
						"pid": pid,
						"err": err.Error(),
					}).Warnf("Get peer tss pubkey with error")
					return err
				} else {
					lock.Lock()
					result[strconv.Itoa(int(pid))] = pkAddr
					lock.Unlock()
					return nil
				}
			}, strategy.Wait(500*time.Millisecond),
			); err != nil {
				bxh.logger.WithFields(logrus.Fields{
					"pid": pid,
					"err": err.Error(),
				}).Warnf("retry error")
			}

			wg.Done()
		}(pid, result, &wg, &lock)
	}

	wg.Wait()

	return result
}
