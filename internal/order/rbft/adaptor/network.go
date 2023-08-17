package adaptor

import (
	"context"

	"github.com/samber/lo"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-bft/common/consensus"
	"github.com/axiomesh/axiom-kit/types/pb"
	"github.com/axiomesh/axiom/pkg/repo"
)

func (s *RBFTAdaptor) Broadcast(ctx context.Context, msg *consensus.ConsensusMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	p2pmsg := &pb.Message{
		Type:    pb.Message_CONSENSUS,
		Data:    data,
		Version: repo.P2PMsgV1,
	}

	raw, err := p2pmsg.MarshalVT()
	if err != nil {
		return err
	}

	return s.msgPipe.Broadcast(ctx, lo.Map(s.EpochInfo.ValidatorSet, func(item *rbft.NodeInfo, index int) string {
		return item.P2PNodeID
	}), raw)
}

func (s *RBFTAdaptor) Unicast(ctx context.Context, msg *consensus.ConsensusMessage, to string) error {
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
	return s.msgPipe.Send(ctx, to, raw)
}
