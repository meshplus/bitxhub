package peermgr

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	contracts2 "github.com/meshplus/bitxhub-core/eth-contracts/interchain-contracts"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/model"
	"github.com/meshplus/bitxhub/internal/model/events"
	network "github.com/meshplus/go-lightp2p"
	network_pb "github.com/meshplus/go-lightp2p/pb"
	solsha3 "github.com/miguelmota/go-solidity-sha3"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

const SubscribeResponse = "Successfully subscribe"

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
		case pb.Message_GET_BLOCK_HEADERS:
			return swarm.handleGetBlockHeadersPack(s, m)
		case pb.Message_GET_BLOCKS:
			return swarm.handleGetBlocksPack(s, m)
		case pb.Message_FETCH_CERT:
			return swarm.handleFetchCertMessage(s)
		case pb.Message_CONSENSUS:
			go swarm.orderMessageFeed.Send(events.OrderMessageEvent{Data: m.Data})
		case pb.Message_FETCH_BLOCK_SIGN:
			swarm.handleFetchBlockSignMessage(s, m.Data)
		case pb.Message_FETCH_ASSET_EXCHANGE_SIGN:
			swarm.handleFetchAssetExchangeSignMessage(s, m.Data)
		case pb.Message_FETCH_IBTP_SIGN:
			swarm.handleFetchIBTPSignMessage(s, m.Data)
		case pb.Message_FETCH_BURN_SIGN:
			swarm.handleFetchBurnSignMessage(s, m.Data)
		case pb.Message_CHECK_MASTER_PIER:
			swarm.handleAskPierMaster(s, m.Data)
		case pb.Message_CHECK_MASTER_PIER_ACK:
			swarm.handleReplyPierMaster(s, m.Data)
		case pb.Message_PIER_SEND_TRANSACTION:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierSendTransaction(s, payload.Data)
		case pb.Message_PIER_GET_TRANSACTION:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierGetTransaction(s, payload.Data)
		case pb.Message_PIER_SEND_VIEW:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierSendView(s, payload.Data)
		case pb.Message_PIER_GET_RECEIPT:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierGetReceipt(s, payload.Data)
		case pb.Message_PIER_GET_MULTI_SIGNS:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierGetMultiSigns(s, payload.Data)
		case pb.Message_PIER_GET_CHAIN_META:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierGetChainMeta(s, payload.Data)
		case pb.Message_PIER_GET_PENDING_NONCE_BY_ACCOUNT:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierGetPendingNonceByAccount(s, payload.Data)
		case pb.Message_PIER_SUBSCRIBE_BLOCK_HEADER, pb.Message_PIER_SUBSCRIBE_INTERCHAIN_TX_WRAPPERS:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			go swarm.handlePierSubscribe(s, payload.Data, m.From)
		case pb.Message_PIER_GET_INTERCHAIN_TX_WRAPPERS:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			go swarm.handlePierGetInterchainTxWrappers(s, payload.Data)
		case pb.Message_PIER_GET_BLOCK_HEADER:
			payload := &pb.PierPayload{}
			if err := payload.Unmarshal(m.Data); err != nil {
				swarm.logger.WithFields(logrus.Fields{"data": string(m.Data), "err": err.Error()}).Errorf("unmarshal pier payload error")
				return nil
			}
			swarm.handlePierGetBlockHeader(s, payload.Data)
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

