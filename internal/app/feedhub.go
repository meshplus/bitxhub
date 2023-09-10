package app

import (
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom/pkg/model/events"
	"github.com/axiomesh/axiom/pkg/repo"
)

func (ax *Axiom) start() {
	go ax.listenEvent()

	go func() {
		for {
			select {
			case commitEvent := <-ax.Order.Commit():
				ax.logger.WithFields(logrus.Fields{
					"height": commitEvent.Block.BlockHeader.Number,
					"count":  len(commitEvent.Block.Transactions),
				}).Info("Generated block")
				ax.BlockExecutor.ExecuteBlock(commitEvent)
			case <-ax.Ctx.Done():
				return
			}
		}
	}()
}

func (ax *Axiom) listenEvent() {
	blockCh := make(chan events.ExecutedEvent)
	configCh := make(chan *repo.Repo)

	blockSub := ax.BlockExecutor.SubscribeBlockEvent(blockCh)
	configSub := ax.repo.SubscribeConfigChange(configCh)

	defer blockSub.Unsubscribe()
	defer configSub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			ax.Order.ReportState(ev.Block.BlockHeader.Number, ev.Block.BlockHash, ev.TxHashList)
		case config := <-configCh:
			ax.ReConfig(config)
		case <-ax.Ctx.Done():
			return
		}
	}
}
