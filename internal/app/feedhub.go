package app

import (
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/sirupsen/logrus"
)

func (bxh *BitXHub) start() {
	go bxh.listenEvent()

	go func() {
		for {
			select {
			case commitEvent := <-bxh.Order.Commit():
				bxh.logger.WithFields(logrus.Fields{
					"height": commitEvent.Block.BlockHeader.Number,
					"count":  len(commitEvent.Block.Transactions.Transactions),
				}).Info("Generated block")
				bxh.BlockExecutor.ExecuteBlock(commitEvent)
			case <-bxh.Ctx.Done():
				return
			}
		}
	}()
}

func (bxh *BitXHub) listenEvent() {
	blockCh := make(chan events.ExecutedEvent)
	orderMsgCh := make(chan peermgr.OrderMessageEvent)
	configCh := make(chan *repo.Repo)

	blockSub := bxh.BlockExecutor.SubscribeBlockEvent(blockCh)
	orderMsgSub := bxh.PeerMgr.SubscribeOrderMessage(orderMsgCh)
	configSub := bxh.repo.SubscribeConfigChange(configCh)

	defer blockSub.Unsubscribe()
	defer orderMsgSub.Unsubscribe()
	defer configSub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			go bxh.Order.ReportState(ev.Block.BlockHeader.Number, ev.Block.BlockHash, ev.TxHashList)
		case ev := <-orderMsgCh:
			go func() {
				if ev.IsTxsFromRemote {
					if err := bxh.Order.SubmitTxsFromRemote(ev.Txs); err != nil {
						bxh.logger.Error(err)
					}
				} else {
					if err := bxh.Order.Step(ev.Data); err != nil {
						bxh.logger.Error(err)
					}
				}
			}()
		case config := <-configCh:
			bxh.ReConfig(config)
		case <-bxh.Ctx.Done():
			return
		}
	}
}
