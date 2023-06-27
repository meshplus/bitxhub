package rbft

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/gogo/protobuf/proto"
	"github.com/meshplus/bitxhub-core/order"
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"github.com/ultramesh/rbft/rbftpb"
)

type Stack struct {
	localID           uint64
	store             *Storage
	peerMgr           orderPeerMgr.OrderPeerManager
	priv              crypto.PrivateKey
	nodes             map[uint64]*pb.VpInfo
	readyC            chan *ready
	blockC            chan *pb.CommitEvent
	logger            logrus.FieldLogger
	getChainMetaFunc  func() *pb.ChainMeta
	stateUpdating     bool
	stateUpdateHeight uint64
	applyConfChange   func(cc *rbftpb.ConfState)
	cancel            context.CancelFunc
	isNew             bool
}

type ready struct {
	txs       []pb.Transaction
	localList []bool
	height    uint64
	timestamp int64
}

func NewStack(store *Storage, config *order.Config, blockC chan *pb.CommitEvent, cancel context.CancelFunc, isNew bool) (*Stack, error) {
	stack := &Stack{
		localID:          config.ID,
		store:            store,
		peerMgr:          config.PeerMgr,
		priv:             config.PrivKey,
		nodes:            config.Nodes,
		readyC:           make(chan *ready, 1024),
		logger:           config.Logger,
		getChainMetaFunc: config.GetChainMetaFunc,
		blockC:           blockC,
		cancel:           cancel,
		isNew:            isNew,
	}
	return stack, nil
}

func (s *Stack) Broadcast(msg *rbftpb.ConsensusMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	p2pmsg := &pb.Message{
		Type:    pb.Message_CONSENSUS,
		Data:    data,
		Version: []byte("0.1.0"),
	}

	return s.peerMgr.Broadcast(p2pmsg)
}

func (s *Stack) Unicast(msg *rbftpb.ConsensusMessage, to uint64) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	m := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: data,
	}

	return s.peerMgr.AsyncSend(to, m)
}

func (s *Stack) UpdateTable(change *rbftpb.ConfChange) {
	switch change.Type {
	case rbftpb.ConfChangeType_ConfChangeAddNode:
		newNodeID := change.NodeID
		newPeerInfo := &rbftpb.Peer{}
		if err := proto.Unmarshal(change.Context, newPeerInfo); err != nil {
			s.logger.Errorf("Unmarshal new peer info failed, err: %s", err.Error())
			return
		}
		vpInfo := &pb.VpInfo{}
		if err := proto.Unmarshal(newPeerInfo.Context, vpInfo); err != nil {
			s.logger.Errorf("Unmarshal vp info failed, err: %s", err.Error())
			return
		}
		s.peerMgr.AddNode(newNodeID, vpInfo)
		s.nodes[newNodeID] = vpInfo
		if newNodeID == s.localID {
			s.isNew = false
		}
		s.logger.Infof("Apply latest routing table %+v to RBFT core", s.peerMgr.Peers())
		peers, err := sortPeers(s.peerMgr.OrderPeers())
		if err != nil {
			s.logger.Errorf("Sort peers failed, err: %s", err.Error())
			return
		}
		quorumRouters := &rbftpb.Router{
			Peers: peers,
		}
		cs := &rbftpb.ConfState{
			QuorumRouter: quorumRouters,
		}
		s.applyConfChange(cs)

	case rbftpb.ConfChangeType_ConfChangeRemoveNode:
		delID := change.NodeID
		oldPeers := s.peerMgr.OrderPeers()
		s.peerMgr.DelNode(delID)
		delete(s.nodes, delID)
		if delID == s.localID {
			delete(oldPeers, delID)
			go s.stop(oldPeers)
			return
		}
		s.logger.Infof("Apply latest routing table %+v to RBFT core", s.peerMgr.Peers())
		peers, err := sortPeers(s.peerMgr.OrderPeers())
		if err != nil {
			s.logger.Errorf("Sort peers failed, err: %s", err.Error())
			return
		}
		quorumRouters := &rbftpb.Router{
			Peers: peers,
		}
		cs := &rbftpb.ConfState{
			QuorumRouter: quorumRouters,
		}
		s.applyConfChange(cs)

	case rbftpb.ConfChangeType_ConfChangeUpdateNode:
		router := &rbftpb.Router{}
		if err := proto.Unmarshal(change.Context, router); err != nil {
			s.logger.Errorf("Unmarshal router failed, err: %s", err.Error())
			return
		}
		vpInfos := make(map[uint64]*pb.VpInfo)
		for _, peer := range router.Peers {
			vpInfo := &pb.VpInfo{}
			if err := proto.Unmarshal(peer.Context, vpInfo); err != nil {
				s.logger.Errorf("Unmarshal vp info failed, err: %s", err.Error())
				return
			}
			vpInfos[vpInfo.Id] = vpInfo
		}
		s.nodes = vpInfos
		// for restart node, if it has been deleted, then exit the consensus cluster.
		isExit := s.peerMgr.UpdateRouter(vpInfos, s.isNew)
		if isExit {
			s.stop(vpInfos)
			return
		}
		cs := &rbftpb.ConfState{
			QuorumRouter: router,
		}
		s.applyConfChange(cs)
	}
}

