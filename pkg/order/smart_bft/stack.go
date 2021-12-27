package smart_bft

import (
	"sort"
	"strconv"
	"time"

	bft "github.com/SmartBFT-Go/consensus/pkg/types"
	"github.com/SmartBFT-Go/consensus/smartbftprotos"
	"github.com/golang/protobuf/proto"
	"github.com/meshplus/bitxhub-model/pb"
	proto2 "github.com/meshplus/bitxhub/pkg/order/smart_bft/proto"
	"github.com/sirupsen/logrus"
)

func (n *Node) SendConsensus(targetID uint64, m *smartbftprotos.Message) {
	n.logger.WithFields(logrus.Fields{
		"target": targetID,
		"msg":    m,
	}).Info("heartbeat")

	mData, err := proto.Marshal(m)
	if err != nil {
		n.logger.Errorf("Marshal smartbft message error")
		return
	}
	msg := proto2.BftMessage{
		Type:   proto2.BftMessage_CONSENSUS,
		FromId: n.id,
		Data:   mData,
	}
	data, err := msg.Marshal()
	if err != nil {
		n.logger.Errorf("Marshal bft message error")
		return
	}
	p2pMsg := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: data,
	}
	if err := n.peerMgr.AsyncSend(targetID, p2pMsg); err != nil {
		n.logger.Debugf("Fail to async send consensus msg to %d", targetID)
	}
}

func (n *Node) SendTransaction(targetID uint64, request []byte) {
	msg := proto2.BftMessage{
		Type:   proto2.BftMessage_BROADCAST_TX,
		FromId: n.id,
		Data:   request,
	}
	data, err := msg.Marshal()
	if err != nil {
		n.logger.Errorf("Marshal bft message error")
		return
	}
	p2pMsg := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: data,
	}
	if err := n.peerMgr.AsyncSend(targetID, p2pMsg); err != nil {
		n.logger.Debugf("Fail to async send tx msg to %d", targetID)
	}
}

func (n *Node) Nodes() []uint64 {
	nodes := make([]uint64, 0)
	for id, _ := range n.peerMgr.OrderPeers() {
		nodes = append(nodes, id)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i] < nodes[j]
	})
	return nodes
}

func (n *Node) AssembleProposal(metadata []byte, requests [][]byte) bft.Proposal {
	txs := make([]pb.Transaction, 0, len(requests))
	for _, request := range requests {
		tx, err := pb.UnmarshalTx(request)
		if err != nil {
			n.logger.Errorf("Unable to unmarshal tx, error: %v", err)
		}
		txs = append(txs, tx)
	}
	md := &smartbftprotos.ViewMetadata{}
	if err := proto.Unmarshal(metadata, md); err != nil {
		n.logger.Errorf("Unable to unmarshal metadata, error: %v", err)
	}
	blockHeader := &pb.BlockHeader{
		Version:   []byte("1.0.0"),
		Number:    md.LatestSequence,
		Timestamp: time.Now().Unix(),
	}
	n.logger.Infof("======== Replica %d call assemble, height=%d", n.id, blockHeader.Number)
	hData, err := blockHeader.Marshal()
	if err != nil {
		n.logger.Errorf("Unable to marshal block header, error: %v", err)
	}
	transactions := pb.Transactions{Transactions: txs}
	data, err := transactions.Marshal()
	if err != nil {
		n.logger.Errorf("Unable to marshal pb.transactions, error: %v", err)
	}
	return bft.Proposal{
		Header:   hData,
		Payload:  data,
		Metadata: metadata,
	}
}

func (n *Node) MembershipChange() bool {
	return false
}

func (n *Node) Deliver(proposal bft.Proposal, signature []bft.Signature) bft.Reconfig {
	var header pb.BlockHeader
	if err := header.Unmarshal(proposal.Header); err != nil {
		n.logger.Errorf("Unable to unmarshal pb.BlockHeader, error: %v", err)
	}
	n.logger.Infof("======== Replica %d call execute, height=%d", n.id, header.Number)
	var txs pb.Transactions
	if err := txs.Unmarshal(proposal.Payload); err != nil {
		n.logger.Errorf("Unable to unmarshal pb.transactions, error: %v", err)
	}
	sig2s := make([]*proto2.Signature, 0, len(signature))
	for _, sig := range signature {
		sig2s = append(sig2s, &proto2.Signature{
			Id:    sig.ID,
			Value: sig.Value,
			Msg:   sig.Msg,
		})
	}
	sig3s := &proto2.Signatures{Signature: sig2s}
	sigsData, err := sig3s.Marshal()
	if err != nil {
		n.logger.Errorf("Unable to marshal signatures, error: %v", err)
	}
	block := &pb.Block{
		BlockHeader:  &header,
		Transactions: &txs,
		Signature:    sigsData,
	}
	localList := make([]bool, len(txs.Transactions))
	for i := 0; i < len(txs.Transactions); i++ {
		localList[i] = false
	}
	executeEvent := &pb.CommitEvent{
		Block:     block,
		LocalList: localList,
	}

	n.commitC <- executeEvent
	n.lastExec = header.Number

	md := &smartbftprotos.ViewMetadata{}
	if err := proto.Unmarshal(proposal.Metadata, md); err != nil {
		n.logger.Errorf("Unable to unmarshal metadata, error: %v", err)
	}

	return bft.Reconfig{InLatestDecision: false}
}

// Sync TODO:sync block from others peer
func (n *Node) Sync() bft.SyncResponse {
	return bft.SyncResponse{}
}

func (n *Node) RequestID(req []byte) bft.RequestInfo {
	tx, err := pb.UnmarshalTx(req)
	if err != nil {
		n.logger.Errorf("Unable to marshal pb.transaction, error: %v", err)
	}
	//TODO:need clientID?
	return bft.RequestInfo{ID: tx.GetHash().String(), ClientID: strconv.FormatUint(tx.GetNonce(), 10)}
}

func (n *Node) VerifyProposal(proposal bft.Proposal) ([]bft.RequestInfo, error) {
	txs := pb.Transactions{}
	if err := txs.Unmarshal(proposal.Payload); err != nil {
		return nil, err
	}
	infos := make([]bft.RequestInfo, len(txs.Transactions))
	for i, tx := range txs.Transactions {
		infos[i] = bft.RequestInfo{ID: tx.GetHash().String(), ClientID: strconv.FormatUint(tx.GetNonce(), 10)}
	}
	return infos, nil
}

func (n *Node) VerifyRequest(val []byte) (bft.RequestInfo, error) {
	return n.RequestID(val), nil
}

func (n *Node) VerifyConsenterSig(signature bft.Signature, prop bft.Proposal) ([]byte, error) {
	return nil, nil
}

func (n *Node) VerifySignature(signature bft.Signature) error {
	return nil
}

func (n *Node) VerificationSequence() uint64 {
	return 0
}

func (n *Node) RequestsFromProposal(proposal bft.Proposal) []bft.RequestInfo {
	infos, _ := n.VerifyProposal(proposal)
	return infos
}

func (n *Node) AuxiliaryData(bytes []byte) []byte {
	return nil
}

func (n *Node) Sign(bytes []byte) []byte {
	return nil
}

func (n *Node) SignProposal(proposal bft.Proposal, auxiliaryInput []byte) *bft.Signature {
	return &bft.Signature{
		ID: n.id,
	}
}
