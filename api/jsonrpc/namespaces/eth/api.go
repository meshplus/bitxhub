package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	rpctypes "github.com/meshplus/bitxhub/api/jsonrpc/types"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/repo"
	vm1 "github.com/meshplus/eth-kit/evm"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	types2 "github.com/meshplus/eth-kit/types"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PublicEthereumAPI is the eth_ prefixed set of APIs in the Web3 JSON-RPC spec.
type PublicEthereumAPI struct {
	ctx     context.Context
	cancel  context.CancelFunc
	config  *repo.Config
	chainID *big.Int
	logger  logrus.FieldLogger
	api     api.CoreAPI
}

// NewAPI creates an instance of the public ETH Web3 API.
func NewAPI(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) (*PublicEthereumAPI, error) {
	ctx, cancel := context.WithCancel(context.Background())

	return &PublicEthereumAPI{
		ctx:     ctx,
		cancel:  cancel,
		config:  config,
		chainID: big.NewInt(int64(config.Genesis.ChainID)),
		logger:  logger,
		api:     api,
	}, nil
}

// ProtocolVersion returns the supported Ethereum protocol version.
func (api *PublicEthereumAPI) ProtocolVersion() hexutil.Uint {
	api.logger.Debug("eth_protocolVersion")
	return hexutil.Uint(rpctypes.ProtocolVersion)
}

// ChainId returns the chain's identifier in hex format
func (api *PublicEthereumAPI) ChainId() (hexutil.Uint, error) { // nolint
	api.logger.Debug("eth_chainId")
	return hexutil.Uint(api.chainID.Uint64()), nil
}

// Syncing returns whether or not the current node is syncing with other peers. Returns false if not, or a struct
// outlining the state of the sync if it is.
func (api *PublicEthereumAPI) Syncing() (interface{}, error) {
	api.logger.Debug("eth_syncing")

	// TODO

	return nil, nil
}

// Mining returns whether or not this node is currently mining. Always false.
func (api *PublicEthereumAPI) Mining() bool {
	api.logger.Debug("eth_mining")
	return false
}

// Hashrate returns the current node's hashrate. Always 0.
func (api *PublicEthereumAPI) Hashrate() hexutil.Uint64 {
	api.logger.Debug("eth_hashrate")
	return 0
}

