package app

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	bkg "github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/pkg/peermgr"
	"github.com/meshplus/bitxhub/pkg/tssmgr"
	"strconv"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
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
			go func() {
				if !bxh.repo.Config.Tss.EnableTSS {
					bxh.logger.Debugf("receive tss msg, but tss config is %v, "+
						"if you are a new node,First admins should propose to delete node proposal, "+
						"then admins vote pass, finally repeat the process of addNode. "+
						"if you are a joined node, it means new joining node config tss however we are not support."+
						"you should ignore this msg，then  propose to delete new node proposal", bxh.repo.Config.Tss.EnableTSS)
					return

					//bxh.PeerMgr.Disconnect(bxh.PeerMgr.OrderPeers())
					//bxh.logger.Infof("======== THIS NODE HAS BEEN STOPPED!!!")
					//os.Exit(1)

					//// rewrite bitxhub config
					//err := repo.RewriteBxhConfig(bxh.repo.Config.RepoRoot, true)
					//if err != nil {
					//	bxh.logger.Errorf("rewrite bitxhub.toml err:%s", err)
					//	return
					//}
					//err = bxh.generateAndStartTss()
					//if err != nil {
					//	bxh.logger.Errorf(fmt.Sprintf("gererate Tss error: %v", err))
					//	return
					//}

				}
				switch msg.Type {
				case pb.Message_FETCH_TSS_NODES:
					id := binary.LittleEndian.Uint64(msg.Data)
					bxh.logger.Debugf("get node%s order ready", id)
					bxh.TssMgr.SetOrderReadyPeers(id)
					return
				default:
					if bxh.TssMgr.GetTssStatus() {
						bxh.logger.Debugf("keygen task is done")
						return
					}
					bxh.logger.Debugf("get tss msg to put")
					wireMsg := &message.WireMessage{}
					if err := json.Unmarshal(msg.Data, wireMsg); err != nil {
						bxh.logger.Errorf(fmt.Sprintf("unmarshal wire msg error: %v", err))
					} else {
						bxh.TssMgr.PutTssMsg(msg, wireMsg.MsgID)
					}
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
			go func() {
				if reqMsg.Type == pb.Message_TSS_KEYGEN_REQ {
					if err := retry.Retry(func(attempt uint) error {
						err := bxh.TssMgr.CheckThreshold()
						if err != nil {
							bxh.logger.WithFields(logrus.Fields{"config num": len(bxh.PeerMgr.OrderPeers()),
								"order ready peer": bxh.TssMgr.CountOrderReadyPeers()}).Warning(err)
							return err
						}
						return nil

					}, strategy.Limit(5), strategy.Wait(2*time.Second)); err != nil {
						if uint64(bxh.TssMgr.CountOrderReadyPeers()) > bxh.Order.Quorum()-1 {
							bxh.logger.WithFields(logrus.Fields{"Quorum num": bxh.Order.Quorum(),
								"order ready peer": bxh.TssMgr.CountOrderReadyPeers()}).Info("meet the threshold")
						} else {
							bxh.logger.WithFields(logrus.Fields{"Quorum num": bxh.Order.Quorum(),
								"order ready peer": bxh.TssMgr.CountOrderReadyPeers()}).Errorf("not meet the threshold")
							panic(err)
						}

					}
					// update threshold
					bxh.TssMgr.UpdateThreshold(bxh.Order.Quorum() - 1)

					// keygen
					time1 := time.Now()
					bxh.logger.Infof("...... tss req key gen")
					if err := bxh.TssMgr.Keygen(true); err != nil {
						bxh.logger.Errorf("tss key generate error: %v", err)

					}
					timeKeygen := time.Since(time1)
					bxh.logger.Infof("=============================keygen time: %v", timeKeygen)
				}
			}()
		case config := <-configCh:
			bxh.ReConfig(config)
		case <-bxh.Ctx.Done():
			return
		}
	}
}

func (bxh *BitXHub) generateAndStartTss() error {
	repoRoot := bxh.repo.Config.RepoRoot
	preParams, err := getPreparams(repoRoot)
	if err != nil {
		return fmt.Errorf("get preparams error: %w", err)
	}

	var preParam *bkg.LocalPreParams
	if len(preParams) <= int(bxh.repo.NetworkConfig.ID) {
		preParam = nil
	} else {
		preParam = preParams[bxh.repo.NetworkConfig.ID-1]
	}

	tssMgr, err := tssmgr.NewTssMgr(bxh.repo.Key.Libp2pPrivKey, bxh.repo.Config.Tss, bxh.repo.NetworkConfig, repoRoot, preParam, bxh.PeerMgr, loggers.Logger(loggers.TSS))
	if err != nil {
		return fmt.Errorf("create tss manager: %w, %v", err, bxh.repo.Config.Tss.PreParamTimeout)
	}
	bxh.PeerMgr.(*peermgr.Swarm).Tss = tssMgr
	bxh.TssMgr = tssMgr

	bxh.TssMgr.Start(bxh.Order.Quorum() - 1)

	time1 := time.Now()
	bxh.logger.Debugf("...... tss start key gen")
	if err := bxh.TssMgr.Keygen(false); err != nil {
		bxh.logger.Errorf("tss key generate error: %v", err)
		return fmt.Errorf("tss key generate: %w", err)
	}
	timeKeygen := time.Since(time1)
	bxh.logger.Infof("=============================keygen time: %v", timeKeygen)
	return nil
}
