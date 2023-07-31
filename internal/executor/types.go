package executor

import (
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/model/events"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/ethereum/go-ethereum/event"
)

type Executor interface {
	// Start
	Start() error

	// Stop
	Stop() error

	// ExecutorBlock
	ExecuteBlock(commitEvent *types.CommitEvent)

	// ApplyReadonlyTransactions execute readonly tx
	ApplyReadonlyTransactions(txs []*types.Transaction) []*types.Receipt

	// SubscribeBlockEvent
	SubscribeBlockEvent(chan<- events.ExecutedEvent) event.Subscription

	// SubscribeBlockEventForRemote
	SubscribeBlockEventForRemote(chan<- events.ExecutedEvent) event.Subscription

	// SubscribeLogEvent
	SubscribeLogsEvent(chan<- []*types.EvmLog) event.Subscription

	GetEvm(txCtx vm.TxContext, vmConfig vm.Config) *vm.EVM
}
