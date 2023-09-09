package adaptor

import (
	"context"

	"github.com/pkg/errors"

	"github.com/axiomesh/axiom-bft/common/consensus"
)

func (s *RBFTAdaptor) Broadcast(ctx context.Context, msg *consensus.ConsensusMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	if s.config.Config.Rbft.EnableMultiPipes {
		pipe, ok := s.msgPipes[int32(msg.Type)]
		if !ok {
			return errors.Errorf("unsupported broadcast msg type: %v", msg.Type)
		}

		return pipe.Broadcast(ctx, s.broadcastNodes, data)
	}

	return s.globalMsgPipe.Broadcast(ctx, s.broadcastNodes, data)
}

func (s *RBFTAdaptor) Unicast(ctx context.Context, msg *consensus.ConsensusMessage, to string) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	if s.config.Config.Rbft.EnableMultiPipes {
		pipe, ok := s.msgPipes[int32(msg.Type)]
		if !ok {
			return errors.Errorf("unsupported unicast msg type: %v", msg.Type)
		}

		return pipe.Send(ctx, to, data)
	}

	return s.globalMsgPipe.Send(ctx, to, data)
}
