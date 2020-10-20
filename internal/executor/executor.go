package executor

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
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
	ledger            ledger.Ledger
	logger            logrus.FieldLogger
	blockC            chan *pb.Block
	preBlockC         chan *pb.Block
	persistC          chan *ledger.BlockData
	interchainCounter map[string][]uint64
	normalTxs         []types.Hash
	ibtpVerify        proof.Verify
	validationEngine  validator.Engine
	currentHeight     uint64
	currentBlockHash  types.Hash
	boltContracts     map[string]boltvm.Contract
	wasmInstances     map[string]wasmer.Instance

	blockFeed event.Feed

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates executor instance
func New(chainLedger ledger.Ledger, logger logrus.FieldLogger) (*BlockExecutor, error) {
	ibtpVerify := proof.New(chainLedger, logger)

	boltContracts := registerBoltContracts()

	ctx, cancel := context.WithCancel(context.Background())

	return &BlockExecutor{
		ledger:            chainLedger,
		logger:            logger,
		interchainCounter: make(map[string][]uint64),
		ctx:               ctx,
		cancel:            cancel,
		blockC:            make(chan *pb.Block, blockChanNumber),
		preBlockC:         make(chan *pb.Block, blockChanNumber),
		persistC:          make(chan *ledger.BlockData, persistChanNumber),
		ibtpVerify:        ibtpVerify,
		validationEngine:  ibtpVerify.ValidationEngine(),
		currentHeight:     chainLedger.GetChainMeta().Height,
		currentBlockHash:  chainLedger.GetChainMeta().BlockHash,
		boltContracts:     boltContracts,
		wasmInstances:     make(map[string]wasmer.Instance),
	}, nil
}

// Start starts executor
func (exec *BlockExecutor) Start() error {
	go exec.listenExecuteEvent()

	go exec.listenPreExecuteEvent()

	go exec.persistData()

	exec.logger.WithFields(logrus.Fields{
		"height": exec.currentHeight,
		"hash":   exec.currentBlockHash.ShortString(),
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
func (exec *BlockExecutor) ExecuteBlock(block *pb.Block) {
	exec.preBlockC <- block
}

func (exec *BlockExecutor) SyncExecuteBlock(block *pb.Block) {
	exec.processExecuteEvent(block)
}

// SubscribeBlockEvent registers a subscription of NewBlockEvent.
func (exec *BlockExecutor) SubscribeBlockEvent(ch chan<- events.NewBlockEvent) event.Subscription {
	return exec.blockFeed.Subscribe(ch)
}

func (exec *BlockExecutor) ApplyReadonlyTransactions(txs []*pb.Transaction) []*pb.Receipt {
	current := time.Now()
	receipts := make([]*pb.Receipt, 0, len(txs))

	for i, tx := range txs {
		receipt := &pb.Receipt{
			Version: tx.Version,
			TxHash:  tx.TransactionHash,
		}

		ret, err := exec.applyTransaction(i, tx)
		if err != nil {
			receipt.Status = pb.Receipt_FAILED
			receipt.Ret = []byte(err.Error())
		} else {
			receipt.Status = pb.Receipt_SUCCESS
			receipt.Ret = ret
		}

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
		case block := <-exec.blockC:
			now := time.Now()
			blockData := exec.processExecuteEvent(block)
			exec.logger.WithFields(logrus.Fields{
				"height": block.BlockHeader.Number,
				"count":  len(block.Transactions),
				"elapse": time.Since(now),
			}).Infof("Executed block")
			exec.persistC <- blockData
		case <-exec.ctx.Done():
			return
		}
	}
}

func (exec *BlockExecutor) verifyProofs(block *pb.Block) *pb.Block {
	if block.BlockHeader.Number == 1 {
		return block
	}
	if block.Extra != nil {
		block.Extra = nil
		return block
	}

	var (
		index []int
		wg    sync.WaitGroup
		lock  sync.Mutex
	)
	txs := block.Transactions

	wg.Add(len(txs))
	for i, tx := range txs {
		go func(i int, tx *pb.Transaction) {
			defer wg.Done()
			ok, _ := exec.ibtpVerify.CheckProof(tx)
			if !ok {
				lock.Lock()
				defer lock.Unlock()
				index = append(index, i)
			}
		}(i, tx)
	}
	wg.Wait()

	if len(index) > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(index)))
		for _, idx := range index {
			txs = append(txs[:idx], txs[idx+1:]...)
		}
		block.Transactions = txs
	}

	return block
}

func (exec *BlockExecutor) persistData() {
	for data := range exec.persistC {
		now := time.Now()
		exec.ledger.PersistBlockData(data)
		exec.postBlockEvent(data.Block, data.InterchainMeta)
		exec.logger.WithFields(logrus.Fields{
			"height": data.Block.BlockHeader.Number,
			"hash":   data.Block.BlockHash.ShortString(),
			"count":  len(data.Block.Transactions),
			"elapse": time.Since(now),
		}).Info("Persisted block")
	}
}

func registerBoltContracts() map[string]boltvm.Contract {
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
	}

	return boltvm.Register(boltContracts)
}
