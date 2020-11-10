package order

import (
	"context"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

//go:generate mockgen -destination mock_order/mock_order.go -package mock_order -source order.go
type Order interface {
	// Start the order service.
	Start() error

	// Stop means frees the resources which were allocated for this service.
	Stop()

	// Prepare means send transaction to the consensus engine
	Prepare(tx *pb.Transaction) error

	// Commit recv blocks form Order and commit it by order
	Commit() chan *pb.Block

	// Step send msg to the consensus engine
	Step(ctx context.Context, msg []byte) error

	// Ready means whether order has finished electing leader
	Ready() error

	// ReportState means block was persisted and report it to the consensus engine
	ReportState(height uint64, hash types.Hash)

	// Quorum means minimum number of nodes in the cluster that can work
	Quorum() uint64

	// GetPendingNonce will return the latest pending nonce of a given account
	GetPendingNonceByAccount(account string) uint64
}
