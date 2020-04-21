package peermgr

import (
	"crypto/x509"
	"fmt"
	"strconv"

	"github.com/libp2p/go-libp2p-core/network"
	network2 "github.com/libp2p/go-libp2p-core/network"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/cert"
	"github.com/sirupsen/logrus"
)

func (swarm *Swarm) handleMessage(s network.Stream, data []byte) {
	m := &pb.Message{}
	if err := m.Unmarshal(data); err != nil {
		swarm.logger.Error(err)
		return
	}

	handler := func() error {
		switch m.Type {
		case pb.Message_GET_BLOCK:
			return swarm.handleGetBlockPack(s, m)
		case pb.Message_FETCH_CERT:
			return swarm.handleFetchCertMessage(s)
		case pb.Message_CONSENSUS:
			go swarm.orderMessageFeed.Send(events.OrderMessageEvent{Data: m.Data})
		case pb.Message_FETCH_BLOCK_SIGN:
			swarm.handleFetchBlockSignMessage(s, m.Data)
		default:
			swarm.logger.WithField("module", "p2p").Errorf("can't handle msg[type: %v]", m.Type)
			return nil
		}

		return nil
	}

	if err := handler(); err != nil {
		swarm.logger.WithFields(logrus.Fields{
			"error": err,
			"type":  m.Type.String(),
		}).Error("Handle message")
	}
}

func (swarm *Swarm) handleGetBlockPack(s network.Stream, msg *pb.Message) error {
	num, err := strconv.Atoi(string(msg.Data))
	if err != nil {
		return err
	}

	block, err := swarm.ledger.GetBlock(uint64(num))
	if err != nil {
		return err
	}

	v, err := block.Marshal()
	if err != nil {
		return err
	}

	m := &pb.Message{
		Type: pb.Message_GET_BLOCK_ACK,
		Data: v,
	}

	if err := swarm.SendWithStream(s, m); err != nil {
		return err
	}

	return nil
}

func (swarm *Swarm) handleFetchCertMessage(s network.Stream) error {
	certs := &model.CertsMessage{
		AgencyCert: swarm.repo.Certs.AgencyCertData,
		NodeCert:   swarm.repo.Certs.NodeCertData,
	}

	data, err := certs.Marshal()
	if err != nil {
		return fmt.Errorf("marshal certs: %w", err)
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_CERT,
		Data: data,
	}

	err = swarm.SendWithStream(s, msg)
	if err != nil {
		return fmt.Errorf("send msg: %w", err)
	}

	return nil
}

func verifyCerts(nodeCert *x509.Certificate, agencyCert *x509.Certificate, caCert *x509.Certificate) error {
	if err := cert.VerifySign(agencyCert, caCert); err != nil {
		return fmt.Errorf("verify agency cert: %w", err)
	}

	if err := cert.VerifySign(nodeCert, agencyCert); err != nil {
		return fmt.Errorf("verify node cert: %w", err)
	}

	return nil
}

func (swarm *Swarm) handleFetchBlockSignMessage(s network2.Stream, data []byte) {
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
		Address:   swarm.repo.Key.Address,
		Signature: signed,
	}

	body, err := m.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal merkle wrapper sign: %s", err)
		return
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_BLOCK_SIGN_ACK,
		Data: body,
	}

	if err := swarm.SendWithStream(s, msg); err != nil {
		swarm.logger.Errorf("send block sign back: %s", err)
	}
}