func (swarm *Swarm) handleGetBlockHeadersPack(s network.Stream, msg *pb.Message) error {
	req := &pb.GetBlockHeadersRequest{}
	if err := req.Unmarshal(msg.Data); err != nil {
		return err
	}

	res := &pb.GetBlockHeadersResponse{}
	blockHeaders := make([]*pb.BlockHeader, 0)
	for i := req.Start; i <= req.End; i++ {
		block, err := swarm.ledger.GetBlock(i)
		if err != nil {
			return err
		}
		blockHeaders = append(blockHeaders, block.BlockHeader)
	}
	res.BlockHeaders = blockHeaders
	v, err := res.Marshal()
	if err != nil {
		return err
	}
	m := &pb.Message{
		Type: pb.Message_GET_BLOCK_HEADERS_ACK,
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
		Type: pb.Message_FETCH_CERT_ACK,
		Data: data,
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

func (swarm *Swarm) handleFetchBurnSignMessage(s network.Stream, data []byte) {
	handle := func(hash string) (string, []byte, error) {
		receipt, err := swarm.ledger.GetReceipt(types.NewHashByStr(hash))
		if err != nil {
			return "", nil, fmt.Errorf("cannot find receipt with hash %s", hash)
		}
		ok, interchainSwapAddr := swarm.ledger.GetState(constant.EthHeaderMgrContractAddr.Address(), []byte(contracts.InterchainSwapAddrKey))
		if !ok {
			return "", nil, fmt.Errorf("cannot find interchainswap contract")
		}

		addr := &contracts.ContractAddr{}
		err = json.Unmarshal(interchainSwapAddr, &addr)
		if err != nil {
			return "", nil, err
		}
		var burn *contracts2.InterchainSwapBurn
		for _, log := range receipt.GetEvmLogs() {
			if !strings.EqualFold(log.Address.String(), addr.Addr) {
				continue
			}

			if log.Removed {
				continue
			}
			interchainSwap, err := contracts2.NewInterchainSwap(common.Address{}, nil)
			if err != nil {
				continue
			}
			data, err := json.Marshal(log)
			if err != nil {
				continue
			}
			ethLog := &types2.Log{}
			err = json.Unmarshal(data, &ethLog)
			if err != nil {
				continue
			}
			burn, err = interchainSwap.ParseBurn(*ethLog)
			if err != nil {
				continue
			}
		}

		if burn == nil {
			return "", nil, fmt.Errorf("not found burn log:%v", receipt.TxHash.Hash)
		}

		//abi.encodePacked
		abiHash := solsha3.SoliditySHA3(
			solsha3.Address(burn.AppToken),
			solsha3.Address(burn.Burner),
			solsha3.Address(burn.Recipient),
			solsha3.Uint256(burn.Amount),
			solsha3.String(string(data)),
		)
		prefixedHash := crypto.Keccak256Hash(
			[]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%v", len(abiHash))),
			abiHash,
		)
		key := swarm.repo.Key
		sign, err := key.PrivKey.Sign(prefixedHash[:])
		if err != nil {
			return "", nil, fmt.Errorf("bitxhub sign: %w", err)
		}
		return key.Address, sign, nil
	}

	address, signed, err := handle(string(data))
	if err != nil {
		swarm.logger.Errorf("handle fetch-burn-sign: %s", err)
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
		Type: pb.Message_FETCH_BURN_SIGN_ACK,
		Data: body,
	}

	if err := swarm.SendWithStream(s, msg); err != nil {
		swarm.logger.Errorf("send burn sign back: %s", err)
	}
}

func (swarm *Swarm) handleGetBlocksPack(s network.Stream, msg *pb.Message) error {
	req := &pb.GetBlocksRequest{}
	if err := req.Unmarshal(msg.Data); err != nil {
		return err
	}

	res := &pb.GetBlocksResponse{}
	blocks := make([]*pb.Block, 0)
	for i := req.Start; i <= req.End; i++ {
		block, err := swarm.ledger.GetBlock(i)
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
	}
	res.Blocks = blocks
	v, err := res.Marshal()
	if err != nil {
		return err
	}
	m := &pb.Message{
		Type: pb.Message_GET_BLOCKS_ACK,
		Data: v,
	}

	if err := swarm.SendWithStream(s, m); err != nil {
		return err
	}

	return nil
}

func (swarm *Swarm) handleAskPierMaster(s network.Stream, data []byte) {
	address := string(data)
	resp := &pb.CheckPierResponse{}
	if swarm.piers.pierChan.checkAddress(address) {
		resp.Status = pb.CheckPierResponse_HAS_MASTER
	} else {
		if !swarm.piers.pierMap.hasPier(address) {
			return
		}
		if swarm.piers.pierMap.checkMaster(address) {
			resp.Status = pb.CheckPierResponse_HAS_MASTER
		} else {
			swarm.piers.pierMap.rmMaster(address)
			return
		}
	}
	resp.Address = address
	msgData, err := resp.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal ask pier master response: %s", err)
		return
	}
	message := &pb.Message{
		Data: msgData,
		Type: pb.Message_CHECK_MASTER_PIER_ACK,
	}
	msg, err := message.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response message: %s", err)
		return
	}

	msg1 := &network_pb.Message{
		Data: msg,
		Msg:  network_pb.Message_DATA_ASYNC,
	}

	addr := &peer.AddrInfo{
		ID:    s.RemotePeerID(),
		Addrs: []ma.Multiaddr{s.RemotePeerAddr()},
	}
	if err := swarm.p2p.AsyncSend(addr, msg1); err != nil {
		swarm.logger.Errorf("send response: %s", err)
		return
	}
}

func (swarm *Swarm) handleReplyPierMaster(s network.Stream, data []byte) {
	resp := &pb.CheckPierResponse{}
	err := resp.Unmarshal(data)
	if err != nil {
		swarm.logger.Errorf("unmarshal response: %s", err)
		return
	}
	swarm.piers.pierChan.writeChan(resp)
}