func (s *Stack) Sign(msg []byte) ([]byte, error) {
	h := sha256.Sum256(msg)
	return s.priv.Sign(h[:])
}

func (s *Stack) Verify(peerID uint64, signature []byte, msg []byte) error {
	h := sha256.Sum256(msg)
	addr := types.NewAddressByStr(s.nodes[peerID].Account)
	ret, err := asym.Verify(crypto.Secp256k1, signature, h[:], *addr)
	if err != nil {
		return err
	}

	if !ret {
		return fmt.Errorf("verify error")
	}

	return nil
}

func (s *Stack) Execute(requests []pb.Transaction, localList []bool, seqNo uint64, timestamp int64) {
	s.readyC <- &ready{
		txs:       requests,
		localList: localList,
		height:    seqNo,
		timestamp: timestamp,
	}
}

func (s *Stack) StateUpdate(seqNo uint64, digest string, peers []uint64) {
	s.stateUpdating = true
	s.stateUpdateHeight = seqNo

	chain := s.getChainMetaFunc()
	s.logger.WithFields(logrus.Fields{
		"target":       seqNo,
		"target_hash":  digest,
		"current":      chain.Height,
		"current_hash": chain.BlockHash.String(),
	}).Info("State Update")
	get := func(peers []uint64, i int) (block *pb.Block, err error) {
		for _, id := range peers {
			block, err = s.getBlock(id, i)
			if err != nil {
				s.logger.Error(err)
				continue
			}

			return block, nil
		}

		return nil, fmt.Errorf("can't get block from all peers")
	}

	blockCache := make([]*pb.Block, seqNo-chain.Height)
	var block *pb.Block
	for i := seqNo; i > chain.Height; i-- {
		if err := retry.Retry(func(attempt uint) (err error) {
			block, err = get(peers, int(i))
			if err != nil {
				s.logger.Info(err)
				return err
			}

			if digest != block.BlockHash.String() {
				s.logger.WithFields(logrus.Fields{
					"required": digest,
					"received": block.BlockHash.String(),
					"height":   i,
				}).Error("block hash is inconsistent in state update state")
				return err
			}

			digest = block.BlockHeader.ParentHash.String()
			blockCache[i-chain.Height-1] = block

			return nil
		}, strategy.Wait(200*time.Millisecond)); err != nil {
			s.logger.Error(err)
		}
	}

	for _, block := range blockCache {
		if block == nil {
			s.logger.Error("Receive a nil block")
			return
		}
		localList := make([]bool, len(block.Transactions.Transactions))
		for i := 0; i < len(block.Transactions.Transactions); i++ {
			localList[i] = false
		}
		commitEvent := &pb.CommitEvent{
			Block:     block,
			LocalList: localList,
		}
		s.blockC <- commitEvent
	}
}

func (s *Stack) SendFilterEvent(informType rbftpb.InformType, message ...interface{}) {
	// TODO: add implement
}

func (s *Stack) getBlock(id uint64, i int) (*pb.Block, error) {
	m := &pb.Message{
		Type: pb.Message_GET_BLOCK,
		Data: []byte(strconv.Itoa(i)),
	}

	res, err := s.peerMgr.Send(id, m)
	if err != nil {
		return nil, err
	}

	block := &pb.Block{}
	if err := block.Unmarshal(res.Data); err != nil {
		return nil, err
	}

	return block, nil
}

func (s *Stack) stop(peers map[uint64]*pb.VpInfo) {
	s.logger.Infof("======== THIS NODE WILL STOP IN 3 SECONDS")
	<-time.After(3 * time.Second)
	s.cancel()
	_ = s.Destroy()
	s.peerMgr.Disconnect(peers)
	s.logger.Infof("======== THIS NODE HAS BEEN DELETED!!!")
	os.Exit(1)
	return
}
