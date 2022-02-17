package main

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"math/rand"
	"path/filepath"
	"runtime"
	"time"

	"github.com/meshplus/bitxhub-core/agency"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	service_mgr "github.com/meshplus/bitxhub-core/service-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/executor/oracle/appchain"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/genesis"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/profile"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/internal/storages"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

//var storeLogger = log.NewWithModule("cmd")
const TIMEOUT_HEIGHT = 5

func executeCMD() cli.Command {
	return cli.Command{
		Name:  "executor",
		Usage: "Start a executor test",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:     "passwd",
				Usage:    "bitxhub key password",
				Required: false,
			},
			cli.StringFlag{
				Name:  "config",
				Usage: "bitxhub config path",
			},
			cli.StringFlag{
				Name:  "network",
				Usage: "bitxhub network config path",
			},
			cli.StringFlag{
				Name:  "order",
				Usage: "bitxhub order config path",
			},
			cli.IntFlag{
				Name:  "interchainTxNum",
				Usage: "the number of txs in a block",
				Value: 1000,
			},
			cli.IntFlag{
				Name:  "normalTxNum",
				Usage: "the number of txs in a block",
				Value: 0,
			},
			cli.IntFlag{
				Name:  "fromNum",
				Usage: "the number of from in a block",
				Value: 10,
			},
			cli.IntFlag{
				Name:  "duration",
				Usage: "program run time, ms",
				Value: 10000,
			},
			cli.StringFlag{
				Name:  "randFrom",
				Usage: "whether the from addrs are randomly distributed",
				Value: "false",
			},
			//normalGatherN
			cli.IntFlag{
				Name:  "normalGatherN",
				Usage: "normal tx gather num",
				Value: 1,
			},
			cli.StringFlag{
				Name:  "normalTxType",
				Usage: "normal tx type, transfer or bvm",
				Value: "bvm",
			},
		},
		Action: testExecutor,
	}
}

