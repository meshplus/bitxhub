package peermgr

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types/pb"
	network "github.com/axiomesh/axiom-p2p"
	"github.com/axiomesh/axiom/pkg/model"
)

func (swarm *Swarm) handleMessage(s network.Stream, data []byte) {
	m := &pb.Message{}
	handler := func() error {
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
		case pb.Message_GET_BLOCK_HEADERS:
			return swarm.handleGetBlockHeadersPack(s, m)
		case pb.Message_GET_BLOCKS:
			return swarm.handleGetBlocksPack(s, m)
		case pb.Message_FETCH_P2P_PUBKEY:
			return swarm.handleFetchP2PPubkey(s)
		case pb.Message_CONSENSUS:
			go swarm.orderMessageFeed.Send(OrderMessageEvent{
				IsTxsFromRemote: false,
				Data:            m.Data,
			})
		case pb.Message_PUSH_TXS:
			if !swarm.msgPushLimiter.Allow() {
				// rate limit exceeded, refuse to process the message
				swarm.logger.Warnf("Node received too many PUSH_TXS messages. Rate limiting in effect.")
				return nil
			}

			tx := &pb.BytesSlice{}
			if err := tx.UnmarshalVT(m.Data); err != nil {
				return fmt.Errorf("unmarshal PushTxs error: %v", err)
			}
			go swarm.orderMessageFeed.Send(OrderMessageEvent{
				IsTxsFromRemote: true,
				Txs:             tx.Slice,
			})
		case pb.Message_FETCH_BLOCK_SIGN:
			swarm.handleFetchBlockSignMessage(s, m.Data)
		default:
			swarm.logger.WithField("module", "p2p").Errorf("can't handle msg[type: %v]", m.Type)
			return nil
		}

		return nil
	}

	go func() {
		if err := handler(); err != nil {
			swarm.logger.WithFields(logrus.Fields{
				"error": err,
				"type":  m.Type.String(),
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

func (swarm *Swarm) handleGetBlockHeadersPack(s network.Stream, msg *pb.Message) error {
	req := &pb.GetBlockHeadersRequest{}
	if err := req.UnmarshalVT(msg.Data); err != nil {
		return fmt.Errorf("unmarshal get block headers request error: %w", err)
	}

	res := &pb.GetBlockHeadersResponse{}
	blockHeaders := make([][]byte, 0)
	for i := req.Start; i <= req.End; i++ {
		block, err := swarm.ledger.GetBlock(i)
		if err != nil {
			return fmt.Errorf("get block with height %d from ledger failed: %w", i, err)
		}
		raw, err := block.BlockHeader.Marshal()
		if err != nil {
			return fmt.Errorf("marshal block with height %d failed: %w", i, err)
		}
		blockHeaders = append(blockHeaders, raw)
	}
	res.BlockHeaders = blockHeaders
	v, err := res.MarshalVT()
	if err != nil {
		return fmt.Errorf("marshal get block headers response error: %w", err)
	}
	m := &pb.Message{
		Type: pb.Message_GET_BLOCK_HEADERS_ACK,
		Data: v,
	}

	if err := swarm.SendWithStream(s, m); err != nil {
		return fmt.Errorf("send %s with stream failed: %w", m.String(), err)
	}

	return nil
}

func (swarm *Swarm) handleFetchP2PPubkey(s network.Stream) error {
	pubkeyData, err := swarm.p2p.PrivKey().GetPublic().Raw()
	if err != nil {
		return fmt.Errorf("get p2p pubkey data error: %w", err)
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_P2P_PUBKEY_ACK,
		Data: pubkeyData,
	}

	err = swarm.SendWithStream(s, msg)
	if err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	return nil
}

func (swarm *Swarm) handleFetchBlockSignMessage(s network.Stream, data []byte) {
	handle := func(data []byte) ([]byte, error) {
		height, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse height: %w", err)
		}

		swarm.logger.WithField("height", height).Debug("Handle fetching block sign message")

		signed, err := swarm.ledger.GetBlockSign(height)
		if err != nil {
			return nil, fmt.Errorf("get block sign: %w", err)
		}

		return signed, nil
	}

	signed, err := handle(data)
	if err != nil {
		swarm.logger.Errorf("handle fetch-block-sign: %s", err)
		return
	}

	m := model.MerkleWrapperSign{
		Address:   swarm.repo.NodeAddress,
		Signature: signed,
	}

	body, err := m.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal merkle wrapper sign: %s", err.Error())
		return
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_BLOCK_SIGN_ACK,
		Data: body,
	}

	if err := swarm.SendWithStream(s, msg); err != nil {
		swarm.logger.Errorf("send block sign back: %s", err.Error())
	}
}

func (swarm *Swarm) handleGetBlocksPack(s network.Stream, msg *pb.Message) error {
	req := &pb.GetBlocksRequest{}
	if err := req.UnmarshalVT(msg.Data); err != nil {
		return fmt.Errorf("unmarshal get blcoks request error: %w", err)
	}

	res := &pb.GetBlocksResponse{}
	blocks := make([][]byte, 0)
	for i := req.Start; i <= req.End; i++ {
		block, err := swarm.ledger.GetBlock(i)
		if err != nil {
			return fmt.Errorf("get block with height %d from ledger failed: %w", i, err)
		}
		raw, err := block.Marshal()
		if err != nil {
			return fmt.Errorf("marshal block with height %d failed: %w", i, err)
		}
		blocks = append(blocks, raw)
	}
	res.Blocks = blocks
	v, err := res.MarshalVT()
	if err != nil {
		return fmt.Errorf("marshal get blocks response error: %w", err)
	}
	m := &pb.Message{
		Type: pb.Message_GET_BLOCKS_ACK,
		Data: v,
	}

	if err := swarm.SendWithStream(s, m); err != nil {
		return fmt.Errorf("send %s with stream failed: %w", m.String(), err)
	}

	return nil
}
