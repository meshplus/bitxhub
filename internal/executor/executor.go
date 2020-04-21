package executor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/cache"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/vm/boltvm"
	"github.com/sirupsen/logrus"
)

const blockChanNumber = 1024

var _ Executor = (*BlockExecutor)(nil)

// BlockExecutor executes block from order
type BlockExecutor struct {
	ledger            ledger.Ledger
	logger            logrus.FieldLogger
	blockC            chan *pb.Block
	pendingBlockQ     *cache.Cache
	interchainCounter map[string][]uint64
	validationEngine  validator.Engine
	currentHeight     uint64
	currentBlockHash  types.Hash

	blockFeed event.Feed

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates executor instance
func New(ledger ledger.Ledger, logger logrus.FieldLogger) (*BlockExecutor, error) {
	pendingBlockQ, err := cache.NewCache()
	if err != nil {
		return nil, fmt.Errorf("create cache: %w", err)
	}

	ve := validator.NewValidationEngine(ledger, logger)

	registerBoltContracts()

	ctx, cancel := context.WithCancel(context.Background())

	return &BlockExecutor{
		ledger:            ledger,
		logger:            logger,
		interchainCounter: make(map[string][]uint64),
		ctx:               ctx,
		cancel:            cancel,
		blockC:            make(chan *pb.Block, blockChanNumber),
		pendingBlockQ:     pendingBlockQ,
		validationEngine:  ve,
		currentHeight:     ledger.GetChainMeta().Height,
		currentBlockHash:  ledger.GetChainMeta().BlockHash,
	}, nil
}

// Start starts executor
func (exec *BlockExecutor) Start() error {
	go exec.listenExecuteEvent()

	exec.logger.WithFields(logrus.Fields{
		"height": exec.currentHeight,
		"hash":   exec.currentBlockHash.ShortString(),
	}).Infof("Executor started")

	return nil
}

// Stop stops executor
func (exec *BlockExecutor) Stop() error {
	exec.cancel()

	exec.logger.Info("Executor stopped")

	return nil
}

// ExecuteBlock executes block from order
func (exec *BlockExecutor) ExecuteBlock(block *pb.Block) {
	exec.blockC <- block
}

func (exec *BlockExecutor) SyncExecuteBlock(block *pb.Block) {
	exec.handleExecuteEvent(block)
}

// SubscribeBlockEvent registers a subscription of NewBlockEvent.
func (exec *BlockExecutor) SubscribeBlockEvent(ch chan<- events.NewBlockEvent) event.Subscription {
	return exec.blockFeed.Subscribe(ch)
}

func (exec *BlockExecutor) listenExecuteEvent() {
	for {
		select {
		case block := <-exec.blockC:
			exec.handleExecuteEvent(block)
		case <-exec.ctx.Done():
			return
		}
	}
}

func registerBoltContracts() {
	boltContracts := []*boltvm.BoltContract{
		{
			Enabled:  true,
			Name:     "appchain manager contract",
			Address:  constant.InterchainContractAddr.String(),
			Contract: &contracts.Interchain{},
		},
		{
			Enabled:  true,
			Name:     "store service",
			Address:  constant.StoreContractAddr.String(),
			Contract: &contracts.Store{},
		},
		{
			Enabled:  true,
			Name:     "rule manager service",
			Address:  constant.RuleManagerContractAddr.String(),
			Contract: &contracts.RuleManager{},
		},
		{
			Enabled:  true,
			Name:     "role manager service",
			Address:  constant.RoleContractAddr.String(),
			Contract: &contracts.Role{},
		},
	}

	boltvm.Register(boltContracts)
}
