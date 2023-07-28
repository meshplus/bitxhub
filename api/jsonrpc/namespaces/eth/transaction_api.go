package eth

import (
	"context"
	"fmt"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types3 "github.com/ethereum/go-ethereum/core/types"
	"github.com/meshplus/bitxhub-kit/types"
	rpctypes "github.com/meshplus/bitxhub/api/jsonrpc/types"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

// TransactionAPI provide apis to get and create transaction
type TransactionAPI struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *repo.Config
	api    api.CoreAPI
	logger logrus.FieldLogger
}

func NewTransactionAPI(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) *TransactionAPI {
	ctx, cancel := context.WithCancel(context.Background())
	return &TransactionAPI{ctx: ctx, cancel: cancel, config: config, api: api, logger: logger}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block identified by its height.
func (api *TransactionAPI) GetBlockTransactionCountByNumber(blockNum rpctypes.BlockNumber) *hexutil.Uint {
	api.logger.Debugf("eth_getBlockTransactionCountByNumber, block number: %d", blockNum)
	if blockNum == rpctypes.PendingBlockNumber || blockNum == rpctypes.LatestBlockNumber {
		meta, _ := api.api.Chain().Meta()
		blockNum = rpctypes.BlockNumber(meta.Height)
	}

	block, err := api.api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", blockNum))
	if err != nil {
		api.logger.Debugf("eth api GetBlockTransactionCountByNumber err:%s", err)
		return nil
	}

	count := uint(len(block.Transactions))

	return (*hexutil.Uint)(&count)
}

// GetBlockTransactionCountByHash returns the number of transactions in the block identified by hash.
func (api *TransactionAPI) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	api.logger.Debugf("eth_getBlockTransactionCountByHash, hash: %s", hash.String())

	block, err := api.api.Broker().GetBlock("HASH", hash.String())
	if err != nil {
		api.logger.Debugf("eth api GetBlockTransactionCountByHash err:%s", err)
		return nil
	}

	count := uint(len(block.Transactions))

	return (*hexutil.Uint)(&count)
}

// GetTransactionByBlockNumberAndIndex returns the transaction identified by number and index.
func (api *TransactionAPI) GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	api.logger.Debugf("eth_getTransactionByBlockNumberAndIndex, number: %d, index: %d", blockNum, idx)

	height := uint64(0)

	switch blockNum {
	//Latest and Pending type return current block height
	case rpctypes.LatestBlockNumber, rpctypes.PendingBlockNumber:
		meta, err := api.api.Chain().Meta()
		if err != nil {
			return nil, err
		}
		height = meta.Height
	default:
		height = uint64(blockNum.Int64())
	}

	return getTxByBlockInfoAndIndex(api.api, "HEIGHT", fmt.Sprintf("%d", height), idx)
}

// GetTransactionByBlockHashAndIndex returns the transaction identified by hash and index.
func (api *TransactionAPI) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	api.logger.Debugf("eth_getTransactionByHashAndIndex, hash: %s, index: %d", hash.String(), idx)

	return getTxByBlockInfoAndIndex(api.api, "HASH", hash.String(), idx)
}

// GetTransactionCount returns the number of transactions at the given address, blockNum is ignored.
func (api *TransactionAPI) GetTransactionCount(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Uint64, error) {
	api.logger.Debugf("eth_getTransactionCount, address: %s", address)

	if blockNumber, ok := blockNrOrHash.Number(); ok && blockNumber == rpctypes.PendingBlockNumber {
		nonce := api.api.Broker().GetPendingNonceByAccount(address.String())
		return (*hexutil.Uint64)(&nonce), nil
	}

	stateLedger, err := getStateLedgerAt(api.api)
	if err != nil {
		return nil, err
	}

	nonce := stateLedger.GetNonce(types.NewAddress(address.Bytes()))

	return (*hexutil.Uint64)(&nonce), nil
}

