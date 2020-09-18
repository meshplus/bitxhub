package app

import (
	"context"

	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/sirupsen/logrus"
)

func (bxh *BitXHub) start() {
	go bxh.listenEvent()

	go func() {
		for {
			select {
			case block := <-bxh.Order.Commit():
				bxh.logger.WithFields(logrus.Fields{
					"height": block.BlockHeader.Number,
					"count":  len(block.Transactions),
				}).Info("Generated block")
				bxh.BlockExecutor.ExecuteBlock(block)
			case <-bxh.ctx.Done():
				return
			}
		}
	}()
}

func (bxh *BitXHub) listenEvent() {
	blockCh := make(chan events.NewBlockEvent)
	orderMsgCh := make(chan events.OrderMessageEvent)
	blockSub := bxh.BlockExecutor.SubscribeBlockEvent(blockCh)
	orderMsgSub := bxh.PeerMgr.SubscribeOrderMessage(orderMsgCh)

	defer blockSub.Unsubscribe()
	defer orderMsgSub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			go bxh.Order.ReportState(ev.Block.BlockHeader.Number, ev.Block.BlockHash)
			go bxh.Router.PutBlockAndMeta(ev.Block, ev.InterchainMeta)
		case ev := <-orderMsgCh:
			go func() {
				if err := bxh.Order.Step(context.Background(), ev.Data); err != nil {
					bxh.logger.Error(err)
				}
			}()
		case <-bxh.ctx.Done():
			return
		}
	}
}
