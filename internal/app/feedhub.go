package app

import (
	"github.com/meshplus/bitxhub-core/governance"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/internal/repo"
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
	orderMsgCh := make(chan orderPeerMgr.OrderMessageEvent)
	nodeCh := make(chan events.NodeEvent)
	configCh := make(chan *repo.Repo)

	blockSub := bxh.BlockExecutor.SubscribeBlockEvent(blockCh)
	orderMsgSub := bxh.PeerMgr.SubscribeOrderMessage(orderMsgCh)
	nodeSub := bxh.BlockExecutor.SubscribeNodeEvent(nodeCh)
	configSub := bxh.repo.SubscribeConfigChange(configCh)

	defer blockSub.Unsubscribe()
	defer orderMsgSub.Unsubscribe()
	defer nodeSub.Unsubscribe()
	defer configSub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			go bxh.Order.ReportState(ev.Block.BlockHeader.Number, ev.Block.BlockHash, ev.TxHashList)
			go bxh.Router.PutBlockAndMeta(ev.Block, ev.InterchainMeta)
		case ev := <-orderMsgCh:
			go func() {
				if err := bxh.Order.Step(ev.Data); err != nil {
					bxh.logger.Error(err)
				}
			}()
		case ev := <-nodeCh:
			switch ev.NodeEventType {
			case governance.EventLogout:
				go func() {
					if err := bxh.Order.Ready(); err != nil {
						bxh.logger.Error(err)
						return
					}
					if err := bxh.Order.DelNode(ev.NodeId); err != nil {
						bxh.logger.Error(err)
					}
				}()
			}
		case config := <-configCh:
			bxh.ReConfig(config)
		case <-bxh.Ctx.Done():
			return
		}
	}
}