// GetTransactionByHash returns the transaction identified by hash.
func (api *TransactionAPI) GetTransactionByHash(hash common.Hash) (*rpctypes.RPCTransaction, error) {
	api.logger.Debugf("eth_getTransactionByHash, hash: %s", hash.String())

	ethTx, meta, err := getEthTransactionByHash(api.api, api.logger, types.NewHash(hash.Bytes()))
	if err != nil {
		return nil, err
	}

	return newRPCTransaction(ethTx, common.BytesToHash(meta.BlockHash.Bytes()), meta.BlockHeight, meta.Index), nil
}

// GetTransactionReceipt returns the transaction receipt identified by hash.
func (api *TransactionAPI) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	api.logger.Debugf("eth_getTransactionReceipt, hash: %s", hash.String())

	txHash := types.NewHash(hash.Bytes())
	//tx, meta, err := getEthTransactionByHash(api.config, api.api, api.logger, txHash)
	tx, err := api.api.Broker().GetTransaction(txHash)
	if err != nil {
		return nil, nil
	}

	meta, err := api.api.Broker().GetTransactionMeta(txHash)
	if err != nil {
		return nil, fmt.Errorf("get tx meta from ledger: %w", err)
	}
	if err != nil {
		api.logger.Debugf("no tx found for hash %s", txHash.String())
		return nil, err
	}

	receipt, err := api.api.Broker().GetReceipt(txHash)
	if err != nil {
		api.logger.Debugf("no receipt found for tx %s", txHash.String())
		return nil, err
	}

	block, err := api.api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", meta.BlockHeight))
	if err != nil {
		api.logger.Debugf("no block found for height %d", meta.BlockHeight)
		return nil, err
	}

	cumulativeGasUsed, err := getBlockCumulativeGas(api.api, block, meta.Index)
	if err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"type":              hexutil.Uint(tx.GetType()),
		"cumulativeGasUsed": hexutil.Uint64(cumulativeGasUsed),
		"transactionHash":   hash,
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"blockHash":         common.BytesToHash(meta.BlockHash.Bytes()),
		"blockNumber":       hexutil.Uint64(meta.BlockHeight),
		"transactionIndex":  hexutil.Uint64(meta.Index),
		"from":              common.BytesToAddress(tx.GetFrom().Bytes()),
	}
	if receipt.Bloom == nil {
		fields["logsBloom"] = types.Bloom{}
	} else {
		fields["logsBloom"] = *receipt.Bloom
	}
	ethLogs := make([]*types3.Log, 0)
	for _, log := range receipt.EvmLogs {
		ethLog := &types3.Log{
			Address: log.Address.ETHAddress(),
			Topics: lo.Map(log.Topics, func(item *types.Hash, index int) common.Hash {
				return item.ETHHash()
			}),
			Data:        log.Data,
			BlockNumber: log.BlockNumber,
			TxHash:      log.TransactionHash.ETHHash(),
			TxIndex:     uint(log.TransactionIndex),
			BlockHash:   log.BlockHash.ETHHash(),
			Index:       uint(log.LogIndex),
			Removed:     log.Removed,
		}
		ethLogs = append(ethLogs, ethLog)
	}
	fields["logs"] = ethLogs

	if receipt.Status == types.ReceiptSUCCESS {
		fields["status"] = hexutil.Uint(1)
	} else {
		fields["status"] = hexutil.Uint(0)
		// TODO: need process
		if receipt.Ret != nil {

		}
	}

	if receipt.ContractAddress != nil {
		fields["contractAddress"] = common.BytesToAddress(receipt.ContractAddress.Bytes())
	}

	if tx.GetTo() != nil {
		fields["to"] = common.BytesToAddress(tx.GetTo().Bytes())
	}

	api.logger.Debugf("eth_getTransactionReceipt: %v", fields)

	return fields, nil
}

