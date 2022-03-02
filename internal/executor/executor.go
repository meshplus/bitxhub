package executor

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/executor/oracle/appchain"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/proof"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	vm "github.com/meshplus/eth-kit/evm"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

const (
	blockChanNumber   = 1024
	persistChanNumber = 1024
)

var _ Executor = (*BlockExecutor)(nil)

// BlockExecutor executes block from order
type BlockExecutor struct {
	client           *appchain.Client
	ledger           *ledger.Ledger
	logger           logrus.FieldLogger
	blockC           chan *BlockWrapper
	preBlockC        chan *pb.CommitEvent
	persistC         chan *ledger.BlockData
	ibtpVerify       proof.Verify
	validationEngine validator.Engine
	currentHeight    uint64
	currentBlockHash *types.Hash
	txsExecutor      agency.TxsExecutor
	blockFeed        event.Feed
	logsFeed         event.Feed
	nodeFeed         event.Feed
	auditFeed        event.Feed
	ctx              context.Context
	cancel           context.CancelFunc

	evm         *vm.EVM
	evmChainCfg *params.ChainConfig
	gasLimit    uint64
	config      repo.Config
	bxhGasPrice *big.Int
	lock        *sync.Mutex
	admins      []string
}

func (exec *BlockExecutor) GetBoltContracts() map[string]agency.Contract {
	return exec.txsExecutor.GetBoltContracts()
}

