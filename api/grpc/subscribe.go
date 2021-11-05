package grpc

import (
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/api/jsonrpc/namespaces/eth/filters"
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
		return cbs.handleInterchainTxWrapperSubscription(server, req.Extra)
	case pb.SubscriptionRequest_UNION_INTERCHAIN_TX_WRAPPER.String():
		return cbs.handleInterchainTxWrapperSubscription(server, req.Extra)
	case pb.SubscriptionRequest_EVM_LOG.String():
		return cbs.handleEvmLogSubscription(server, req.Extra)
	}

	return nil
}

func (cbs *ChainBrokerService) handleNewBlockSubscription(server pb.ChainBroker_SubscribeServer) error {
	blockCh := make(chan events.ExecutedEvent)
	sub := cbs.api.Feed().SubscribeNewBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for ev := range blockCh {
		block := ev.Block
		data, err := block.Marshal()
		if err != nil {
			return fmt.Errorf("marshal block error: %w", err)
		}

		if err := server.Send(&pb.Response{
			Data: data,
		}); err != nil {
			return fmt.Errorf("send block failed: %w", err)
		}
	}

	return nil
}

func (cbs *ChainBrokerService) handleBlockHeaderSubscription(server pb.ChainBroker_SubscribeServer) error {
	blockCh := make(chan events.ExecutedEvent)
	sub := cbs.api.Feed().SubscribeNewBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for ev := range blockCh {
		header := ev.Block.BlockHeader
		data, err := header.Marshal()
		if err != nil {
			return fmt.Errorf("marshal block header error: %w", err)
		}

		if err := server.Send(&pb.Response{
			Data: data,
		}); err != nil {
			return fmt.Errorf("send block header failed: %w", err)
		}
	}

	return nil
}

func (cbs *ChainBrokerService) handleInterchainTxSubscription(server pb.ChainBroker_SubscribeServer) error {
	blockCh := make(chan events.ExecutedEvent)
	sub := cbs.api.Feed().SubscribeNewBlockEvent(blockCh)
	defer sub.Unsubscribe()

	for {
		select {
		case ev := <-blockCh:
			interStatus, err := cbs.interStatus(ev.Block, ev.InterchainMeta)
			if err != nil {
				cbs.logger.Fatalf("Wrap interchain tx status error: %s", err.Error())
				return fmt.Errorf("wrap interchain tx status error: %w", err)
			}
			if interStatus == nil {
				continue
			}
			data, err := json.Marshal(interStatus)
			if err != nil {
				cbs.logger.Fatalf("Marshal new block event: %s", err.Error())
				return fmt.Errorf("marshal interchain tx status failed: %w", err)
			}
			if err := server.Send(&pb.Response{
				Data: data,
			}); err != nil {
				cbs.logger.Warnf("Send new interchain tx event failed %s", err.Error())
				return fmt.Errorf("send new interchain tx event failed: %w", err)
			}
		case <-server.Context().Done():
			return nil
		}
	}
}

func (cbs *ChainBrokerService) handleInterchainTxWrapperSubscription(server pb.ChainBroker_SubscribeServer,
	extra []byte) error {
	pierID := string(extra)
	ch, err := cbs.api.Broker().AddPier(pierID)
	defer cbs.api.Broker().RemovePier(pierID)
	if err != nil {
		return fmt.Errorf("add pier %s failed: %w", pierID, err)
	}

	for {
		select {
		case wrapper, ok := <-ch:
			// if channel is unexpected closed, return
			if !ok {
				cbs.logger.Errorf("subs closed")
				return nil
			}
			data, err := wrapper.Marshal()
			if err != nil {
				return fmt.Errorf("marshal interchain tx wrapper error: %w", err)
			}

			if err := server.Send(&pb.Response{
				Data: data,
			}); err != nil {
				cbs.logger.Warnf("Send new interchain tx wrapper failed %s", err.Error())
				return fmt.Errorf("send new interchain tx wrapper failed: %w", err)
			}
		case <-server.Context().Done():
			cbs.logger.Errorf("Server lost connection with pier")
			return nil
		}
	}
}

func (cbs *ChainBrokerService) handleEvmLogSubscription(server pb.ChainBroker_SubscribeServer, query []byte) error {
	filter := filters.FilterQuery{}
	if err := json.Unmarshal(query, &filter); err != nil {
		return fmt.Errorf("unmarshal filter query error: %w", err)
	}

	ch := make(chan []*pb.EvmLog)
	logsSub := cbs.api.Feed().SubscribeLogsEvent(ch)
	defer logsSub.Unsubscribe()

	for {
		select {
		case logs, ok := <-ch:
			// if channel is unexpected closed, return
			if !ok {
				cbs.logger.Errorf("subs closed")
				return nil
			}

			matchedLogs := filters.FilterLogs(logs, filter.FromBlock, filter.ToBlock, filter.Addresses, filter.Topics)
			for _, log := range matchedLogs {
				data, err := log.Marshal()
				if err != nil {
					cbs.logger.Warnf("Marshal evm log failed %s", err.Error())
					return fmt.Errorf("marshal evm log failed: %w", err)
				}

				if err := server.Send(&pb.Response{
					Data: data,
				}); err != nil {
					cbs.logger.Warnf("Send evm log failed %s", err.Error())
					return fmt.Errorf("send evm log failed: %w", err)
				}
			}
		case <-server.Context().Done():
			cbs.logger.Errorf("Server lost connection with pier")
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

type SubscriptionKey struct {
	PierID      string `json:"pier_id"`
	AppchainDID string `json:"appchain_did"`
}

func (cbs *ChainBrokerService) interStatus(block *pb.Block, interchainMeta *pb.InterchainMeta) (*interchainEvent, error) {
	// empty interchain tx
	if len(interchainMeta.Counter) == 0 {
		return nil, nil
	}

	meta, err := cbs.api.Chain().Meta()
	if err != nil {
		return nil, fmt.Errorf("get chain meta from ledger failed: %w", err)
	}

	ev := &interchainEvent{
		InterchainTx:      make([]*InterchainStatus, 0),
		InterchainReceipt: make([]*InterchainStatus, 0),
		InterchainTxCount: meta.InterchainTxCount,
		BlockHeight:       block.BlockHeader.Number,
	}
	txs := block.Transactions.Transactions

	for _, indices := range interchainMeta.Counter {
		for _, vi := range indices.Slice {
			ibtp := txs[vi.Index].GetIBTP()
			if ibtp == nil {
				return nil, fmt.Errorf("ibtp is empty")
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
