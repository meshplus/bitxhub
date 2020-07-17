package grpc

import (
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
)

type InterchainStatus struct {
	From string `json:"from"`
	To   string `json:"to"`
	Hash string `json:"hash"`
}

// Subscribe implements the interface for client to Subscribe the certain type of event
// arose in bitxhub. This request will establish a websocket conn with client.
func (cbs *ChainBrokerService) Subscribe(req *pb.SubscriptionRequest, server pb.ChainBroker_SubscribeServer) error {
	switch req.Type.String() {
	case pb.SubscriptionRequest_INTERCHAIN_TX.String():
		return cbs.handleInterchainTxSubscription(server)
	case pb.SubscriptionRequest_BLOCK.String():
		return cbs.handleNewBlockSubscription(server)
	case pb.SubscriptionRequest_BLOCK_HEADER.String():
		return cbs.handleBlockHeaderSubscription(server)
	case pb.SubscriptionRequest_INTERCHAIN_TX_WRAPPER.String():
		return cbs.handleInterchainTxWrapperSubscription(server, types.Bytes2Address(req.Extra).String())
	}

	return nil
}

func (cbs *ChainBrokerService) handleNewBlockSubscription(server pb.ChainBroker_SubscribeServer) error {
	blockCh := make(chan events.NewBlockEvent)
	sub := cbs.api.Feed().SubscribeNewBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for ev := range blockCh {
		block := ev.Block
		data, err := block.Marshal()
		if err != nil {
			return err
		}

		if err := server.Send(&pb.Response{
			Data: data,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (cbs *ChainBrokerService) handleBlockHeaderSubscription(server pb.ChainBroker_SubscribeServer) error {
	blockCh := make(chan events.NewBlockEvent)
	sub := cbs.api.Feed().SubscribeNewBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for ev := range blockCh {
		header := ev.Block.BlockHeader
		data, err := header.Marshal()
		if err != nil {
			return err
		}

		if err := server.Send(&pb.Response{
			Data: data,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (cbs *ChainBrokerService) handleInterchainTxSubscription(server pb.ChainBroker_SubscribeServer) error {
	blockCh := make(chan events.NewBlockEvent)
	sub := cbs.api.Feed().SubscribeNewBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			interStatus, err := cbs.interStatus(ev.Block, ev.InterchainMeta)
			if err != nil {
				cbs.logger.Fatal(err)
				return fmt.Errorf("wrap interchain tx status error")
			}
			if interStatus == nil {
				continue
			}
			data, err := json.Marshal(interStatus)
			if err != nil {
				cbs.logger.Fatalf("Marshal new block event: %s", err.Error())
				return fmt.Errorf("marshal interchain tx status failed")
			}
			if err := server.Send(&pb.Response{
				Data: data,
			}); err != nil {
				cbs.logger.Warnf("Send new interchain tx event failed %s", err.Error())
				return fmt.Errorf("send new interchain tx event failed")
			}
		case <-server.Context().Done():
			return nil
		}
	}
}

func (cbs *ChainBrokerService) handleInterchainTxWrapperSubscription(server pb.ChainBroker_SubscribeServer, pid string) error {
	ch, err := cbs.api.Broker().AddPier(pid)
	defer cbs.api.Broker().RemovePier(pid)
	if err != nil {
		return err
	}

	for {
		select {
		case wrapper, ok := <-ch:
			// if channel is unexpected closed, return
			if !ok {
				return nil
			}
			data, err := wrapper.Marshal()
			if err != nil {
				return err
			}

			if err := server.Send(&pb.Response{
				Data: data,
			}); err != nil {
				cbs.logger.Warnf("Send new interchain tx wrapper failed %s", err.Error())
				return fmt.Errorf("send new interchain tx wrapper failed")
			}
		case <-server.Context().Done():
			return nil
		}
	}
}

type interchainEvent struct {
	InterchainTx      []*InterchainStatus `json:"interchain_tx"`
	InterchainReceipt []*InterchainStatus `json:"interchain_receipt"`
	InterchainConfirm []*InterchainStatus `json:"interchain_confirm"`
	InterchainTxCount uint64              `json:"interchain_tx_count"`
	BlockHeight       uint64              `json:"block_height"`
}

func (cbs *ChainBrokerService) interStatus(block *pb.Block, interchainMeta *pb.InterchainMeta) (*interchainEvent, error) {
	// empty interchain tx
	if len(interchainMeta.Counter) == 0 {
		return nil, nil
	}

	meta, err := cbs.api.Chain().Meta()
	if err != nil {
		return nil, err
	}

	ev := &interchainEvent{
		InterchainTx:      make([]*InterchainStatus, 0),
		InterchainReceipt: make([]*InterchainStatus, 0),
		InterchainTxCount: meta.InterchainTxCount,
		BlockHeight:       block.BlockHeader.Number,
	}
	txs := block.Transactions

	for _, indices := range interchainMeta.Counter {
		for _, idx := range indices.Slice {
			ibtp, err := txs[idx].GetIBTP()
			if err != nil {
				return nil, err
			}

			status := &InterchainStatus{
				From: ibtp.From,
				To:   ibtp.To,
				Hash: ibtp.ID(),
			}
			switch ibtp.Type {
			case pb.IBTP_INTERCHAIN:
				ev.InterchainTx = append(ev.InterchainTx, status)
			case pb.IBTP_RECEIPT_SUCCESS:
				ev.InterchainReceipt = append(ev.InterchainReceipt, status)
			case pb.IBTP_RECEIPT_FAILURE:
				ev.InterchainReceipt = append(ev.InterchainReceipt, status)
			}
		}
	}
	return ev, nil
}
