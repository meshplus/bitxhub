package adaptor

import (
	"context"

	"github.com/hyperchain/go-hpc-rbft/common/consensus"
	"github.com/meshplus/bitxhub-kit/types/pb"
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

	return s.peerMgr.Broadcast(p2pmsg)
}

func (s *RBFTAdaptor) unicast(_ context.Context, msg *consensus.ConsensusMessage, to any) error {
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

func (s *RBFTAdaptor) Unicast(ctx context.Context, msg *consensus.ConsensusMessage, to uint64) error {
	return s.unicast(ctx, msg, to)
}

func (s *RBFTAdaptor) UnicastByHostname(ctx context.Context, msg *consensus.ConsensusMessage, to string) error {
	return s.unicast(ctx, msg, to)
}
