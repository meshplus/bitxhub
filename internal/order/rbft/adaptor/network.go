package adaptor

import (
	"context"

	"github.com/samber/lo"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-bft/common/consensus"
)

func (s *RBFTAdaptor) Broadcast(ctx context.Context, msg *consensus.ConsensusMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	return s.msgPipe.Broadcast(ctx, lo.Map(lo.Flatten([][]*rbft.NodeInfo{s.EpochInfo.ValidatorSet, s.EpochInfo.CandidateSet}), func(item *rbft.NodeInfo, index int) string {
		return item.P2PNodeID
	}), data)
}

func (s *RBFTAdaptor) Unicast(ctx context.Context, msg *consensus.ConsensusMessage, to string) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	return s.msgPipe.Send(ctx, to, data)
}