func testExecutor(ctx *cli.Context) error {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return fmt.Errorf("get repo path: %w", err)
	}
	passwd := ctx.String("passwd")
	configPath := ctx.String("config")
	networkPath := ctx.String("network")
	interchainTxNum := ctx.Int("interchainTxNum")
	normalTxNum := ctx.Int("normalTxNum")
	fromNum := ctx.Int("fromNum")
	duration := ctx.Int("duration")
	randFrom := ctx.String("randFrom")
	normalGatherN := ctx.Int("normalGatherN")
	normalTxType := ctx.String("normalTxType")

	repo1, err := repo.Load(repoRoot, passwd, configPath, networkPath)
	if err != nil {
		return fmt.Errorf("repo load: %w", err)
	}

	// 1. init logger
	err = log.Initialize(
		log.WithReportCaller(repo1.Config.Log.ReportCaller),
		log.WithPersist(true),
		log.WithFilePath(filepath.Join(repoRoot, repo1.Config.Log.Dir)),
		log.WithFileName(repo1.Config.Log.Filename),
		log.WithMaxAge(90*24*time.Hour),
		log.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("log initialize: %w", err)
	}

	loggers.Initialize(repo1.Config)

	// 2. init storage =========================================================================
	if err := storages.Initialize(repoRoot); err != nil {
		fmt.Println(fmt.Errorf("storages initialize: %w", err))
		return fmt.Errorf("storages initialize: %w", err)
	}

	bcStorage, err := storages.Get(storages.BlockChain)
	if err != nil {
		fmt.Println(fmt.Errorf("create blockchain storage: %w", err))
		return fmt.Errorf("create blockchain storage: %w", err)
	}

	stateStorage, err := ledger.OpenStateDB(repo.GetStoragePath(repoRoot, "ledger"), repo1.Config.Ledger.Type)
	if err != nil {
		fmt.Println(fmt.Errorf("create tm-leveldb: %w", err))
		return fmt.Errorf("create tm-leveldb: %w", err)
	}

	bf, err := blockfile.NewBlockFile(repoRoot, loggers.Logger(loggers.Storage))
	if err != nil {
		fmt.Println(fmt.Errorf("blockfile initialize: %w", err))
		return fmt.Errorf("blockfile initialize: %w", err)
	}

	rwLdg, err := ledger.New(repo1, bcStorage, stateStorage, bf, nil, loggers.Logger(loggers.Executor))
	if err != nil {
		fmt.Println(fmt.Errorf("create RW ledger: %w", err))
		return fmt.Errorf("create RW ledger: %w", err)
	}

	viewLdg := &ledger.Ledger{
		ChainLedger: rwLdg.ChainLedger,
	}
	viewLdg.StateLedger, err = ledger.NewSimpleLedger(repo1, stateStorage.(storage.Storage), nil, loggers.Logger(loggers.Executor))
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 3. init exec
	execLogger := loggers.Logger(loggers.Executor)

	appchainClient := &appchain.Client{}
	if repo1.Config.Appchain.Enable {
		appchainClient, err = appchain.NewAppchainClient(filepath.Join(repoRoot, repo1.Config.Appchain.EthHeaderPath), repo.GetStoragePath(repoRoot, "appchain_client"), loggers.Logger(loggers.Executor))
		if err != nil {
			return fmt.Errorf("initialize appchain client failed: %w", err)
		}
	}

	blockExec, err := executor.New(rwLdg, execLogger, appchainClient, repo1.Config, big.NewInt(int64(repo1.Config.Genesis.BvmGasPrice)))
	if err != nil {
		fmt.Println(fmt.Errorf("create BlockExecutor: %w", err))
		return fmt.Errorf("create BlockExecutor: %w", err)
	}

	// 4. prepare =================================================================================
	// start proof
	pprof, err := profile.NewPprof(repo1.Config)
	if err != nil {
		return err
	}
	if err := pprof.Start(); err != nil {
		return err
	}

	// init info
	printVersion()
	execLogger.WithFields(logrus.Fields{
		"num": fromNum,
	}).Info("init several addresses...")
	addresses, err := initSeveralAddress(fromNum)
	if err != nil {
		fmt.Println(err)
		return err
	}
	execLogger.Infoln("init address map...")
	addressMap := initAddressMap(addresses)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// init storage
	if rwLdg.ChainLedger.GetChainMeta().Height == 0 {
		if err := genesis.Initialize(&repo1.Config.Genesis, repo1.NetworkConfig.Nodes, repo1.NetworkConfig.N, rwLdg, blockExec); err != nil {
			return err
		}
		logger.WithFields(logrus.Fields{
			"genesis block hash": rwLdg.ChainLedger.GetChainMeta().BlockHash,
			"block height":       rwLdg.ChainLedger.GetChainMeta().Height,
		}).Info("Initialize genesis")
	}

	height := 2
	// ======================================================================================== transfer
	txs := genTransferTransaction(repoRoot, addresses, execLogger)
	applyTransaction(blockExec, rwLdg, txs, addressMap, uint64(height), execLogger, false)
	height++

	// ======================================================================================== register
	txs1 := genRegisterTransactions(addresses)
	applyTransaction(blockExec, rwLdg, txs1, addressMap, uint64(height), execLogger, false)
	height++

	// ======================================================================================== test
	execLogger.WithFields(logrus.Fields{
		"interchainTxNum": interchainTxNum,
		"normalTxNum":     normalTxNum,
		"fromNum":         fromNum,
		"duration":        duration,
	}).Info("start exec test...")
	startTime := time.Now()
	for {
		//execLogger.Infoln("generate ibtp transactions...")
		interchainTxs := genInterchainTxs(addresses, addressMap, interchainTxNum, fromNum, randFrom)
		var normalTxs []pb.Transaction
		if normalTxType == "bvm" {
			normalTxs = genNormalBvmTxs(repoRoot, addresses, execLogger, normalTxNum, fromNum)
		} else {
			normalTxs = genNormalTxs(repoRoot, addresses, execLogger, normalTxNum, fromNum)
		}
		txs2 := mergeTxs(interchainTxs, normalTxs, normalGatherN)
		applyTransaction(blockExec, rwLdg, txs2, addressMap, uint64(height), execLogger, true)
		height++
		if time.Since(startTime).Milliseconds() >= int64(duration) {
			break
		}
	}
	execLogger.Infoln("end block", height)

	return nil
}

