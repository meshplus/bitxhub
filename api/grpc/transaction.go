package grpc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SendTransaction handles transaction sent by the client.
// If the transaction is valid, it will return the transaction hash.
func (cbs *ChainBrokerService) SendTransaction(_ context.Context, tx *pb.BxhTransaction) (*pb.TransactionHashMsg, error) {
	err := cbs.api.Broker().OrderReady()
	if err != nil {
		return nil, status.Newf(codes.Internal, "the system is temporarily unavailable %s", err.Error()).Err()
	}

	if err := cbs.checkTransaction(tx); err != nil {
		return nil, status.Newf(codes.InvalidArgument, "check transaction fail for %s", err.Error()).Err()
	}

	hash, err := cbs.sendTransaction(tx)
	if tx.IsIBTP() {
		cbs.logger.WithFields(logrus.Fields{"from": tx.From, "hash": tx.Hash(), "nonce": tx.Nonce, "to": tx.IBTP.To, "type": tx.IBTP.Type, "index": tx.IBTP.Index}).Debug("get local ibtp")
	} else {
		cbs.logger.WithFields(logrus.Fields{"from": tx.From, "hash": tx.Hash(), "nonce": tx.Nonce}).Debug("get local tx")
	}
	if err != nil {
		return nil, status.Newf(codes.Internal, "internal handling transaction fail %s", err.Error()).Err()
	}

	return &pb.TransactionHashMsg{TxHash: hash}, nil
}

func (cbs *ChainBrokerService) SendTransactions(_ context.Context, txs *pb.MultiTransaction) (*pb.MultiTransactionHash, error) {
	err := cbs.api.Broker().OrderReady()
	if err != nil {
		return nil, status.Newf(codes.Internal, "the system is temporarily unavailable %s", err.Error()).Err()
	}
	hashList := make([]*pb.TransactionHashMsg, 0, len(txs.Txs))
	for _, tx := range txs.Txs {
		if err := cbs.checkTransaction(tx); err != nil {
			cbs.logger.Errorf("api checkTransaction err: nonce is %d", tx.GetNonce())
			return nil, status.Newf(codes.InvalidArgument, "check transaction fail for %s", err.Error()).Err()
		}
		tx.ReceiveTimestamp = time.Now().UnixNano()
		hash, err := cbs.sendTransaction(tx)
		if err != nil {
			return nil, status.Newf(codes.Internal, "internal handling transaction fail %s", err.Error()).Err()
		}
		hashList = append(hashList, &pb.TransactionHashMsg{TxHash: hash})
	}

	return &pb.MultiTransactionHash{TxHashList: hashList}, nil
}

func (cbs *ChainBrokerService) SendView(_ context.Context, tx *pb.BxhTransaction) (*pb.Receipt, error) {
	if err := cbs.checkTransaction(tx); err != nil {
		return nil, fmt.Errorf("check transaction %s failed: %w", tx.GetHash().String(), err)
	}

	result, err := cbs.sendView(tx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (cbs *ChainBrokerService) checkTransaction(tx *pb.BxhTransaction) error {
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

	if tx.Timestamp < time.Now().UnixNano()-30*time.Minute.Nanoseconds() ||
		tx.Timestamp > time.Now().UnixNano()+5*time.Minute.Nanoseconds() {
		return fmt.Errorf("timestamp is illegal")
	}

	if len(tx.Signature) == 0 {
		return fmt.Errorf("signature can't be empty")
	}

	if err := tx.VerifySignature(); err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	tx.TransactionHash = tx.Hash()

	return nil
}

func (cbs *ChainBrokerService) sendTransaction(tx *pb.BxhTransaction) (string, error) {
	err := cbs.api.Broker().HandleTransaction(tx)
	if err != nil {
		return "", fmt.Errorf("handle transaction %s failed: %w", tx.GetHash().String(), err)
	}

	return tx.GetHash().String(), nil
}

func (cbs *ChainBrokerService) sendView(tx *pb.BxhTransaction) (*pb.Receipt, error) {
	result, err := cbs.api.Broker().HandleView(tx)
	if err != nil {
		return nil, fmt.Errorf("handle read-only transaction %s failed: %w", tx.GetHash().String(), err)
	}

	return result, nil
}

func (cbs *ChainBrokerService) GetTransaction(_ context.Context, req *pb.TransactionHashMsg) (*pb.GetTransactionResponse, error) {
	hash := types.NewHashByStr(req.TxHash)
	if hash == nil {
		return nil, fmt.Errorf("invalid format of tx hash for querying transaction")
	}
	tx, err := cbs.api.Broker().GetTransaction(hash)
	if err != nil {
		return nil, fmt.Errorf("get transaction %s failed: %w", hash.String(), err)
	}

	bxhTx, ok := tx.(*pb.BxhTransaction)
	if !ok {
		return nil, fmt.Errorf("cannot get non bxh tx via grpc")
	}
	meta := &pb.TransactionMeta{}
	err = retry.Retry(func(attempt uint) error {
		meta, err = cbs.api.Broker().GetTransactionMeta(hash)
		if err != nil {
			return fmt.Errorf("get transaction %s meta failed: %w", tx.GetHash().Hash, err)
		}
		return nil
	}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond)))

	return &pb.GetTransactionResponse{
		Tx:     bxhTx,
		TxMeta: meta,
	}, err
}

