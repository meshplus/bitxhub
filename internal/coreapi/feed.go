package coreapi

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/model/events"
)

type FeedAPI CoreAPI

var _ api.FeedAPI = (*FeedAPI)(nil)

func (api *FeedAPI) SubscribeNewBlockEvent(ch chan<- events.NewBlockEvent) event.Subscription {
	return api.bxh.Executor.SubscribeBlockEvent(ch)
}