func mergeTxs(interchainTxs []pb.Transaction, normalTxs []pb.Transaction, normalGatherN int) []pb.Transaction {
	interchainN := len(interchainTxs)
	normalN := len(normalTxs)
	secN := normalN / normalGatherN
	if secN == 0 {
		secN = 1
	}
	interchainGatherN := interchainN / secN

	var txs []pb.Transaction
	interchainJ := 0
	normalJ := 0
	for i := 0; i < secN; i++ {
		for j := 0; j < interchainGatherN; j++ {
			if interchainJ >= interchainN {
				break
			}
			txs = append(txs, interchainTxs[interchainJ])
			interchainJ++
		}
		for j := 0; j < normalGatherN; j++ {
			if normalJ >= normalN {
				break
			}
			txs = append(txs, normalTxs[normalJ])
			normalJ++
		}
	}
	for ; interchainJ < interchainN; interchainJ++ {
		txs = append(txs, interchainTxs[interchainJ])
	}
	for ; normalJ < normalN; normalJ++ {
		txs = append(txs, normalTxs[normalJ])
	}

	return txs
}

func initSeveralAddress(num int) ([]*types.Address, error) {
	var address []*types.Address
	for i := 0; i < num; i++ {
		privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}
		newAddress, err := privKey.PublicKey().Address()
		if err != nil {
			return nil, fmt.Errorf("generate address: %w", err)
		}
		address = append(address, newAddress)
	}

	return address, nil
}

func initAddressMap(addresses []*types.Address) map[string]uint64 {
	addressMap := make(map[string]uint64)
	for _, addr := range addresses {
		addressMap[addr.String()] = 1
	}

	return addressMap
}

func genTransferTransaction(repoRoot string, addresses []*types.Address, logger logrus.FieldLogger) []pb.Transaction {
	keyPath1 := filepath.Join(repoRoot, "key.json")
	priAdmin1, _ := asym.RestorePrivateKey(keyPath1, "bitxhub")
	fromAdmin1, _ := priAdmin1.PublicKey().Address()
	//adminNonce1 := api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	td := &pb.TransactionData{
		Type:   pb.TransactionData_NORMAL,
		Amount: "100000000000000000000000000000",
	}

	payload, _ := td.Marshal()

	var txs []pb.Transaction
	for _, addr := range addresses {
		tx := &pb.BxhTransaction{
			From:      fromAdmin1,
			To:        addr,
			Payload:   payload,
			Timestamp: time.Now().UnixNano(),
			//Nonce:     nonce,
		}
		tx.Sign(priAdmin1)

		tx.TransactionHash = tx.Hash()

		txs = append(txs, tx)
		logger.WithFields(logrus.Fields{
			"to":     addr,
			"amount": td.Amount,
		}).Info("transfer...")
	}

	return txs
}

