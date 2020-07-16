package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

// SendTransaction handles transaction sent by the client.
// If the transaction is valid, it will return the transaction hash.
func (cbs *ChainBrokerService) SendTransaction(ctx context.Context, tx *pb.SendTransactionRequest) (*pb.TransactionHashMsg, error) {
	if !cbs.api.Broker().OrderReady() {
		return nil, fmt.Errorf("the system is temporarily unavailable")
	}

	if err := cbs.checkTransaction(tx); err != nil {
		return nil, err
	}

	hash, err := cbs.sendTransaction(tx)
	if err != nil {
		return nil, err
	}

	return &pb.TransactionHashMsg{TxHash: hash}, nil
}

func (cbs *ChainBrokerService) SendView(_ context.Context, tx *pb.SendTransactionRequest) (*pb.Response, error) {
	if err := cbs.checkTransaction(tx); err != nil {
		return nil, err
	}

	result, err := cbs.sendView(tx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (cbs *ChainBrokerService) checkTransaction(tx *pb.SendTransactionRequest) error {
	if tx.Data == nil {
		return fmt.Errorf("tx data can't be empty")
	}

	if tx.Data.Type == pb.TransactionData_NORMAL && tx.Data.Amount == 0 {
		return fmt.Errorf("amount can't be 0 in transfer tx")
	}

	emptyAddress := types.Address{}.Hex()
	if tx.From.Hex() == emptyAddress {
		return fmt.Errorf("from can't be empty")
	}

	if tx.From == tx.To {
		return fmt.Errorf("from can`t be the same as to")
	}

	if tx.To.Hex() == emptyAddress && len(tx.Data.Payload) == 0 {
		return fmt.Errorf("can't deploy empty contract")
	}

	if tx.Timestamp < time.Now().UnixNano()-10*time.Minute.Nanoseconds() ||
		tx.Timestamp > time.Now().UnixNano()+10*time.Minute.Nanoseconds() {
		return fmt.Errorf("timestamp is illegal")
	}

	if tx.Nonce <= 0 {
		return fmt.Errorf("nonce is illegal")
	}

	if len(tx.Signature) == 0 {
		return fmt.Errorf("signature can't be empty")
	}

	return nil
}

func (cbs *ChainBrokerService) sendTransaction(req *pb.SendTransactionRequest) (string, error) {
	tx := &pb.Transaction{
		Version:   req.Version,
		From:      req.From,
		To:        req.To,
		Timestamp: req.Timestamp,
		Data:      req.Data,
		Nonce:     req.Nonce,
		Signature: req.Signature,
		Extra:     req.Extra,
	}

	tx.TransactionHash = tx.Hash()
	err := cbs.api.Broker().HandleTransaction(tx)
	if err != nil {
		return "", err
	}

	return tx.TransactionHash.Hex(), nil
}

func (cbs *ChainBrokerService) sendView(req *pb.SendTransactionRequest) (*pb.Response, error) {
	tx := &pb.Transaction{
		Version:   req.Version,
		From:      req.From,
		To:        req.To,
		Timestamp: req.Timestamp,
		Data:      req.Data,
		Nonce:     req.Nonce,
		Signature: req.Signature,
		Extra:     req.Extra,
	}

	result, err := cbs.api.Broker().HandleView(tx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (cbs *ChainBrokerService) GetTransaction(ctx context.Context, req *pb.TransactionHashMsg) (*pb.GetTransactionResponse, error) {
	hash := types.String2Hash(req.TxHash)
	tx, err := cbs.api.Broker().GetTransaction(hash)
	if err != nil {
		return nil, err
	}

	meta, err := cbs.api.Broker().GetTransactionMeta(hash)
	if err != nil {
		return nil, err
	}

	return &pb.GetTransactionResponse{
		Tx:     tx,
		TxMeta: meta,
	}, nil
}
