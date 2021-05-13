package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

// SendTransaction handles transaction sent by the client.
// If the transaction is valid, it will return the transaction hash.
func (cbs *ChainBrokerService) SendTransaction(ctx context.Context, tx *pb.Transaction) (*pb.TransactionHashMsg, error) {
	err := cbs.api.Broker().OrderReady()
	if err != nil {
		return nil, fmt.Errorf("the system is temporarily unavailable, err: %s", err.Error())
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

func (cbs *ChainBrokerService) SendView(_ context.Context, tx *pb.Transaction) (*pb.Receipt, error) {
	if err := cbs.checkTransaction(tx); err != nil {
		return nil, err
	}

	result, err := cbs.sendView(tx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (cbs *ChainBrokerService) checkTransaction(tx *pb.Transaction) error {
	if tx.Payload == nil && tx.IBTP == nil {
		tx.Payload = []byte{}
	}
	if tx.From == nil {
		return fmt.Errorf("tx from address is nil")
	}
	if tx.To == nil {
		return fmt.Errorf("tx to address is nil")
	}

	emptyAddress := &types.Address{}
	if tx.From.String() == emptyAddress.String() {
		return fmt.Errorf("from can't be empty")
	}

	if tx.From.String() == tx.To.String() {
		return fmt.Errorf("from can`t be the same as to")
	}

	if tx.To.String() == emptyAddress.String() && len(tx.Payload) == 0 {
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

func (cbs *ChainBrokerService) sendTransaction(tx *pb.Transaction) (string, error) {
	tx.TransactionHash = tx.Hash()
	ok, _ := asym.Verify(crypto.Secp256k1, tx.Signature, tx.SignHash().Bytes(), *tx.From)
	if !ok {
		return "", fmt.Errorf("invalid signature")
	}
	err := cbs.api.Broker().HandleTransaction(tx)
	if err != nil {
		return "", err
	}

	return tx.TransactionHash.String(), nil
}

func (cbs *ChainBrokerService) sendView(tx *pb.Transaction) (*pb.Receipt, error) {
	result, err := cbs.api.Broker().HandleView(tx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (cbs *ChainBrokerService) GetTransaction(ctx context.Context, req *pb.TransactionHashMsg) (*pb.GetTransactionResponse, error) {
	hash := types.NewHashByStr(req.TxHash)
	if hash == nil {
		return nil, fmt.Errorf("invalid format of tx hash for querying transaction")
	}
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