func genRegisterTransactions(addresses []*types.Address) []pb.Transaction {
	var txs []pb.Transaction
	i := 0
	for _, addr := range addresses {
		// register appchain
		args := []*pb.Arg{
			pb.String(fmt.Sprintf("appchain%d", i)),
			pb.String(fmt.Sprintf("应用链%s", addr)),
			pb.String(appchainMgr.ChainTypeETH),
			pb.Bytes(nil),
			pb.String("broker"),
			pb.String("desc"),
			pb.String(validator.HappyRuleAddr),
			pb.String("url"),
			pb.String(addr.String()),
			pb.String("reason"),
		}
		invokePayload := &pb.InvokePayload{
			Method: "RegisterAppchain",
			Args:   args,
		}
		payload, _ := invokePayload.Marshal()
		txData := &pb.TransactionData{
			Type:    pb.TransactionData_INVOKE,
			Amount:  "",
			VmType:  pb.TransactionData_BVM,
			Payload: payload,
			Extra:   nil,
		}
		data, _ := txData.Marshal()
		hash := sha256.Sum256([]byte(randString(20)))
		tx := &pb.BxhTransaction{
			From:            addr,
			To:              constant.AppchainMgrContractAddr.Address(),
			TransactionHash: types.NewHash(hash[:]),
			Payload:         data,
		}
		txs = append(txs, tx)

		// register service
		args = []*pb.Arg{
			pb.String(fmt.Sprintf("appchain%d", i)),
			pb.String("serviceA"),
			pb.String(fmt.Sprintf("服务%s", addr)),
			pb.String(string(service_mgr.ServiceCallContract)),
			pb.String("intro"),
			pb.Uint64(1),
			pb.String(""),
			pb.String("details"),
			pb.String("raeson"),
		}
		invokePayload = &pb.InvokePayload{
			Method: "RegisterService",
			Args:   args,
		}
		payload, _ = invokePayload.Marshal()
		txData = &pb.TransactionData{
			Type:    pb.TransactionData_INVOKE,
			Amount:  "",
			VmType:  pb.TransactionData_BVM,
			Payload: payload,
			Extra:   nil,
		}
		data, _ = txData.Marshal()
		hash = sha256.Sum256([]byte(randString(20)))
		tx1 := &pb.BxhTransaction{
			From:            addr,
			To:              constant.ServiceMgrContractAddr.Address(),
			TransactionHash: types.NewHash(hash[:]),
			Payload:         data,
		}
		txs = append(txs, tx1)

		i++
	}
	return txs
}

func genInterchainTxs(addresses []*types.Address, addressMap map[string]uint64, txNum, fromNum int, randFrom string) []pb.Transaction {
	var txs []pb.Transaction
	content := &pb.Content{
		Func: "interchainCharge",
		Args: [][]byte{[]byte("Alice"), []byte("Alice"), []byte("1")},
	}
	bytes, _ := content.Marshal()
	payload := &pb.Payload{
		Encrypted: false,
		Content:   bytes,
	}
	ibtppd, _ := payload.Marshal()
	proof := []byte("")
	orderIndex := 0
	for i := 0; i < txNum; i++ {
		orderIndex++
		fromIndex := orderIndex % fromNum
		if randFrom == "true" {
			rand.Seed(time.Now().UnixNano())
			fromIndex = rand.Intn(fromNum)
		}
		hash := sha256.Sum256([]byte(randString(20)))
		tx := &pb.BxhTransaction{
			From:            addresses[fromIndex],
			To:              constant.InterchainContractAddr.Address(),
			TransactionHash: types.NewHash(hash[:]),
			IBTP: &pb.IBTP{
				From:          fmt.Sprintf("%s:%s:%s", "1356", fmt.Sprintf("appchain%d", fromIndex), "serviceA"),
				To:            fmt.Sprintf("%s:%s:%s", "1356", fmt.Sprintf("appchain%d", (fromIndex+1)%fromNum), "serviceA"),
				Index:         addressMap[addresses[fromIndex].String()],
				TimeoutHeight: TIMEOUT_HEIGHT,
				Payload:       ibtppd,
				Proof:         proof,
			},
		}
		txs = append(txs, tx)
		addressMap[addresses[fromIndex].String()] = addressMap[addresses[fromIndex].String()] + 1
	}
	return txs
}

func genNormalTxs(repoRoot string, addresses []*types.Address, logger logrus.FieldLogger, txNum, fromNum int) []pb.Transaction {
	keyPath1 := filepath.Join(repoRoot, "key.json")
	priAdmin1, _ := asym.RestorePrivateKey(keyPath1, "bitxhub")
	fromAdmin1, _ := priAdmin1.PublicKey().Address()
	//adminNonce1 := api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	td := &pb.TransactionData{
		Type:   pb.TransactionData_NORMAL,
		Amount: "1",
	}

	payload, _ := td.Marshal()

	var txs []pb.Transaction
	for i := 0; i < txNum; i++ {
		rand.Seed(time.Now().UnixNano())
		randIndex := rand.Intn(fromNum)
		tx := &pb.BxhTransaction{
			From:      fromAdmin1,
			To:        addresses[randIndex],
			Payload:   payload,
			Timestamp: time.Now().UnixNano(),
			//Nonce:     nonce,
		}
		tx.Sign(priAdmin1)

		tx.TransactionHash = tx.Hash()

		txs = append(txs, tx)
	}

	return txs
}

