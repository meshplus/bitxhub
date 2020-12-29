package executor

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
)

type Executor interface {
	// Start
	Start() error

	// Stop
	Stop() error

	// ExecutorBlock
	ExecuteBlock(commitEvent *pb.CommitEvent)

	// ApplyReadonlyTransactions execute readonly tx
	ApplyReadonlyTransactions(txs []*pb.Transaction) []*pb.Receipt

	// SubscribeBlockEvent
	SubscribeBlockEvent(chan<- events.ExecutedEvent) event.Subscription
}