func (swarm *Swarm) handlePierSendTransaction(s network.Stream, data []byte) {
	bxhTx := &pb.BxhTransaction{}
	if err := bxhTx.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal BxhTransaction: %v", err)
		return
	}
	ret, err := swarm.grpcClient.SendTransaction(swarm.ctx, bxhTx)
	if err != nil {
		swarm.logger.Errorf("send transaction: %v", err)
		return
	}
	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

func (swarm *Swarm) handlePierGetTransaction(s network.Stream, data []byte) {
	txHashMsg := &pb.TransactionHashMsg{}
	if err := txHashMsg.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal TransactionHashMsg: %v", err)
		return
	}
	ret, err := swarm.grpcClient.GetTransaction(swarm.ctx, txHashMsg)
	if err != nil {
		swarm.logger.Errorf("get transaction: %v", err)
		return
	}
	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

func (swarm *Swarm) handlePierSendView(s network.Stream, data []byte) {
	bxhTx := &pb.BxhTransaction{}
	if err := bxhTx.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal BxhTransaction: %v", err)
		return
	}
	td := &pb.TransactionData{}
	err := td.Unmarshal(bxhTx.Payload)
	if err != nil {
		swarm.logger.Errorf("bxhTx unmarshal payload err")
	}

	ret, err := swarm.grpcClient.SendView(swarm.ctx, bxhTx)
	if err != nil {
		swarm.logger.Errorf("send view: %v", err)
		return
	}
	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

func (swarm *Swarm) handlePierGetReceipt(s network.Stream, data []byte) {
	txHashMsg := &pb.TransactionHashMsg{}
	if err := txHashMsg.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal TransactionHashMsg: %v", err)
		return
	}
	ret, err := swarm.grpcClient.GetReceipt(swarm.ctx, txHashMsg)
	if err != nil {
		swarm.logger.Errorf("get receipt: %v", err)
		return
	}
	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

func (swarm *Swarm) handlePierGetMultiSigns(s network.Stream, data []byte) {
	req := &pb.GetMultiSignsRequest{}
	if err := req.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal GetMultiSignsRequest: %v", err)
		return
	}
	ret, err := swarm.grpcClient.GetMultiSigns(swarm.ctx, req)
	if err != nil {
		swarm.logger.Errorf("get multi signs: %v", err)
		return
	}
	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

func (swarm *Swarm) handlePierGetChainMeta(s network.Stream, data []byte) {
	req := &pb.Request{}
	if err := req.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal Request: %v", err)
		return
	}
	ret, err := swarm.grpcClient.GetChainMeta(swarm.ctx, req)
	if err != nil {
		swarm.logger.Errorf("get chain meta: %v", err)
		return
	}
	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

func (swarm *Swarm) handlePierGetPendingNonceByAccount(s network.Stream, data []byte) {
	req := &pb.Address{}
	if err := req.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal Address: %v", err)
		return
	}
	ret, err := swarm.grpcClient.GetPendingNonceByAccount(swarm.ctx, req)
	if err != nil {
		swarm.logger.Errorf("get pending nonce by account: %v", err)
		return
	}
	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

func (swarm *Swarm) handlePierSubscribe(s network.Stream, data []byte, from string) {
	if err := s.AsyncSend([]byte(fmt.Sprintf("bxhNodeId: %d %s", swarm.localID, SubscribeResponse))); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}

	req := &pb.SubscriptionRequest{}
	if err := req.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal SubscriptionRequest: %v", err)
		return
	}

	// 1. 调用grpc订阅接口拿到grpcClient流
	stream, err := swarm.grpcClient.Subscribe(swarm.ctx, req)
	if err != nil || stream == nil {
		swarm.logger.Errorf("subscribe: %v", err)
		return
	}

	// 2. 起协程接收数据：从流中获取数据，放进subResCh
	subResCh := make(chan *pb.Response)
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				swarm.logger.Errorf("get response from grpc client stream: %v", err)
				return
			}
			subResCh <- resp
		}
	}()

	// 3. 发送channel中的数据
	toAddr := swarm.getPierMultiAddr(from)
	sendType := getSubMsgType(req.Type)
	for {
		select {
		case info := <-subResCh:
			swarm.logger.WithFields(logrus.Fields{
				"to": toAddr,
				//"info": info,
				"type": req.Type,
			}).Infof("subscirbe info get")

			sendData, err := info.Marshal()
			if err != nil {
				swarm.logger.Errorf("marshal subscribe response: %v", err)
				return
			}

			m := &pb.Message{
				Type: sendType,
				Data: sendData,
			}
			mData, err := m.Marshal()
			if err != nil {
				swarm.logger.Errorf("marshal subscribe response ack msg: %v", err)
				return
			}

			resp, err := swarm.p2p.SendByMultiAddr(toAddr, mData)
			if err != nil {
				swarm.logger.Errorf("send subscribe response ack by multi addr: %v", err)
				return
			}
			if !strings.Contains(string(resp), SubscribeResponse) {
				err = fmt.Errorf("pier is not recieve subscribe ack")
				swarm.logger.Errorf("swarm SendByMultiAddr subscrbe err: %w", err)
				return
			}
		case <-swarm.ctx.Done():
			close(subResCh)
			return
		}
	}
}

