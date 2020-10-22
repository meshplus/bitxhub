package peermgr

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/model"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/meshplus/bitxhub/pkg/cert"
	network "github.com/meshplus/go-lightp2p"
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
		case pb.Message_FETCH_ASSET_EXCHANEG_SIGN:
			swarm.handleFetchAssetExchangeSignMessage(s, m.Data)
		case pb.Message_FETCH_IBTP_SIGN:
			swarm.handleFetchIBTPSignMessage(s, m.Data)
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

func (swarm *Swarm) handleFetchAssetExchangeSignMessage(s network.Stream, data []byte) {
	handle := func(id string) (string, []byte, error) {
		swarm.logger.WithField("asset exchange id", id).Debug("Handle fetching asset exchange sign message")

		ok, record := swarm.ledger.GetState(constant.AssetExchangeContractAddr.Address(), []byte(contracts.AssetExchangeKey(id)))
		if !ok {
			return "", nil, fmt.Errorf("cannot find asset exchange record with id %s", id)
		}

		aer := contracts.AssetExchangeRecord{}
		if err := json.Unmarshal(record, &aer); err != nil {
			return "", nil, err
		}

		hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", id, aer.Status)))
		key := swarm.repo.Key
		sign, err := key.PrivKey.Sign(hash[:])
		if err != nil {
			return "", nil, fmt.Errorf("fetch asset exchange sign: %w", err)
		}

		return key.Address, sign, nil
	}

	address, signed, err := handle(string(data))
	if err != nil {
		swarm.logger.Errorf("handle fetch-asset-exchange-sign: %s", err)
		return
	}

	m := model.MerkleWrapperSign{
		Address:   address,
		Signature: signed,
	}

	body, err := m.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal merkle wrapper sign: %s", err)
		return
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_ASSET_EXCHANGE_SIGN_ACK,
		Data: body,
	}

	if err := swarm.SendWithStream(s, msg); err != nil {
		swarm.logger.Errorf("send asset exchange sign back: %s", err)
	}
}

func (swarm *Swarm) handleFetchIBTPSignMessage(s network.Stream, data []byte) {
	handle := func(id string) (string, []byte, error) {
		hash := sha256.Sum256([]byte(id))
		key := swarm.repo.Key
		sign, err := key.PrivKey.Sign(hash[:])
		if err != nil {
			return "", nil, fmt.Errorf("fetch ibtp sign: %w", err)
		}

		return key.Address, sign, nil
	}

	address, signed, err := handle(string(data))
	if err != nil {
		swarm.logger.Errorf("handle fetch-ibtp-sign: %s", err)
		return
	}

	m := model.MerkleWrapperSign{
		Address:   address,
		Signature: signed,
	}

	body, err := m.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal merkle wrapper sign: %s", err)
		return
	}

	msg := &pb.Message{
		Type: pb.Message_FETCH_IBTP_SIGN_ACK,
		Data: body,
	}

	if err := swarm.SendWithStream(s, msg); err != nil {
		swarm.logger.Errorf("send asset exchange sign back: %s", err)
	}
}
