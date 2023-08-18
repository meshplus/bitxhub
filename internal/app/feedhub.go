package app

import (
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom/pkg/model/events"
	"github.com/axiomesh/axiom/pkg/repo"
)

func (bxh *Axiom) start() {
	go bxh.listenEvent()

	go func() {
		for {
			select {
			case commitEvent := <-bxh.Order.Commit():
				bxh.logger.WithFields(logrus.Fields{
					"height": commitEvent.Block.BlockHeader.Number,
					"count":  len(commitEvent.Block.Transactions),
				}).Info("Generated block")
				bxh.BlockExecutor.ExecuteBlock(commitEvent)
			case <-bxh.Ctx.Done():
				return
			}
		}
	}()
}

func (bxh *Axiom) listenEvent() {
	blockCh := make(chan events.ExecutedEvent)
	configCh := make(chan *repo.Repo)

	blockSub := bxh.BlockExecutor.SubscribeBlockEvent(blockCh)
	configSub := bxh.repo.SubscribeConfigChange(configCh)

	defer blockSub.Unsubscribe()
	defer configSub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			go bxh.Order.ReportState(ev.Block.BlockHeader.Number, ev.Block.BlockHash, ev.TxHashList)
		case config := <-configCh:
			bxh.ReConfig(config)
		case <-bxh.Ctx.Done():
			return
		}
	}
}