// SendRawTransaction send a raw Ethereum transaction.
func (api *TransactionAPI) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	api.logger.Debugf("eth_sendRawTransaction, data: %s", data.String())

	tx := &types.Transaction{}
	if err := tx.Unmarshal(data); err != nil {
		return [32]byte{}, err
	}
	api.logger.Debugf("get new eth tx: %s", tx.GetHash().String())

	if tx.GetFrom() == nil {
		return [32]byte{}, fmt.Errorf("verify signature failed")
	}

	err := api.api.Broker().OrderReady()
	if err != nil {
		return [32]byte{}, fmt.Errorf("the system is temporarily unavailable %s", err.Error())
	}

	if err := checkTransaction(api.logger, tx); err != nil {
		return [32]byte{}, fmt.Errorf("check transaction fail for %s", err.Error())
	}

	return sendTransaction(api.api, tx)
}

func getTxByBlockInfoAndIndex(api api.CoreAPI, mode string, key string, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	block, err := api.Broker().GetBlock(mode, key)
	if err != nil {
		return nil, err
	}

	if int(idx) >= len(block.Transactions) {
		return nil, fmt.Errorf("index beyond block transactions' size")
	}

	tx := block.Transactions[idx]

	meta, err := api.Broker().GetTransactionMeta(tx.GetHash())
	if err != nil {
		return nil, err
	}

	return newRPCTransaction(tx, common.BytesToHash(meta.BlockHash.Bytes()), meta.BlockHeight, meta.Index), nil
}

func checkTransaction(logger logrus.FieldLogger, tx *types.Transaction) error {
	if tx.GetFrom() == nil {
		return fmt.Errorf("tx from address is nil")
	}
	logger.Debugf("from address: %s, nonce: %d", tx.GetFrom().String(), tx.GetNonce())

	emptyAddress := &types.Address{}
	if tx.GetFrom().String() == emptyAddress.String() {
		return fmt.Errorf("from can't be empty")
	}

	if tx.GetTo() == nil {
		if len(tx.GetPayload()) == 0 {
			return fmt.Errorf("can't deploy empty contract")
		}
	} else {
		if tx.GetFrom().String() == tx.GetTo().String() {
			return fmt.Errorf("from can`t be the same as to")
		}
	}
	if tx.GetTimeStamp() < time.Now().Unix()-10*60 ||
		tx.GetTimeStamp() > time.Now().Unix()+10*60 {
		return fmt.Errorf("timestamp is illegal")
	}

	if len(tx.GetSignature()) == 0 {
		return fmt.Errorf("signature can't be empty")
	}

	return nil
}

func sendTransaction(api api.CoreAPI, tx *types.Transaction) (common.Hash, error) {
	if err := tx.VerifySignature(); err != nil {
		return [32]byte{}, err
	}
	err := api.Broker().HandleTransaction(tx)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.GetHash().ETHHash(), nil
}

func getEthTransactionByHash(api api.CoreAPI, logger logrus.FieldLogger, hash *types.Hash) (*types.Transaction, *types.TransactionMeta, error) {
	var err error
	meta := &types.TransactionMeta{}

	tx := api.Broker().GetPoolTransaction(hash)
	if tx == nil {
		logger.Debugf("tx %s is not in mempool", hash.String())
		tx, err = api.Broker().GetTransaction(hash)
		if err != nil {
			logger.Debugf("tx %s is not in ledger", hash.String())
			return nil, nil, fmt.Errorf("get tx from ledger: %w", err)
		}

		meta, err = api.Broker().GetTransactionMeta(hash)
		if err != nil {
			logger.Debugf("tx meta for %s is not found", hash.String())
			return nil, nil, fmt.Errorf("get tx meta from ledger: %w", err)
		}
	} else {
		logger.Debugf("tx %s is found in mempool", hash.String())
		err = retry.Retry(func(attempt uint) error {
			meta, err = api.Broker().GetTransactionMeta(hash)
			if err != nil {
				logger.Debugf("tx meta for %s is not found", hash.String())
				return err
			}
			return nil
		}, strategy.Limit(5), strategy.Backoff(backoff.Fibonacci(200*time.Millisecond)))
		if err != nil {
			meta = &types.TransactionMeta{}
			return nil, meta, err
		}
	}

	return tx, meta, nil
}