func genNormalBvmTxs(repoRoot string, addresses []*types.Address, logger logrus.FieldLogger, txNum, fromNum int) []pb.Transaction {
	//keyPath1 := filepath.Join(repoRoot, "key.json")
	//priAdmin1, _ := asym.RestorePrivateKey(keyPath1, "bitxhub")
	//fromAdmin1, _ := priAdmin1.PublicKey().Address()
	//adminNonce1 := api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	var txs []pb.Transaction
	for i := 0; i < txNum; i++ {
		rand.Seed(time.Now().UnixNano())
		randIndex := rand.Intn(fromNum)

		args := []*pb.Arg{
			pb.String(fmt.Sprintf("appchain%d", randIndex)),
			pb.String(fmt.Sprintf("应用链%s", addresses[randIndex])),
			pb.String(fmt.Sprintf("desc%s", time.Now().String())),
			pb.Bytes(nil),
			pb.String(addresses[randIndex].String()),
			pb.String("reason"),
		}
		invokePayload := &pb.InvokePayload{
			Method: "UpdateAppchain",
			Args:   args,
		}
		invokePayloadData, _ := invokePayload.Marshal()

		td := &pb.TransactionData{
			Type:    pb.TransactionData_INVOKE,
			VmType:  pb.TransactionData_BVM,
			Payload: invokePayloadData,
		}

		payload, _ := td.Marshal()

		tx := &pb.BxhTransaction{
			From:      addresses[randIndex],
			To:        constant.AppchainMgrContractAddr.Address(),
			Payload:   payload,
			Timestamp: time.Now().UnixNano(),
			//Nonce:     nonce,
		}
		//tx.Sign(priAdmin1)

		tx.TransactionHash = tx.Hash()

		txs = append(txs, tx)
	}

	return txs
}

func applyTransaction(blockExec *executor.BlockExecutor, rwLdg *ledger.Ledger, txs []pb.Transaction, addressMap map[string]uint64, height uint64, execLogger logrus.FieldLogger, isInterchain bool) {
	txsExec := blockExec.GetTxsExecutor()
	execLogger.Debugln("begin block", height)
	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number:      uint64(height),
			StateRoot:   types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"),
			TxRoot:      types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"),
			ReceiptRoot: types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"),
			ParentHash:  types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"),
			Timestamp:   time.Now().UnixNano(),
		},
		Transactions: &pb.Transactions{
			Transactions: txs,
		},
		BlockHash: types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"),
		Signature: []byte("111223123123213211121312312"),
		Extra:     []byte(""),
	}

	time1 := time.Now()
	receipts := txsExec.ApplyTransactions(block.Transactions.Transactions, make(map[int]agency.InvalidReason))
	time2 := time.Now()
	exeTime := time2.Sub(time1)
	fmt.Println(exeTime.Milliseconds())
	for _, ret := range receipts {
		if !ret.IsSuccess() {
			execLogger.WithFields(logrus.Fields{
				"ret": string(ret.Ret),
			}).Error("apply tx error...")
		}
	}

	interchainMeta := &pb.InterchainMeta{}
	if isInterchain {
		counter := make(map[string]*pb.VerifiedIndexSlice)
		for k, _ := range addressMap {
			var v []*pb.VerifiedIndex
			for i := 0; i < 5; i++ {
				v = append(v, &pb.VerifiedIndex{
					Index: uint64(i),
					Valid: true,
				})
			}
			counter[k] = &pb.VerifiedIndexSlice{Slice: v}
		}
		var l2Roots []types.Hash
		for i := 0; i < 200; i++ {
			l2Roots = append(l2Roots, *types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"))
		}
		var timeoutL2Roots []types.Hash
		for i := 0; i < 2; i++ {
			timeoutL2Roots = append(timeoutL2Roots, *types.NewHashByStr("0xc30B6E0ad5327fc8548f4BaFab3271cA6a5bD92f084095958c84970165bfA6E7"))
		}
		interchainMeta = &pb.InterchainMeta{
			Counter:        counter,
			L2Roots:        l2Roots,
			TimeoutCounter: make(map[string]*pb.StringSlice),
			TimeoutL2Roots: timeoutL2Roots,
			MultiTxCounter: make(map[string]*pb.StringSlice),
		}
	}

	var txHashList []*types.Hash
	for _, tx := range txs {
		txHashList = append(txHashList, tx.GetHash())
	}

	//rwLdg.SetState(constant.TransactionMgrContractAddr.Address(), []byte(contracts.TimeoutKey(height+TIMEOUT_HEIGHT-1)), []byte(""))
	accounts, journalHash := rwLdg.FlushDirtyData()
	data := &ledger.BlockData{
		Block:          block,
		Receipts:       receipts,
		Accounts:       accounts,
		InterchainMeta: interchainMeta,
		TxHashList:     txHashList,
	}
	data.Block.BlockHeader.StateRoot = journalHash
	rwLdg.PersistBlockData(data)

	blockExec.CurrentHeight = block.BlockHeader.Number
	blockExec.CurrentBlockHash = block.BlockHash
	accounts = nil
	rwLdg.Clear()
	runtime.GC()
}

