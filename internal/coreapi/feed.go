package coreapi

import (
	"github.com/ethereum/go-ethereum/event"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/coreapi/api"
	"github.com/axiomesh/axiom-ledger/pkg/events"
)

type FeedAPI CoreAPI

var _ api.FeedAPI = (*FeedAPI)(nil)

func (api *FeedAPI) SubscribeNewTxEvent(ch chan<- []*types.Transaction) event.Subscription {
	return api.axiomLedger.Consensus.SubscribeTxEvent(ch)
}

func (api *FeedAPI) SubscribeNewBlockEvent(ch chan<- events.ExecutedEvent) event.Subscription {
	return api.axiomLedger.BlockExecutor.SubscribeBlockEventForRemote(ch)
}

func (api *FeedAPI) SubscribeLogsEvent(ch chan<- []*types.EvmLog) event.Subscription {
	return api.axiomLedger.BlockExecutor.SubscribeLogsEvent(ch)
}

// TODO: check it
func (api *FeedAPI) BloomStatus() (uint64, uint64) {
	return 4096, 0
}