// New creates executor instance
func New(chainLedger *ledger.Ledger, logger logrus.FieldLogger, client *appchain.Client, config *repo.Config, gasPrice *big.Int) (*BlockExecutor, error) {
	ibtpVerify := proof.New(chainLedger, logger, config.ChainID, config.GasLimit)

	txsExecutor, err := agency.GetExecutorConstructor(config.Executor.Type)
	if err != nil {
		return nil, fmt.Errorf("get executor constructor failed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	blockExecutor := &BlockExecutor{
		client:           client,
		ledger:           chainLedger,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		blockC:           make(chan *BlockWrapper, blockChanNumber),
		preBlockC:        make(chan *pb.CommitEvent, blockChanNumber),
		persistC:         make(chan *ledger.BlockData, persistChanNumber),
		ibtpVerify:       ibtpVerify,
		validationEngine: ibtpVerify.ValidationEngine(),
		currentHeight:    chainLedger.GetChainMeta().Height,
		currentBlockHash: chainLedger.GetChainMeta().BlockHash,
		evmChainCfg:      newEVMChainCfg(config),
		config:           *config,
		bxhGasPrice:      gasPrice,
		gasLimit:         config.GasLimit,
		lock:             &sync.Mutex{},
	}

	for _, admin := range config.Genesis.Admins {
		blockExecutor.admins = append(blockExecutor.admins, admin.Address)
	}

	blockExecutor.evm = newEvm(1, uint64(0), blockExecutor.evmChainCfg, blockExecutor.ledger, blockExecutor.ledger.ChainLedger, blockExecutor.admins[0])

	blockExecutor.txsExecutor = txsExecutor(blockExecutor.applyTx, blockExecutor.registerBoltContracts, logger)

	return blockExecutor, nil
}

// Start starts executor
func (exec *BlockExecutor) Start() error {
	go exec.listenExecuteEvent()

	go exec.listenPreExecuteEvent()

	go exec.persistData()

	exec.logger.WithFields(logrus.Fields{
		"height": exec.currentHeight,
		"hash":   exec.currentBlockHash.String(),
		"desc":   exec.txsExecutor.GetDescription(),
	}).Infof("BlockExecutor started")

	return nil
}

// Stop stops executor
func (exec *BlockExecutor) Stop() error {
	exec.cancel()

	exec.logger.Info("BlockExecutor stopped")

	return nil
}

// ExecuteBlock executes block from order
func (exec *BlockExecutor) ExecuteBlock(block *pb.CommitEvent) {
	exec.preBlockC <- block
}

// SubscribeBlockEvent registers a subscription of ExecutedEvent.
func (exec *BlockExecutor) SubscribeBlockEvent(ch chan<- events.ExecutedEvent) event.Subscription {
	return exec.blockFeed.Subscribe(ch)
}

func (exec *BlockExecutor) SubscribeLogsEvent(ch chan<- []*pb.EvmLog) event.Subscription {
	return exec.logsFeed.Subscribe(ch)
}

func (exec *BlockExecutor) SubscribeNodeEvent(ch chan<- events.NodeEvent) event.Subscription {
	return exec.nodeFeed.Subscribe(ch)
}

func (exec *BlockExecutor) SubscribeAuditEvent(ch chan<- *pb.AuditTxInfo) event.Subscription {
	return exec.auditFeed.Subscribe(ch)
}

func (exec *BlockExecutor) ApplyReadonlyTransactions(txs []pb.Transaction) []*pb.Receipt {
	current := time.Now()
	receipts := make([]*pb.Receipt, 0, len(txs))

	exec.lock.Lock()
	defer exec.lock.Unlock()

	meta := exec.ledger.GetChainMeta()
	block, err := exec.ledger.GetBlock(meta.Height)
	if err != nil {
		exec.logger.Errorf("fail to get block at %d: %v", meta.Height, err.Error())
		return nil
	}

	switch sl := exec.ledger.StateLedger.(type) {
	case *ledger2.ComplexStateLedger:
		newSl, err := sl.StateAt(block.BlockHeader.StateRoot)
		if err != nil {
			exec.logger.Errorf("fail to new state ledger at %s: %v", meta.BlockHash.String(), err.Error())
			return nil
		}
		exec.ledger.StateLedger = newSl
	}

	exec.ledger.PrepareBlock(meta.BlockHash, meta.Height)
	exec.evm = newEvm(meta.Height, uint64(block.BlockHeader.Timestamp), exec.evmChainCfg, exec.ledger.StateLedger, exec.ledger.ChainLedger, exec.admins[0])
	for i, tx := range txs {
		receipt := exec.applyTransaction(i, tx, "", nil)

		receipts = append(receipts, receipt)
		// clear potential write to ledger
		exec.ledger.Clear()
	}

	exec.logger.WithFields(logrus.Fields{
		"time":  time.Since(current),
		"count": len(txs),
	}).Debug("Apply readonly transactions elapsed")

	return receipts
}

func (exec *BlockExecutor) listenExecuteEvent() {
	for {
		select {
		case blockWrapper := <-exec.blockC:
			exec.processExecuteEvent(blockWrapper)
		case <-exec.ctx.Done():
			close(exec.persistC)
			return
		}
	}
}

func (exec *BlockExecutor) verifyProofs(blockWrapper *BlockWrapper) {
	block := blockWrapper.block

	if block.BlockHeader.Number == 1 {
		return
	}
	if block.Extra != nil {
		block.Extra = nil
		return
	}

	var (
		invalidTxs = make([]int, 0)
		// wg         sync.WaitGroup
		lock sync.Mutex
	)
	txs := block.Transactions.Transactions

	// wg.Add(len(txs))
	errM := make(map[int]string)
	for i, tx := range txs {
		// go func(i int, tx pb.Transaction) {
		// 	defer wg.Done()
		if _, ok := blockWrapper.invalidTx[i]; !ok {
			ok, gasUsed, err := exec.ibtpVerify.CheckProof(tx)
			if !ok {
				lock.Lock()
				defer lock.Unlock()
				invalidTxs = append(invalidTxs, i)
				errM[i] = err.Error()
			}
			exec.logger.WithField("gasUsed", gasUsed).Debug("Verify proofs")
			// if err := exec.payGasFee(tx, gasUsed); err != nil {
			// 	lock.Lock()
			// 	defer lock.Unlock()
			// 	invalidTxs = append(invalidTxs, i)
			// 	errM[i] = "run out of gas"
			// }
		}
		// }(i, tx)
	}
	// wg.Wait()

	for _, i := range invalidTxs {
		blockWrapper.invalidTx[i] = agency.InvalidReason(errM[i])
	}
}

func (exec *BlockExecutor) persistData() {
	for data := range exec.persistC {
		now := time.Now()
		exec.ledger.PersistBlockData(data)
		exec.postBlockEvent(data.Block, data.InterchainMeta, data.TxHashList)
		exec.postLogsEvent(data.Receipts)
		exec.logger.WithFields(logrus.Fields{
			"height": data.Block.BlockHeader.Number,
			"hash":   data.Block.BlockHash.String(),
			"count":  len(data.Block.Transactions.Transactions),
			"elapse": time.Since(now),
		}).Info("Persisted block")
	}
	exec.ledger.Close()
}

func (exec *BlockExecutor) registerBoltContracts() map[string]agency.Contract {
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
			Enabled:  exec.config.Appchain.Enable,
			Name:     "ethereum header service",
			Address:  constant.EthHeaderMgrContractAddr.Address().String(),
			Contract: contracts.NewEthHeaderManager(exec.client.EthOracle),
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
		{
			Enabled:  true,
			Name:     "proposal strategy manager service",
			Address:  constant.ProposalStrategyMgrContractAddr.Address().String(),
			Contract: &contracts.GovStrategy{},
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

func newEVMChainCfg(config *repo.Config) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:             big.NewInt(int64(config.ChainID)),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		// MuirGlacierBlock:    big.NewInt(0),
		// BerlinBlock:         big.NewInt(0),
	}
}