func randString(len int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}

func registerBoltContracts() map[string]agency.Contract {
	boltContracts := []*boltvm.BoltContract{
		{
			Enabled:  true,
			Name:     "interchain manager contract",
			Address:  constant.InterchainContractAddr.Address().String(),
			Contract: &contracts.InterchainManager{},
		},
		{
			Enabled:  true,
			Name:     "store service",
			Address:  constant.StoreContractAddr.Address().String(),
			Contract: &contracts.Store{},
		},
		{
			Enabled:  true,
			Name:     "rule manager service",
			Address:  constant.RuleManagerContractAddr.Address().String(),
			Contract: &contracts.RuleManager{},
		},
		{
			Enabled:  true,
			Name:     "role manager service",
			Address:  constant.RoleContractAddr.Address().String(),
			Contract: &contracts.RoleManager{},
		},
		{
			Enabled:  true,
			Name:     "appchain manager service",
			Address:  constant.AppchainMgrContractAddr.Address().String(),
			Contract: &contracts.AppchainManager{},
		},
		{
			Enabled:  true,
			Name:     "transaction manager service",
			Address:  constant.TransactionMgrContractAddr.Address().String(),
			Contract: &contracts.TransactionManager{},
		},
		{
			Enabled:  true,
			Name:     "governance service",
			Address:  constant.GovernanceContractAddr.Address().String(),
			Contract: &contracts.Governance{},
		},
		{
			Enabled:  true,
			Name:     "node manager service",
			Address:  constant.NodeManagerContractAddr.Address().String(),
			Contract: &contracts.NodeManager{},
		},
		{
			Enabled:  true,
			Name:     "inter broker service",
			Address:  constant.InterBrokerContractAddr.Address().String(),
			Contract: &contracts.InterBroker{},
		},
		{
			Enabled:  true,
			Name:     "service manager service",
			Address:  constant.ServiceMgrContractAddr.Address().String(),
			Contract: &contracts.ServiceManager{},
		},
		{
			Enabled:  true,
			Name:     "dapp manager service",
			Address:  constant.DappMgrContractAddr.Address().String(),
			Contract: &contracts.DappManager{},
		},
	}

	ContractsInfo := agency.GetRegisteredContractInfo()
	for addr, info := range ContractsInfo {
		boltContracts = append(boltContracts, &boltvm.BoltContract{
			Enabled:  true,
			Name:     info.Name,
			Address:  addr,
			Contract: info.Constructor(),
		})
	}

	return boltvm.Register(boltContracts)
}
