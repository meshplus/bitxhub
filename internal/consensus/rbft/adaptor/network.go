package adaptor

import (
	"context"

	"github.com/pkg/errors"

	"github.com/axiomesh/axiom-bft/common/consensus"
)

func (a *RBFTAdaptor) Broadcast(ctx context.Context, msg *consensus.ConsensusMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	if a.config.Config.Rbft.EnableMultiPipes {
		pipe, ok := a.msgPipes[int32(msg.Type)]
		if !ok {
			return errors.Errorf("unsupported broadcast msg type: %v", msg.Type)
		}

		return pipe.Broadcast(ctx, a.broadcastNodes, data)
	}

	return a.globalMsgPipe.Broadcast(ctx, a.broadcastNodes, data)
}

func (a *RBFTAdaptor) Unicast(ctx context.Context, msg *consensus.ConsensusMessage, to string) error {
	data, err := msg.Marshal()
	if err != nil {
		return err
	}

	if a.config.Config.Rbft.EnableMultiPipes {
		pipe, ok := a.msgPipes[int32(msg.Type)]
		if !ok {
			return errors.Errorf("unsupported unicast msg type: %v", msg.Type)
		}

		return pipe.Send(ctx, to, data)
	}

	return a.globalMsgPipe.Send(ctx, to, data)
}
