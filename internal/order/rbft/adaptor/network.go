package adaptor

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/axiomesh/axiom-bft/common/consensus"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-kit/types/pb"
)

func (s *RBFTAdaptor) Broadcast(ctx context.Context, msg *consensus.ConsensusMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	p2pmsg := &pb.Message{
		Type:    pb.Message_CONSENSUS,
		Data:    data,
		Version: []byte("0.1.0"),
	}

	raw, err := p2pmsg.MarshalVT()
	if err != nil {
		return err
	}
	return s.msgPipe.Broadcast(ctx, lo.MapToSlice(s.peerMgr.OrderPeers(), func(k uint64, v *types.VpInfo) string {
		return v.Pid
	}), raw)
}

func (s *RBFTAdaptor) unicast(ctx context.Context, msg *consensus.ConsensusMessage, to any) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	m := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: data,
	}

	raw, err := m.MarshalVT()
	if err != nil {
		return err
	}
	var addr string
	switch to := to.(type) {
	case uint64:
		i, ok := s.peerMgr.OrderPeers()[to]
		if !ok {
			return fmt.Errorf("p2p peer id unknown type: %w", err)
		}
		addr = i.Pid
	case string:
		addr = to
	default:
		return fmt.Errorf("p2p unsupported peer id type: %v", to)
	}
	return s.msgPipe.Send(ctx, addr, raw)
}

func (s *RBFTAdaptor) Unicast(ctx context.Context, msg *consensus.ConsensusMessage, to uint64) error {
	return s.unicast(ctx, msg, to)
}

func (s *RBFTAdaptor) UnicastByHostname(ctx context.Context, msg *consensus.ConsensusMessage, to string) error {
	return s.unicast(ctx, msg, to)
}
