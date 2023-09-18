package app

import (
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom/pkg/model/events"
	"github.com/axiomesh/axiom/pkg/repo"
)

func (axm *Axiom) start() {
	go axm.listenEvent()

	go func() {
		for {
			select {
			case commitEvent := <-axm.Order.Commit():
				axm.logger.WithFields(logrus.Fields{
					"height": commitEvent.Block.BlockHeader.Number,
					"count":  len(commitEvent.Block.Transactions),
				}).Info("Generated block")
				axm.BlockExecutor.ExecuteBlock(commitEvent)
			case <-axm.Ctx.Done():
				return
			}
		}
	}()
}

func (axm *Axiom) listenEvent() {
	blockCh := make(chan events.ExecutedEvent)
	configCh := make(chan *repo.Repo)

	blockSub := axm.BlockExecutor.SubscribeBlockEvent(blockCh)
	configSub := axm.repo.SubscribeConfigChange(configCh)

	defer blockSub.Unsubscribe()
	defer configSub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			axm.Order.ReportState(ev.Block.BlockHeader.Number, ev.Block.BlockHash, ev.TxHashList)
		case config := <-configCh:
			axm.ReConfig(config)
		case <-axm.Ctx.Done():
			return
		}
	}
}