// GasPrice returns the current gas price based on Ethermint's gas price oracle.
func (api *PublicEthereumAPI) GasPrice() *hexutil.Big {
	api.logger.Debug("eth_gasPrice")
	out := big.NewInt(int64(api.config.Genesis.BvmGasPrice))
	return (*hexutil.Big)(out)
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic transactions.
func (api *PublicEthereumAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	api.logger.Debug("eth_maxPriorityFeePerGas")
	return (*hexutil.Big)(new(big.Int)), nil
}

type feeHistoryResult struct {
	OldestBlock  rpc.BlockNumber  `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

func (api *PublicEthereumAPI) FeeHistory(ctx context.Context, blockCount rpctypes.DecimalOrHex, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*feeHistoryResult, error) {
	api.logger.Debug("eth_feeHistory")
	return &feeHistoryResult{}, nil
}

// BlockNumber returns the current block number.
func (api *PublicEthereumAPI) BlockNumber() (hexutil.Uint64, error) {
	api.logger.Debug("eth_blockNumber")
	meta, err := api.api.Chain().Meta()
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(meta.Height), nil
}

// GetBalance returns the provided account's balance, blockNum is ignored.
func (api *PublicEthereumAPI) GetBalance(address common.Address, blockNum rpctypes.BlockNumber) (*hexutil.Big, error) {
	api.logger.Debugf("eth_getBalance, address: %s, block number: %d", address.String(), blockNum)

	stateLedger, err := api.getStateLedgerAt(blockNum)
	if err != nil {
		return nil, err
	}

	balance := stateLedger.GetBalance(types.NewAddress(address.Bytes()))
	api.logger.Debugf("balance: %d", balance)

	return (*hexutil.Big)(balance), nil
}

// GetStorageAt returns the contract storage at the given address and key, blockNum is ignored.
func (api *PublicEthereumAPI) GetStorageAt(address common.Address, key string, blockNum rpctypes.BlockNumber) (hexutil.Bytes, error) {
	api.logger.Debugf("eth_getStorageAt, address: %s, key: %s, block number: %d", address, key, blockNum)

	stateLedger, err := api.getStateLedgerAt(blockNum)
	if err != nil {
		return nil, err
	}

	ok, val := stateLedger.GetState(types.NewAddress(address.Bytes()), []byte(key))
	if !ok {
		return nil, nil
	}

	return val, nil
}

// GetTransactionCount returns the number of transactions at the given address, blockNum is ignored.
func (api *PublicEthereumAPI) GetTransactionCount(address common.Address, blockNum rpctypes.BlockNumber) (*hexutil.Uint64, error) {
	api.logger.Debugf("eth_getTransactionCount, address: %s, block number: %d", address, blockNum)

	stateLedger, err := api.getStateLedgerAt(blockNum)
	if err != nil {
		return nil, err
	}

	nonce := stateLedger.GetNonce(types.NewAddress(address.Bytes()))

	return (*hexutil.Uint64)(&nonce), nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block identified by hash.
func (api *PublicEthereumAPI) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	api.logger.Debugf("eth_getBlockTransactionCountByHash, hash: %s", hash.String())

	block, err := api.api.Broker().GetBlock("HASH", hash.String())
	if err != nil {
		return nil
	}

	count := uint(len(block.Transactions.Transactions))

	return (*hexutil.Uint)(&count)
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block identified by its height.
func (api *PublicEthereumAPI) GetBlockTransactionCountByNumber(blockNum uint64) *hexutil.Uint {
	api.logger.Debugf("eth_getBlockTransactionCountByNumber, block number: %d", blockNum)

	block, err := api.api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", blockNum))
	if err != nil {
		return nil
	}

	count := uint(len(block.Transactions.Transactions))

	return (*hexutil.Uint)(&count)
}

// GetUncleCountByBlockHash returns the number of uncles in the block identified by hash. Always zero.
func (api *PublicEthereumAPI) GetUncleCountByBlockHash(_ common.Hash) hexutil.Uint {
	return 0
}

func (api *PublicEthereumAPI) GetUncleCountByBlockNumber(_ uint64) hexutil.Uint {
	return 0
}

// GetCode returns the contract code at the given address, blockNum is ignored.
func (api *PublicEthereumAPI) GetCode(address common.Address, blockNum rpctypes.BlockNumber) (hexutil.Bytes, error) {
	api.logger.Debugf("eth_getCode, address: %s, block number: %d", address.String(), blockNum)

	stateLedger, err := api.getStateLedgerAt(blockNum)
	if err != nil {
		return nil, err
	}

	code := stateLedger.GetCode(types.NewAddress(address.Bytes()))

	return code, nil
}

// GetTransactionLogs returns the logs given a transaction hash.
func (api *PublicEthereumAPI) GetTransactionLogs(txHash common.Hash) ([]*pb.EvmLog, error) {
	api.logger.Debugf("eth_getTransactionLogs, hash: %s", txHash.String())

	receipt, err := api.api.Broker().GetReceipt(types.NewHash(txHash.Bytes()))
	if err != nil {
		return nil, err
	}

	return receipt.EvmLogs, nil
}

// SendRawTransaction send a raw Ethereum transaction.
func (api *PublicEthereumAPI) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	api.logger.Debugf("eth_sendRawTransaction, data: %s", data.String())

	tx := &types2.EthTransaction{}
	if err := tx.Unmarshal(data); err != nil {
		return [32]byte{}, err
	}
	api.logger.Debugf("get new eth tx: %s", tx.GetHash().String())

	if tx.GetFrom() == nil {
		return [32]byte{}, fmt.Errorf("verify signature failed")
	}

	err := api.api.Broker().OrderReady()
	if err != nil {
		return [32]byte{}, status.Newf(codes.Internal, "the system is temporarily unavailable %s", err.Error()).Err()
	}

	if err := api.checkTransaction(tx); err != nil {
		return [32]byte{}, status.Newf(codes.InvalidArgument, "check transaction fail for %s", err.Error()).Err()
	}

	return api.sendTransaction(tx)
}

func (api *PublicEthereumAPI) getStateLedgerAt(blockNum rpctypes.BlockNumber) (ledger2.StateLedger, error) {
	if api.config.Ledger.Type == "simple" {
		return api.api.Broker().GetStateLedger(), nil
	}

	if blockNum == rpctypes.PendingBlockNumber || blockNum == rpctypes.LatestBlockNumber {
		meta, err := api.api.Chain().Meta()
		if err != nil {
			return nil, err
		}

		blockNum = rpctypes.BlockNumber(meta.Height)
	}

	block, err := api.api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", blockNum))
	if err != nil {
		return nil, err
	}

	return api.api.Broker().GetStateLedger().(*ledger2.ComplexStateLedger).StateAt(block.BlockHeader.StateRoot)
}

func (api *PublicEthereumAPI) checkTransaction(tx *types2.EthTransaction) error {
	if tx.GetFrom() == nil {
		return fmt.Errorf("tx from address is nil")
	}
	api.logger.Debugf("from address: %s, nonce: %d", tx.GetFrom().String(), tx.GetNonce())

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

	if tx.GetTimeStamp() < time.Now().UnixNano()-10*time.Minute.Nanoseconds() ||
		tx.GetTimeStamp() > time.Now().UnixNano()+10*time.Minute.Nanoseconds() {
		return fmt.Errorf("timestamp is illegal")
	}

	if len(tx.GetSignature()) == 0 {
		return fmt.Errorf("signature can't be empty")
	}

	if tx.GetGasPrice().Cmp((*big.Int)(api.GasPrice())) < 0 {
		return fmt.Errorf("gas price is too low, at least %s is required", api.GasPrice().String())
	}

	return nil
}

func (api *PublicEthereumAPI) sendTransaction(tx *types2.EthTransaction) (common.Hash, error) {
	if err := tx.VerifySignature(); err != nil {
		return [32]byte{}, err
	}
	err := api.api.Broker().HandleTransaction(tx)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.GetHash().RawHash, nil
}

func newRevertError(data []byte) *revertError {
	reason, errUnpack := abi.UnpackRevert(data)
	err := errors.New("execution reverted")
	if errUnpack == nil {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return &revertError{
		error:  err,
		reason: hexutil.Encode(data),
	}
}

// revertError is an API error that encompassas an EVM revertal with JSON error
// code and a binary data blob.
type revertError struct {
	error
	reason string // revert reason hex encoded
}

// ErrorCode returns the JSON error code for a revertal.
// See: https://github.com/ethereum/wiki/wiki/JSON-RPC-Error-Codes-Improvement-Proposal
func (e *revertError) ErrorCode() int {
	return 3
}

// ErrorData returns the hex encoded revert reason.
func (e *revertError) ErrorData() interface{} {
	return e.reason
}

// Call performs a raw contract call.
func (api *PublicEthereumAPI) Call(args types2.CallArgs, blockNr rpc.BlockNumber, _ *map[common.Address]rpctypes.Account) (hexutil.Bytes, error) {
	api.logger.Debugf("eth_call, args: %v, block number: %d", args, blockNr.Int64())

	// Determine the highest gas limit can be used during call.
	if args.Gas == nil || uint64(*args.Gas) < params.TxGas {
		// Retrieve the block to act as the gas ceiling
		args.Gas = (*hexutil.Uint64)(&api.config.GasLimit)
	}

	tx := &types2.EthTransaction{}
	tx.FromCallArgs(args)

	receipt, err := api.api.Broker().HandleView(tx)
	if err != nil {
		return nil, err
	}

	api.logger.Debugf("receipt: %v", receipt)

	if receipt.Status == pb.Receipt_FAILED {
		errMsg := string(receipt.Ret)
		if strings.HasPrefix(errMsg, vm1.ErrExecutionReverted.Error()) {
			return nil, newRevertError(receipt.Ret[len(vm1.ErrExecutionReverted.Error()):])
		}
		return nil, errors.New(errMsg)
	}

	return receipt.Ret, nil
}

// EstimateGas returns an estimate of gas usage for the given smart contract call.
// It adds 2,000 gas to the returned value instead of using the gas adjustment
// param from the SDK.
func (api *PublicEthereumAPI) EstimateGas(args types2.CallArgs) (hexutil.Uint64, error) {
	api.logger.Debugf("eth_estimateGas, args: %s", args)

	// Determine the highest gas limit can be used during the estimation.
	if args.Gas == nil || uint64(*args.Gas) < params.TxGas {
		// Retrieve the block to act as the gas ceiling
		args.Gas = (*hexutil.Uint64)(&api.config.GasLimit)
	}
	// Determine the lowest and highest possible gas limits to binary search in between
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	if uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		hi = api.config.GasLimit
	}
	cap = hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (bool, []byte) {
		tx := &types2.EthTransaction{}
		args.Gas = (*hexutil.Uint64)(&gas)
		tx.FromCallArgs(args)

		result, err := api.api.Broker().HandleView(tx)
		if err != nil || !result.IsSuccess() {
			return false, result.Ret
		}
		return true, nil
	}

	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		if ok, _ := executable(mid); !ok {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		if ok, ret := executable(hi); !ok {
			if ret != nil {
				return 0, errors.New(string(ret))
			}
			return 0, errors.New("gas required exceeds allowance or always failing transaction")
		}
	}
	return hexutil.Uint64(hi), nil
}

// GetBlockByHash returns the block identified by hash.
func (api *PublicEthereumAPI) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	api.logger.Debugf("eth_getBlockByHash, hash: %s, full: %v", hash.String(), fullTx)

	block, err := api.api.Broker().GetBlock("HASH", hash.String())
	if err != nil {
		return nil, err
	}
	return api.formatBlock(block, fullTx)
}

// GetBlockByNumber returns the block identified by number.
func (api *PublicEthereumAPI) GetBlockByNumber(blockNum rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	api.logger.Debugf("eth_getBlockByNumber, number: %d, full: %v", blockNum, fullTx)

	if blockNum == rpc.PendingBlockNumber || blockNum == rpc.LatestBlockNumber {
		meta, err := api.api.Chain().Meta()
		if err != nil {
			return nil, err
		}
		blockNum = rpc.BlockNumber(meta.Height)
	}

	block, err := api.api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", blockNum))
	if err != nil {
		return nil, err
	}

	return api.formatBlock(block, fullTx)
}

// GetTransactionByHash returns the transaction identified by hash.
func (api *PublicEthereumAPI) GetTransactionByHash(hash common.Hash) (*rpctypes.RPCTransaction, error) {
	api.logger.Debugf("eth_getTransactionByHash, hash: %s", hash.String())

	ethTx, meta, err := api.GetEthTransactionByHash(types.NewHash(hash.Bytes()))
	if err != nil {
		return nil, err
	}

	return newRPCTransaction(ethTx, common.BytesToHash(meta.BlockHash), meta.BlockHeight, meta.Index), nil
}

func (api *PublicEthereumAPI) GetEthTransactionByHash(hash *types.Hash) (*types2.EthTransaction, *pb.TransactionMeta, error) {
	var err error
	meta := &pb.TransactionMeta{}

	tx := api.api.Broker().GetPoolTransaction(hash)
	if tx == nil {
		api.logger.Debugf("tx %s is not in mempool", hash.String())
		tx, err = api.api.Broker().GetTransaction(hash)
		if err != nil {
			api.logger.Debugf("tx %s is not in ledger", hash.String())
			return nil, nil, fmt.Errorf("get tx from ledger: %w", err)
		}

		meta, err = api.api.Broker().GetTransactionMeta(hash)
		if err != nil {
			api.logger.Debugf("tx meta for %s is not found", hash.String())
			return nil, nil, fmt.Errorf("get tx meta from ledger: %w", err)
		}
	} else {
		api.logger.Debugf("tx %s is found in mempool", hash.String())
		if strings.Contains(strings.ToLower(api.config.Order.Type), "rbft") {
			meta, err = api.api.Broker().GetTransactionMeta(hash)
			if err != nil {
				api.logger.Debugf("tx meta for %s is not found", hash.String())
				meta = &pb.TransactionMeta{}
			}
		}
	}

	ethTx, ok := tx.(*types2.EthTransaction)
	if !ok {
		return nil, nil, fmt.Errorf("tx is not in eth format")
	}

	return ethTx, meta, nil
}

// GetTransactionByBlockHashAndIndex returns the transaction identified by hash and index.
func (api *PublicEthereumAPI) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	api.logger.Debugf("eth_getTransactionByHashAndIndex, hash: %s, index: %d", hash.String(), idx)

	return api.getTxByBlockInfoAndIndex("HASH", hash.String(), idx)
}

// GetTransactionByBlockNumberAndIndex returns the transaction identified by number and index.
func (api *PublicEthereumAPI) GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	api.logger.Debugf("eth_getTransactionByBlockNumberAndIndex, number: %d, index: %d", blockNum, idx)

	height := uint64(0)

	switch blockNum {
	case rpctypes.PendingBlockNumber:
		// get all the EVM pending txs
		// FIXME: get pending txs with correct block number
		txs := api.api.Broker().GetPendingTransactions(200)
		if int(idx) >= len(txs) {
			return nil, fmt.Errorf("index beyond block transactions' size")
		}

		ethTx, ok := txs[idx].(*types2.EthTransaction)
		if !ok {
			return nil, fmt.Errorf("tx is not in eth format")
		}

		return newRPCTransaction(ethTx, common.Hash{}, 0, 0), nil

	case rpctypes.LatestBlockNumber:
		meta, err := api.api.Chain().Meta()
		if err != nil {
			return nil, err
		}
		height = meta.Height
	default:
		height = uint64(blockNum.Int64())
	}

	return api.getTxByBlockInfoAndIndex("HEIGHT", fmt.Sprintf("%d", height), idx)
}

// GetTransactionReceipt returns the transaction receipt identified by hash.
func (api *PublicEthereumAPI) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	api.logger.Debugf("eth_getTransactionReceipt, hash: %s", hash.String())

	txHash := types.NewHash(hash.Bytes())
	tx, meta, err := api.GetEthTransactionByHash(txHash)
	if err != nil {
		api.logger.Debugf("no tx found for hash %s", txHash.String())
		return nil, nil
	}

	receipt, err := api.api.Broker().GetReceipt(txHash)
	if err != nil {
		api.logger.Debugf("no receipt found for tx %s", txHash.String())
		return nil, nil
	}

	block, err := api.api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", meta.BlockHeight))
	if err != nil {
		api.logger.Debugf("no block found for height %d", meta.BlockHeight)
		return nil, err
	}

	cumulativeGasUsed, err := api.getBlockCumulativeGas(block, meta.Index)
	if err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"type":              hexutil.Uint(tx.GetType()),
		"cumulativeGasUsed": hexutil.Uint64(cumulativeGasUsed),
		"logs":              receipt.EvmLogs,
		"transactionHash":   hash,
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"blockHash":         common.BytesToHash(meta.BlockHash),
		"blockNumber":       hexutil.Uint64(meta.BlockHeight),
		"transactionIndex":  hexutil.Uint64(meta.Index),
		"from":              common.BytesToAddress(tx.GetFrom().Bytes()),
	}
	if receipt.Bloom == nil {
		fields["logsBloom"] = types.Bloom{}
	} else {
		fields["logsBloom"] = *receipt.Bloom
	}

	if len(receipt.EvmLogs) == 0 {
		fields["logs"] = make([]*pb.EvmLog, 0)
	}

	if receipt.Status == pb.Receipt_SUCCESS {
		fields["status"] = hexutil.Uint(1)
	} else {
		fields["status"] = hexutil.Uint(0)
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

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (api *PublicEthereumAPI) PendingTransactions() ([]*rpctypes.RPCTransaction, error) {
	api.logger.Debug("eth_pendingTransactions")

	txs := api.api.Broker().GetPendingTransactions(1000)

	rpcTxs := make([]*rpctypes.RPCTransaction, len(txs))

	for _, tx := range txs {
		ethTx, ok := tx.(*types2.EthTransaction)
		if !ok {
			continue
		}
		rpcTxs = append(rpcTxs, newRPCTransaction(ethTx, common.Hash{}, 0, 0))
	}

	return rpcTxs, nil
}

// GetUncleByBlockHashAndIndex returns the uncle identified by hash and index. Always returns nil.
func (api *PublicEthereumAPI) GetUncleByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) map[string]interface{} {
	return nil
}

// GetUncleByBlockNumberAndIndex returns the uncle identified by number and index. Always returns nil.
func (api *PublicEthereumAPI) GetUncleByBlockNumberAndIndex(number hexutil.Uint, idx hexutil.Uint) map[string]interface{} {
	return nil
}

func (api *PublicEthereumAPI) getTxByBlockInfoAndIndex(mode string, key string, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	block, err := api.api.Broker().GetBlock(mode, key)
	if err != nil {
		return nil, err
	}

	if int(idx) >= len(block.Transactions.Transactions) {
		return nil, fmt.Errorf("index beyond block transactions' size")
	}

	ethTx, ok := block.Transactions.Transactions[idx].(*types2.EthTransaction)
	if !ok {
		return nil, fmt.Errorf("tx is not in eth format")
	}

	meta, err := api.api.Broker().GetTransactionMeta(ethTx.GetHash())
	if err != nil {
		return nil, err
	}

	return newRPCTransaction(ethTx, common.BytesToHash(meta.BlockHash), meta.BlockHeight, meta.Index), nil
}

// FormatBlock creates an ethereum block from a tendermint header and ethereum-formatted
// transactions.
func (api *PublicEthereumAPI) formatBlock(block *pb.Block, fullTx bool) (map[string]interface{}, error) {
	cumulativeGas, err := api.getBlockCumulativeGas(block, uint64(len(block.Transactions.Transactions)-1))
	if err != nil {
		return nil, err
	}

	formatTx := func(tx pb.Transaction, index uint64) (interface{}, error) {
		return tx.GetHash(), nil
	}
	if fullTx {
		formatTx = func(tx pb.Transaction, index uint64) (interface{}, error) {
			return newRPCTransaction(tx, common.BytesToHash(block.BlockHash.Bytes()), block.Height(), index), nil
		}
	}
	txs := block.Transactions.Transactions
	transactions := make([]interface{}, len(txs))
	for i, tx := range txs {
		if transactions[i], err = formatTx(tx, uint64(i)); err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"number":           (*hexutil.Big)(big.NewInt(int64(block.Height()))),
		"hash":             block.BlockHash,
		"parentHash":       block.BlockHeader.ParentHash,
		"nonce":            ethtypes.BlockNonce{}, // PoW specific
		"sha3Uncles":       common.Hash{},         // No uncles in raft/rbft
		"logsBloom":        block.BlockHeader.Bloom,
		"transactionsRoot": block.BlockHeader.TxRoot,
		"stateRoot":        block.BlockHeader.StateRoot,
		"miner":            common.Address{},
		"mixHash":          common.Hash{},
		"difficulty":       (*hexutil.Big)(big.NewInt(0)),
		"totalDifficulty":  (*hexutil.Big)(big.NewInt(0)),
		"extraData":        hexutil.Uint64(0),
		"size":             hexutil.Uint64(block.Size()),
		"gasLimit":         hexutil.Uint64(api.config.GasLimit), // Static gas limit
		"gasUsed":          hexutil.Uint64(cumulativeGas),
		"timestamp":        hexutil.Uint64(block.BlockHeader.Timestamp),
		"transactions":     transactions,
		"uncles":           []string{},
		"receiptsRoot":     block.BlockHeader.ReceiptRoot,
	}, nil
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation
func newRPCTransaction(tx pb.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *rpctypes.RPCTransaction {
	from := common.BytesToAddress(tx.GetFrom().Bytes())
	var to *common.Address
	if tx.GetTo() != nil {
		toAddr := common.BytesToAddress(tx.GetTo().Bytes())
		to = &toAddr
	}
	v, r, s := tx.GetRawSignature()
	result := &rpctypes.RPCTransaction{
		Type:     hexutil.Uint64(tx.GetType()),
		From:     from,
		Gas:      hexutil.Uint64(tx.GetGas()),
		GasPrice: (*hexutil.Big)(tx.GetGasPrice()),
		Hash:     tx.GetHash().RawHash,
		Input:    hexutil.Bytes(tx.GetPayload()),
		Nonce:    hexutil.Uint64(tx.GetNonce()),
		To:       to,
		Value:    (*hexutil.Big)(tx.GetValue()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	switch tx.GetType() {
	case ethtypes.AccessListTxType:
		al := tx.(*types2.EthTransaction).GetInner().GetAccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.GetChainID())
	case ethtypes.DynamicFeeTxType:
		al := tx.(*types2.EthTransaction).GetInner().GetAccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.GetChainID())
		result.GasFeeCap = (*hexutil.Big)(tx.(*types2.EthTransaction).GetInner().GetGasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.(*types2.EthTransaction).GetInner().GetGasTipCap())
		result.GasPrice = result.GasFeeCap
	}

	return result
}

// GetBlockCumulativeGas returns the cumulative gas used on a block up to a given transaction index (inclusive)
func (api *PublicEthereumAPI) getBlockCumulativeGas(block *pb.Block, idx uint64) (uint64, error) {
	var gasUsed uint64
	txs := block.Transactions.Transactions

	for i := 0; i <= int(idx) && i < len(txs); i++ {
		receipt, err := api.api.Broker().GetReceipt(txs[i].GetHash())
		if err != nil {
			return 0, err
		}

		gasUsed += receipt.GetGasUsed()
	}

	return gasUsed, nil
}