func (swarm *Swarm) handlePierGetInterchainTxWrappers(s network.Stream, data []byte) {
	req := &pb.GetInterchainTxWrappersRequest{}
	if err := req.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal GetInterchainTxWrappersRequest: %v", err)
		return
	}

	// 1. 根据grpcClient获取待发送数据
	ch := make(chan *pb.InterchainTxWrappers, req.End-req.Begin+1)
	swarm.getInterchainTxWrappers(req, ch)

	// 2. 发数据
	multiTxWrappers := &pb.MultiInterchainTxWrappers{}
	for {
		select {
		case info, ok := <-ch:
			if !ok {
				sendData, err := multiTxWrappers.Marshal()
				if err != nil {
					swarm.logger.Errorf("marshal info: %v", err)
					return
				}

				if err := s.AsyncSend(sendData); err != nil {
					swarm.logger.Errorf("send with stream: %v", err)
					return
				}

				return
			}

			multiTxWrappers.MultiWrappers = append(multiTxWrappers.MultiWrappers, info)
		case <-swarm.ctx.Done():
			break
		}
	}
}

func (swarm *Swarm) getInterchainTxWrappers(req *pb.GetInterchainTxWrappersRequest, ch chan *pb.InterchainTxWrappers) {
	defer close(ch)

	stream, err := swarm.grpcClient.GetInterchainTxWrappers(swarm.ctx, req)
	if err != nil || stream == nil {
		swarm.logger.Errorf("get interchain tx wrappers: %v", err)
		return
	}

	for {
		info, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			swarm.logger.Errorf("get info from grpc client stream: %v", err)
			return
		}
		ch <- info
	}
}

func (swarm *Swarm) handlePierGetBlockHeader(s network.Stream, data []byte) {
	req := &pb.GetBlockHeadersRequest{}
	if err := req.Unmarshal(data); err != nil {
		swarm.logger.Errorf("unmarshal GetBlockHeadersRequest: %v", err)
		return
	}

	ret, err := swarm.grpcClient.GetBlockHeaders(swarm.ctx, req)
	if err != nil {
		swarm.logger.Errorf("get block headers: %v", err)
		return
	}

	retData, err := ret.Marshal()
	if err != nil {
		swarm.logger.Errorf("marshal response: %v", err)
		return
	}

	if err := s.AsyncSend(retData); err != nil {
		swarm.logger.Errorf("send with stream: %v", err)
		return
	}
}

//func (swarm *Swarm) connectByMultiAddr(addr string) {
//	if _, ok := swarm.connectedPiers.Load(addr); !ok {
//		if err := retry.Retry(func(attempt uint) error {
//			if err := swarm.p2p.ConnectByMultiAddr(addr); err != nil {
//				swarm.logger.WithFields(logrus.Fields{
//					"addr":  addr,
//					"error": err,
//				}).Error("Connect failed")
//				return err
//			}
//
//			swarm.logger.WithFields(logrus.Fields{
//				"addr": addr,
//			}).Info("Connect successfully")
//
//			swarm.connectedPiers.Store(addr, struct{}{})
//			return nil
//		},
//			strategy.Wait(1*time.Second),
//		); err != nil {
//			swarm.logger.Error(err)
//		}
//	}
//}

func (swarm *Swarm) getPierMultiAddr(addrStr string) string {
	addrs := strings.Split(strings.Replace(addrStr, " ", "", -1), ",")
	return fmt.Sprintf("/peer%s/netgap%s/peer%s/peer%s", swarm.repo.Config.Pangolin.Addr, swarm.repo.Config.Pangolin.Addr, addrs[1], addrs[0])
}

func getSubMsgType(request_type pb.SubscriptionRequest_Type) pb.Message_Type {
	switch request_type {
	case pb.SubscriptionRequest_BLOCK_HEADER:
		return pb.Message_PIER_SUBSCRIBE_BLOCK_HEADER_ACK
	case pb.SubscriptionRequest_INTERCHAIN_TX_WRAPPER:
		return pb.Message_PIER_SUBSCRIBE_INTERCHAIN_TX_WRAPPERS_ACK
	default:
		panic(fmt.Errorf("unsupported type: %v", request_type))
	}
}