func (cbs *ChainBrokerService) GetTransactionByBlockHashAndIndex(_ context.Context, req *pb.TransactionBlockHashAndIndexMsg) (*pb.GetTransactionResponse, error) {
	blockTyp := pb.GetBlockRequest_HASH
	block, err := cbs.api.Broker().GetBlock(blockTyp.String(), req.BlockHash, true)
	if err != nil {
		return nil, fmt.Errorf("invalid format of blockHash for querying block")
	}
	if int(req.Index) >= len(block.Transactions.Transactions) {
		return nil, fmt.Errorf("invalid format of tx index for querying transaction in a specified block")
	}
	txHash := block.Transactions.Transactions[req.Index].GetHash()

	tx, err := cbs.api.Broker().GetTransaction(txHash)
	if err != nil {
		return nil, fmt.Errorf("get transaction %s failed: %w", txHash.String(), err)
	}

	bxhTx, ok := tx.(*pb.BxhTransaction)
	if !ok {
		return nil, fmt.Errorf("cannot get non bxh tx via grpc")
	}

	meta := &pb.TransactionMeta{}
	err = retry.Retry(func(attempt uint) error {
		meta, err = cbs.api.Broker().GetTransactionMeta(txHash)
		if err != nil {
			return fmt.Errorf("get transaction %s meta failed: %w", tx.GetHash().Hash, err)
		}
		return nil
	}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond)))

	return &pb.GetTransactionResponse{
		Tx:     bxhTx,
		TxMeta: meta,
	}, err
}

func (cbs *ChainBrokerService) GetTransactionByBlockNumberAndIndex(_ context.Context, req *pb.TransactionBlockNumberAndIndexMsg) (*pb.GetTransactionResponse, error) {
	blockTyp := pb.GetBlockRequest_HEIGHT
	block, err := cbs.api.Broker().GetBlock(blockTyp.String(), strconv.FormatUint(req.BlockNumber, 10), true)
	if err != nil {
		return nil, fmt.Errorf("invalid format of blockNumber for querying block")
	}
	if int(req.Index) >= len(block.Transactions.Transactions) {
		return nil, fmt.Errorf("invalid format of tx index for querying transaction in a specified block")
	}
	txHash := block.Transactions.Transactions[req.Index].GetHash()

	tx, err := cbs.api.Broker().GetTransaction(txHash)
	if err != nil {
		return nil, fmt.Errorf("get transaction %s failed: %w", txHash.String(), err)
	}

	bxhTx, ok := tx.(*pb.BxhTransaction)
	if !ok {
		return nil, fmt.Errorf("cannot get non bxh tx via grpc")
	}

	meta := &pb.TransactionMeta{}
	err = retry.Retry(func(attempt uint) error {
		meta, err = cbs.api.Broker().GetTransactionMeta(txHash)
		if err != nil {
			return fmt.Errorf("get transaction %s meta failed: %w", tx.GetHash().Hash, err)
		}
		return nil
	}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond)))

	return &pb.GetTransactionResponse{
		Tx:     bxhTx,
		TxMeta: meta,
	}, err
}

func (cbs *ChainBrokerService) GetPendingTransaction(_ context.Context, req *pb.TransactionHashMsg) (*pb.GetTransactionResponse, error) {
	hash := types.NewHashByStr(req.TxHash)
	if hash == nil {
		return nil, fmt.Errorf("invalid format of tx hash for querying transaction")
	}
	tx := cbs.api.Broker().GetPoolTransaction(hash)

	if tx == nil {
		return nil, fmt.Errorf("tx:%s not in mempool", hash.String())
	}
	bxhTx, ok := tx.(*pb.BxhTransaction)
	if !ok {
		return nil, fmt.Errorf("cannot get non bxh tx via grpc")
	}

	return &pb.GetTransactionResponse{
		Tx: bxhTx,
	}, nil
}
