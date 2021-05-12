package executor

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-core/validator"
	vm "github.com/meshplus/bitxhub-kit/evm"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/proof"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/sirupsen/logrus"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

const (
	blockChanNumber   = 1024
	persistChanNumber = 1024
)

var _ Executor = (*BlockExecutor)(nil)

// BlockExecutor executes block from order
type BlockExecutor struct {
	ledger           ledger.Ledger
	logger           logrus.FieldLogger
	blockC           chan *BlockWrapper
	preBlockC        chan *pb.CommitEvent
	persistC         chan *ledger.BlockData
	ibtpVerify       proof.Verify
	validationEngine validator.Engine
	currentHeight    uint64
	currentBlockHash *types.Hash
	wasmInstances    map[string]wasmer.Instance
	txsExecutor      agency.TxsExecutor
	blockFeed        event.Feed
	logsFeed         event.Feed
	ctx              context.Context
	cancel           context.CancelFunc

	evm         *vm.EVM
	evmChainCfg *params.ChainConfig
	gasLimit    uint64
}

// New creates executor instance
func New(chainLedger ledger.Ledger, logger logrus.FieldLogger, typ string, gasLimit uint64) (*BlockExecutor, error) {
	ibtpVerify := proof.New(chainLedger, logger)

	txsExecutor, err := agency.GetExecutorConstructor(typ)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	blockExecutor := &BlockExecutor{
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
		wasmInstances:    make(map[string]wasmer.Instance),
		evmChainCfg:      newEVMChainCfg(),
		gasLimit:         gasLimit,
	}

	blockExecutor.evm = newEvm(1, uint64(0), blockExecutor.evmChainCfg, blockExecutor.ledger.StateDB())

	blockExecutor.txsExecutor = txsExecutor(blockExecutor.applyTx, registerBoltContracts, logger)

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

func (exec *BlockExecutor) ApplyReadonlyTransactions(txs []pb.Transaction) []*pb.Receipt {
	current := time.Now()
	receipts := make([]*pb.Receipt, 0, len(txs))

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
			now := time.Now()
			blockData := exec.processExecuteEvent(blockWrapper)
			exec.logger.WithFields(logrus.Fields{
				"height": blockWrapper.block.BlockHeader.Number,
				"count":  len(blockWrapper.block.Transactions.Transactions),
				"elapse": time.Since(now),
			}).Debug("Executed block")
			exec.persistC <- blockData
		case <-exec.ctx.Done():
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
		wg         sync.WaitGroup
		lock       sync.Mutex
	)
	txs := block.Transactions.Transactions

	wg.Add(len(txs))
	errM := make(map[int]string)
	for i, tx := range txs {
		go func(i int, tx pb.Transaction) {
			defer wg.Done()
			if _, ok := blockWrapper.invalidTx[i]; !ok {
				ok, err := exec.ibtpVerify.CheckProof(tx)
				if !ok {
					lock.Lock()
					defer lock.Unlock()
					invalidTxs = append(invalidTxs, i)
					errM[i] = err.Error()
				}
			}
		}(i, tx)
	}
	wg.Wait()

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
			Contract: &contracts.Role{},
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
			Name:     "asset exchange service",
			Address:  constant.AssetExchangeContractAddr.Address().String(),
			Contract: &contracts.AssetExchange{},
		},
		{
			Enabled:  true,
			Name:     "interchain broker service",
			Address:  constant.InterRelayBrokerContractAddr.Address().String(),
			Contract: &contracts.InterRelayBroker{},
		},
		{
			Enabled:  true,
			Name:     "governance service",
			Address:  constant.GovernanceContractAddr.Address().String(),
			Contract: &contracts.Governance{},
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

func newEVMChainCfg() *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:             big.NewInt(1),
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
