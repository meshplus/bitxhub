package grpc

import (
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) SubscribeAuditInfo(req *pb.AuditSubscriptionRequest, server pb.ChainBroker_SubscribeAuditInfoServer) error {
	switch req.Type.String() {
	case pb.AuditSubscriptionRequest_AUDIT_NODE.String():
		return cbs.handleAuditNodeSubscription(server, req.AuditNodeId, req.BlockHeight)
	}

	return nil
}

func (cbs *ChainBrokerService) handleAuditNodeSubscription(server pb.ChainBroker_SubscribeServer, auditNodeID string, blockStart uint64) error {
	dataCh := make(chan *pb.AuditTxInfo)

	go func() {
		err := cbs.api.Audit().HandleAuditNodeSubscription(dataCh, auditNodeID, blockStart)
		if err != nil {
			cbs.logger.WithField("auditNodeID", auditNodeID).Errorf("Handle audit node subscription: %v", err)
		}
	}()

	for auditTxInfo := range dataCh {
		data, _ := auditTxInfo.Marshal()
		if err := server.Send(&pb.Response{
			Data: data,
		}); err != nil {
			return fmt.Errorf("send data failed: %w", err)
		}
	}

	return nil
}
