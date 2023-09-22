package consensus

import (
	"fmt"

	"github.com/ethereum/go-ethereum/event"

	"github.com/axiomesh/axiom-bft/common/consensus"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/consensus/common"
	"github.com/axiomesh/axiom-ledger/internal/consensus/rbft"
	"github.com/axiomesh/axiom-ledger/internal/consensus/solo"
	"github.com/axiomesh/axiom-ledger/internal/consensus/solo_dev"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

type Consensus interface {
	// Start the consensus service.
	Start() error

	// Stop means frees the resources which were allocated for this service.
	Stop()

	// Prepare means send transaction to the consensus engine
	Prepare(tx *types.Transaction) error

	// Commit recv blocks form Consensus and commit it by consensus
	Commit() chan *common.CommitEvent

	// Step send msg to the consensus engine
	Step(msg []byte) error

	// Ready means whether consensus has finished electing leader
	Ready() error

	// ReportState means block was persisted and report it to the consensus engine
	ReportState(height uint64, blockHash *types.Hash, txHashList []*types.Hash, stateUpdatedCheckpoint *consensus.Checkpoint)

	// Quorum means minimum number of nodes in the cluster that can work
	Quorum() uint64

	// GetPendingTxCountByAccount will return the pending tx count by account in txpool
	GetPendingTxCountByAccount(account string) uint64

	// GetPendingTxByHash will return the pending tx by hash in txpool
	GetPendingTxByHash(hash *types.Hash) *types.Transaction

	// GetTotalPendingTxCount will return the number of pending txs in txpool
	GetTotalPendingTxCount() uint64

	// GetLowWatermark will return the low watermark of consensus engine
	GetLowWatermark() uint64

	SubscribeTxEvent(events chan<- []*types.Transaction) event.Subscription
}

func New(consensusType string, opts ...common.Option) (Consensus, error) {
	config, err := common.GenerateConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("generate config: %w", err)
	}

	// Get the consensus constructor according to different consensus type.
	switch consensusType {
	case repo.ConsensusTypeSolo:
		return solo.NewNode(config)
	case repo.ConsensusTypeRbft:
		return rbft.NewNode(config)
	case repo.ConsensusTypeSoloDev:
		return solo_dev.NewNode(config)
	default:
		return nil, fmt.Errorf("unsupport consensus type: %s", consensusType)
	}
}
