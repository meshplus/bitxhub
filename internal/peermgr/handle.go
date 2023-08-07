package peermgr

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types/pb"
	network "github.com/axiomesh/axiom-p2p"
)

func (swarm *Swarm) handleMessage(s network.Stream, data []byte) {
	handler := func() error {
		defer swarm.p2p.ReleaseStream(s)
		m := &pb.Message{}
		if err := m.UnmarshalVT(data); err != nil {
			swarm.logger.Errorf("unmarshal message error: %s", err.Error())
			return err
		}
		if m.Type != pb.Message_CONSENSUS {
			swarm.logger.Debugf("handle msg: %s", m.Type)
		}
		switch m.Type {
		case pb.Message_GET_BLOCK:
			return swarm.handleGetBlockPack(s, m)
		default:
			swarm.logger.WithField("module", "p2p").Errorf("can't handle msg[type: %v]", m.Type)
			return nil
		}
	}

	go func() {
		if err := handler(); err != nil {
			swarm.logger.WithFields(logrus.Fields{
				"error": err,
			}).Error("Handle message")
		}
	}()
}

func (swarm *Swarm) handleGetBlockPack(s network.Stream, msg *pb.Message) error {
	num, err := strconv.Atoi(string(msg.Data))
	if err != nil {
		return fmt.Errorf("convert %s string to int failed: %w", string(msg.Data), err)
	}

	block, err := swarm.ledger.GetBlock(uint64(num))
	if err != nil {
		return fmt.Errorf("get block with height %d failed: %w", num, err)
	}

	v, err := block.Marshal()
	if err != nil {
		return fmt.Errorf("marshal block error: %w", err)
	}

	m := &pb.Message{
		Type: pb.Message_GET_BLOCK_ACK,
		Data: v,
	}

	if err := swarm.SendWithStream(s, m); err != nil {
		return fmt.Errorf("send %s with stream failed: %w", m.String(), err)
	}

	return nil
}
