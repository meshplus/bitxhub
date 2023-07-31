package executor

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	vm "github.com/meshplus/eth-kit/evm"
	"github.com/sirupsen/logrus"
)

const (
	blockChanNumber   = 1024
	persistChanNumber = 1024
)

var _ Executor = (*BlockExecutor)(nil)

// BlockExecutor executes block from order
type BlockExecutor struct {
	ledger             *ledger.Ledger
	logger             logrus.FieldLogger
	blockC             chan *BlockWrapper
	preBlockC          chan *types.CommitEvent
	persistC           chan *ledger.BlockData
	currentHeight      uint64
	currentBlockHash   *types.Hash
	blockFeed          event.Feed
	blockFeedForRemote event.Feed
	logsFeed           event.Feed
	ctx                context.Context
	cancel             context.CancelFunc

	evm         *vm.EVM
	evmChainCfg *params.ChainConfig
	gasLimit    uint64
	config      repo.Config
	GasPrice    func() (*big.Int, error)
	lock        *sync.Mutex
	admins      []string
}

// New creates executor instance
func New(chainLedger *ledger.Ledger, logger logrus.FieldLogger, config *repo.Config, gasPrice func() (*big.Int, error)) (*BlockExecutor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	blockExecutor := &BlockExecutor{
		ledger:           chainLedger,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
		blockC:           make(chan *BlockWrapper, blockChanNumber),
		preBlockC:        make(chan *types.CommitEvent, blockChanNumber),
		persistC:         make(chan *ledger.BlockData, persistChanNumber),
		currentHeight:    chainLedger.GetChainMeta().Height,
		currentBlockHash: chainLedger.GetChainMeta().BlockHash,
		evmChainCfg:      newEVMChainCfg(config),
		config:           *config,
		GasPrice:         gasPrice,
		gasLimit:         config.Genesis.GasLimit,
		lock:             &sync.Mutex{},
	}

	for _, admin := range config.Genesis.Admins {
		blockExecutor.admins = append(blockExecutor.admins, admin.Address)
	}

	blockExecutor.evm = newEvm(1, uint64(0), blockExecutor.evmChainCfg, blockExecutor.ledger, blockExecutor.ledger.ChainLedger, blockExecutor.admins[0])

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
func (exec *BlockExecutor) ExecuteBlock(block *types.CommitEvent) {
	exec.preBlockC <- block
}

// SubscribeBlockEvent registers a subscription of ExecutedEvent.
func (exec *BlockExecutor) SubscribeBlockEvent(ch chan<- events.ExecutedEvent) event.Subscription {
	return exec.blockFeed.Subscribe(ch)
}

// SubscribeBlockEvent registers a subscription of ExecutedEvent.
func (exec *BlockExecutor) SubscribeBlockEventForRemote(ch chan<- events.ExecutedEvent) event.Subscription {
	return exec.blockFeedForRemote.Subscribe(ch)
}

func (exec *BlockExecutor) SubscribeLogsEvent(ch chan<- []*types.EvmLog) event.Subscription {
	return exec.logsFeed.Subscribe(ch)
}

func (exec *BlockExecutor) ApplyReadonlyTransactions(txs []*types.Transaction) []*types.Receipt {
	current := time.Now()
	receipts := make([]*types.Receipt, 0, len(txs))

	exec.lock.Lock()
	defer exec.lock.Unlock()

	meta := exec.ledger.GetChainMeta()
	block, err := exec.ledger.GetBlock(meta.Height)
	if err != nil {
		exec.logger.Errorf("fail to get block at %d: %v", meta.Height, err.Error())
		return nil
	}

	// switch sl := exec.ledger.StateLedger.(type) {
	// case *ledger2.ComplexStateLedger:
	// 	newSl, err := sl.StateAt(block.BlockHeader.StateRoot)
	// 	if err != nil {
	// 		exec.logger.Errorf("fail to new state ledger at %s: %v", meta.BlockHash.String(), err.Error())
	// 		return nil
	// 	}
	// 	exec.ledger.StateLedger = newSl
	// }

	exec.ledger.PrepareBlock(meta.BlockHash, meta.Height)
	exec.evm = newEvm(meta.Height, uint64(block.BlockHeader.Timestamp), exec.evmChainCfg, exec.ledger.StateLedger, exec.ledger.ChainLedger, exec.admins[0])
	for i, tx := range txs {
		exec.ledger.SetTxContext(tx.GetHash(), i)
		receipt := exec.applyTransaction(i, tx, "")

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

func (exec *BlockExecutor) persistData() {
	for data := range exec.persistC {
		now := time.Now()
		exec.ledger.PersistBlockData(data)
		exec.postBlockEvent(data.Block, data.TxHashList)
		exec.postLogsEvent(data.Receipts)
		exec.logger.WithFields(logrus.Fields{
			"height": data.Block.BlockHeader.Number,
			"hash":   data.Block.BlockHash.String(),
			"count":  len(data.Block.Transactions),
			"elapse": time.Since(now),
		}).Info("Persisted block")
	}
	exec.ledger.Close()
}

func newEVMChainCfg(config *repo.Config) *params.ChainConfig {
	shanghaiTime := uint64(0)
	CancunTime := uint64(0)
	PragueTime := uint64(0)

	return &params.ChainConfig{
		ChainID:                 big.NewInt(int64(config.Genesis.ChainID)),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		ShanghaiTime:            &shanghaiTime,
		CancunTime:              &CancunTime,
		PragueTime:              &PragueTime,
	}
}
