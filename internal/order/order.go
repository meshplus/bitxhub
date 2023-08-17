package order

import (
	"fmt"

	"github.com/ethereum/go-ethereum/event"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order/common"
	"github.com/axiomesh/axiom/internal/order/rbft"
	"github.com/axiomesh/axiom/internal/order/solo"
	"github.com/axiomesh/axiom/internal/order/solo_dev"
	"github.com/axiomesh/axiom/pkg/repo"
)

//go:generate mockgen -destination mock_order/mock_order.go -package mock_order -source order.go -typed
type Order interface {
	// Start the order service.
	Start() error

	// Stop means frees the resources which were allocated for this service.
	Stop()

	// Prepare means send transaction to the consensus engine
	Prepare(tx *types.Transaction) error

	// Commit recv blocks form Order and commit it by order
	Commit() chan *common.CommitEvent

	// Step send msg to the consensus engine
	Step(msg []byte) error

	// Ready means whether order has finished electing leader
	Ready() error

	// ReportState means block was persisted and report it to the consensus engine
	ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash)

	// Quorum means minimum number of nodes in the cluster that can work
	Quorum() uint64

	// GetPendingNonceByAccount will return the latest pending nonce of a given account
	GetPendingNonceByAccount(account string) uint64

	GetPendingTxByHash(hash *types.Hash) *types.Transaction

	SubscribeTxEvent(events chan<- []*types.Transaction) event.Subscription
}

func New(orderType string, opts ...common.Option) (Order, error) {
	config, err := common.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	// Get the order constructor according to different order type.
	switch orderType {
	case repo.OrderTypeSolo:
		return solo.NewNode(config)
	case repo.OrderTypeRbft:
		return rbft.NewNode(config)
	case repo.OrderTypeSoloDev:
		return solo_dev.NewNode(config)
	default:
		return nil, fmt.Errorf("unsupport order type: %s", orderType)
	}
}
