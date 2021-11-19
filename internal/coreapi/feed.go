package coreapi

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/model/events"
)

type FeedAPI CoreAPI

var _ api.FeedAPI = (*FeedAPI)(nil)

func (api *FeedAPI) SubscribeNewTxEvent(ch chan<- pb.Transactions) event.Subscription {
	return api.bxh.Order.SubscribeTxEvent(ch)
}

func (api *FeedAPI) SubscribeNewBlockEvent(ch chan<- events.ExecutedEvent) event.Subscription {
	return api.bxh.BlockExecutor.SubscribeBlockEvent(ch)
}

func (api *FeedAPI) SubscribeLogsEvent(ch chan<- []*pb.EvmLog) event.Subscription {
	return api.bxh.BlockExecutor.SubscribeLogsEvent(ch)
}

func (api *FeedAPI) SubscribeAuditEvent(ch chan<- *pb.AuditTxInfo) event.Subscription {
	return api.bxh.BlockExecutor.SubscribeAuditEvent(ch)
}

func (api *FeedAPI) BloomStatus() (uint64, uint64) {
	return 4096, 0
}
