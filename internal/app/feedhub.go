package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/meshplus/bitxhub-core/governance"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-core/tss/message"
	"github.com/meshplus/bitxhub-model/pb"
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
	tssMsgCh := make(chan *pb.Message)
	tssKeygenReqCh := make(chan *pb.Message)

	blockSub := bxh.BlockExecutor.SubscribeBlockEvent(blockCh)
	orderMsgSub := bxh.PeerMgr.SubscribeOrderMessage(orderMsgCh)
	nodeSub := bxh.BlockExecutor.SubscribeNodeEvent(nodeCh)
	configSub := bxh.repo.SubscribeConfigChange(configCh)
	tssSub := bxh.PeerMgr.SubscribeTssMessage(tssMsgCh)
	tssKeygenReqSub := bxh.PeerMgr.SubscribeTssKeygenReq(tssKeygenReqCh)

	defer blockSub.Unsubscribe()
	defer orderMsgSub.Unsubscribe()
	defer nodeSub.Unsubscribe()
	defer configSub.Unsubscribe()
	defer tssSub.Unsubscribe()
	defer tssKeygenReqSub.Unsubscribe()

	for {
		select {
		case msg := <-tssMsgCh:
			bxh.logger.Debugf("get tss msg to put")
			go func() {
				wireMsg := &message.WireMessage{}
				if err := json.Unmarshal(msg.Data, wireMsg); err != nil {
					bxh.logger.Errorf(fmt.Sprintf("unmarshal wire msg error: %v", err))
				} else {
					bxh.TssMgr.PutTssMsg(msg, wireMsg.MsgID)
				}
			}()
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
			go func() {
				switch ev.NodeEventType {
				case governance.EventLogout:
					// order
					if err := bxh.Order.Ready(); err != nil {
						bxh.logger.Error(err)
						return
					}
					if err := bxh.Order.DelNode(ev.NodeId); err != nil {
						bxh.logger.Error(err)
					}

					// tss
					if bxh.repo.Config.Tss.EnableTSS {
						// 1. delete node
						err := bxh.TssMgr.DeleteTssNodes([]string{strconv.Itoa(int(ev.NodeId))})
						if err != nil {
							bxh.logger.Errorf("delete tss node error: %v", err)
						}

						// 2. update threshold
						bxh.TssMgr.UpdateThreshold(bxh.Order.Quorum() - 1)
					}

				}
			}()
		case reqMsg := <-tssKeygenReqCh:
			if reqMsg.Type == pb.Message_TSS_KEYGEN_REQ {
				// update threshold
				bxh.TssMgr.UpdateThreshold(bxh.Order.Quorum() - 1)

				// keygen
				time1 := time.Now()
				bxh.logger.Infof("...... tss req key gen")
				if err := bxh.TssMgr.Keygen(true); err != nil {
					bxh.logger.Errorf("tss key generate error: %v", err)

				}
				timeKeygen := time.Since(time1).Milliseconds()
				bxh.logger.Infof("=============================keygen time: %d", timeKeygen)
			}
		case config := <-configCh:
			bxh.ReConfig(config)
		case <-bxh.Ctx.Done():
			return
		}
	}
}
