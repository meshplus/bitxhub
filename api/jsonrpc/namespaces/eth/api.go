package eth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/backoff"
	"github.com/Rican7/retry/strategy"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	types3 "github.com/ethereum/go-ethereum/core/types"
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

var (
	NotSupportApiError = fmt.Errorf("unsupported interface")
)

// BlockChain API provides an API for accessing blockchain data
type BlockChainAPI struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *repo.Config
	api    api.CoreAPI
	logger logrus.FieldLogger
}

func NewBlockChainAPI(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) *BlockChainAPI {
	ctx, cancel := context.WithCancel(context.Background())
	return &BlockChainAPI{ctx: ctx, cancel: cancel, config: config, api: api, logger: logger}
}

// ChainId returns the chain's identifier in hex format
func (api *BlockChainAPI) ChainId() (hexutil.Uint, error) { // nolint
	api.logger.Debug("eth_chainId")
	return hexutil.Uint(api.config.Genesis.ChainID), nil
}

// BlockNumber returns the current block number.
func (api *BlockChainAPI) BlockNumber() (hexutil.Uint64, error) {
	api.logger.Debug("eth_blockNumber")
	meta, err := api.api.Chain().Meta()
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(meta.Height), nil
}

// GetBalance returns the provided account's balance, blockNum is ignored.
func (api *BlockChainAPI) GetBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error) {
	api.logger.Debugf("eth_getBalance, address: %s, block number : %d", address.String())

	stateLedger, err := getStateLedgerAt(api.config, api.api, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	balance := stateLedger.GetBalance(types.NewAddress(address.Bytes()))
	api.logger.Debugf("balance: %d", balance)

	return (*hexutil.Big)(balance), nil
}

type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// todo
// GetProof returns the Merkle-proof for a given account and optionally some storage keys.
func (api *BlockChainAPI) GetProof(address common.Address, storageKeys []string, blockNrOrHash rpctypes.BlockNumberOrHash) (*AccountResult, error) {
	return nil, NotSupportApiError
}

// GetBlockByNumber returns the block identified by number.
func (api *BlockChainAPI) GetBlockByNumber(blockNum rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
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

	return formatBlock(api.api, api.config, block, fullTx)
}

// GetBlockByHash returns the block identified by hash.
func (api *BlockChainAPI) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	api.logger.Debugf("eth_getBlockByHash, hash: %s, full: %v", hash.String(), fullTx)

	block, err := api.api.Broker().GetBlock("HASH", hash.String())
	if err != nil {
		return nil, err
	}
	return formatBlock(api.api, api.config, block, fullTx)
}

// GetCode returns the contract code at the given address, blockNum is ignored.
func (api *BlockChainAPI) GetCode(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error) {
	api.logger.Debugf("eth_getCode, address: %s", address.String())

	stateLedger, err := getStateLedgerAt(api.config, api.api, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	code := stateLedger.GetCode(types.NewAddress(address.Bytes()))

	return code, nil
}

// GetStorageAt returns the contract storage at the given address and key, blockNum is ignored.
func (api *BlockChainAPI) GetStorageAt(address common.Address, key string, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error) {
	api.logger.Debugf("eth_getStorageAt, address: %s, key: %s", address, key)

	stateLedger, err := getStateLedgerAt(api.config, api.api, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	ok, val := stateLedger.GetState(types.NewAddress(address.Bytes()), []byte(key))
	if !ok {
		return nil, nil
	}

	return val, nil
}

// Call performs a raw contract call.
func (api *BlockChainAPI) Call(args types2.CallArgs, blockNrOrHash *rpctypes.BlockNumberOrHash, _ *map[common.Address]rpctypes.Account) (hexutil.Bytes, error) {
	api.logger.Debugf("eth_call, args: %v", args)

	// Determine the highest gas limit can be used during call.
	// if args.Gas == nil || uint64(*args.Gas) < params.TxGas {
	// 	// Retrieve the block to act as the gas ceiling
	// 	args.Gas = (*hexutil.Uint64)(&api.config.GasLimit)
	// }

	// tx := &types2.EthTransaction{}
	// tx.FromCallArgs(args)

	// receipt, err := api.api.Broker().HandleView(tx)
	// if err != nil {
	// 	return nil, err
	// }

	receipt, err := DoCall(api.ctx, api.api, args, api.config.RPCEVMTimeout, api.config.RPCGasCap, api.logger)
	if err != nil {
		return nil, err
	}
	api.logger.Debugf("receipt: %v", receipt)
	if len(receipt.Revert()) > 0 {
		return nil, newRevertError(receipt.Revert())
	}

	return receipt.Return(), nil

	// if receipt.Status == pb.Receipt_FAILED {
	// 	errMsg := string(receipt.Ret)
	// 	if strings.HasPrefix(errMsg, vm1.ErrExecutionReverted.Error()) {
	// 		return nil, newRevertError(receipt.Ret[len(vm1.ErrExecutionReverted.Error()):])
	// 	}
	// 	return nil, errors.New(errMsg)
	// }
}

func DoCall(ctx context.Context, api api.CoreAPI, args types2.CallArgs, timeout time.Duration, globalGasCap uint64, logger logrus.FieldLogger) (*vm1.ExecutionResult, error) {
	defer func(start time.Time) { logger.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	//GET EVM Instance
	msg, err := args.ToMessage(globalGasCap, big.NewInt(0))
	if err != nil {
		return nil, err
	}

	leger := api.Broker().GetStateLedger()
	meta, err := api.Chain().Meta()
	if err != nil {
		return nil, err
	}
	leger.PrepareBlock(meta.BlockHash, meta.Height)
	evm := api.Broker().GetEvm(msg, &vm1.Config{NoBaseFee: true})
	if err != nil {
		return nil, fmt.Errorf("error get evm")
	}

	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	gp := new(vm1.GasPool).AddGas(math.MaxUint64)
	result, err := vm1.ApplyMessage(evm, msg, gp)

	leger.Clear()

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		logger.Errorf("apply msg failed: %s", err.Error())
		return result, err
	}

	return result, nil
}

// EstimateGas returns an estimate of gas usage for the given smart contract call.
// It adds 2,000 gas to the returned value instead of using the gas adjustment
// param from the SDK.
func (api *BlockChainAPI) EstimateGas(args types2.CallArgs, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Uint64, error) {
	api.logger.Debugf("eth_estimateGas, args: %s", args)

	// Determine the highest gas limit can be used during the estimation.
	// if args.Gas == nil || uint64(*args.Gas) < params.TxGas {
	// 	// Retrieve the block to act as the gas ceiling
	// 	args.Gas = (*hexutil.Uint64)(&api.config.GasLimit)
	// }
	// Determine the lowest and highest possible gas limits to binary search in between
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		//todo use block gasLimit instead of config gasLimit
		hi = api.config.GasLimit
	}

	var feeCap *big.Int
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return 0, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	} else if args.GasPrice != nil {
		feeCap = args.GasPrice.ToInt()
	} else if args.MaxFeePerGas != nil {
		feeCap = args.MaxFeePerGas.ToInt()
	} else {
		feeCap = common.Big0
	}
	if feeCap.BitLen() != 0 {
		stateLedger, err := getStateLedgerAt(api.config, api.api, blockNrOrHash)
		if err != nil {
			return 0, err
		}
		balance := stateLedger.GetBalance(types.NewAddress(args.From.Bytes()))
		api.logger.Debugf("balance: %d", balance)
		available := new(big.Int).Set(balance)
		if args.Value != nil {
			if args.Value.ToInt().Cmp(available) >= 0 {
				return 0, errors.New("insufficient funds for transfer")
			}
			available.Sub(available, args.Value.ToInt())
		}
		allowance := new(big.Int).Div(available, feeCap)
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := args.Value
			if transfer == nil {
				transfer = new(hexutil.Big)
			}
			hi = allowance.Uint64()
		}
	}

	gasCap := api.config.RPCGasCap
	if gasCap != 0 && hi > gasCap {
		api.logger.Warn("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
		hi = gasCap
	}

	cap = hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (bool, *vm1.ExecutionResult, error) {
		args.Gas = (*hexutil.Uint64)(&gas)

		result, err := DoCall(api.ctx, api.api, args, api.config.RPCEVMTimeout, api.config.RPCGasCap, api.logger)
		if err != nil {
			if errors.Is(err, errors.New("intrinsic gas too low")) {
				return true, nil, nil // Special case, raise gas limit
			}
			return false, nil, err
		}
		return result.Failed(), result, nil

		// tx := &types2.EthTransaction{}
		// args.Gas = (*hexutil.Uint64)(&gas)
		// tx.FromCallArgs(args)

		// result, err := api.api.Broker().HandleView(tx)
		// if err != nil || !result.IsSuccess() {
		// 	return false, result.Ret
		// }
	}

	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		failed, ret, err := executable(hi)
		if err != nil {
			return 0, err
		}
		if failed {
			if ret != nil && ret.Err != vm1.ErrOutOfGas {
				if len(ret.Revert()) > 0 {
					return 0, newRevertError(ret.Revert())
				}
				return 0, ret.Err
			}
			return 0, errors.New("gas required exceeds allowance or always failing transaction")
		}
	}
	return hexutil.Uint64(hi), nil
}

// accessListResult returns an optional accesslist
// It's the result of the `debug_createAccessList` RPC call.
// It contains an error if the transaction itself failed.
type accessListResult struct {
	Accesslist *ethtypes.AccessList `json:"accessList"`
	Error      string               `json:"error,omitempty"`
	GasUsed    hexutil.Uint64       `json:"gasUsed"`
}

func (s *BlockChainAPI) CreateAccessList(args types2.CallArgs, blockNrOrHash *rpctypes.BlockNumberOrHash) (*accessListResult, error) {
	return nil, NotSupportApiError
}

// BitxhubAPI provides an API to get related info
type BitxhubAPI struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *repo.Config
	api    api.CoreAPI
	logger logrus.FieldLogger
}

func NewBitxhubAPI(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) *BitxhubAPI {
	ctx, cancel := context.WithCancel(context.Background())
	return &BitxhubAPI{ctx: ctx, cancel: cancel, config: config, api: api, logger: logger}
}

// GasPrice returns the current gas price based on Ethermint's gas price oracle.
// todo Supplementary gas price
func (api *BitxhubAPI) GasPrice() *hexutil.Big {
	api.logger.Debug("eth_gasPrice")
	out := big.NewInt(int64(api.config.Genesis.BvmGasPrice))
	return (*hexutil.Big)(out)
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic transactions.
// todo Supplementary gas fee
func (api *BitxhubAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	api.logger.Debug("eth_maxPriorityFeePerGas")
	return (*hexutil.Big)(new(big.Int)), nil
}

type feeHistoryResult struct {
	OldestBlock  rpc.BlockNumber  `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

// FeeHistory return feeHistory
// todo Supplementary feeHsitory
func (api *BitxhubAPI) FeeHistory(blockCount rpctypes.DecimalOrHex, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*feeHistoryResult, error) {
	api.logger.Debug("eth_feeHistory")
	return nil, NotSupportApiError
}

// Syncing returns whether or not the current node is syncing with other peers. Returns false if not, or a struct
// outlining the state of the sync if it is.
func (api *BitxhubAPI) Syncing() (interface{}, error) {
	api.logger.Debug("eth_syncing")

	// TODO
	// Supplementary data
	syncBlock := make(map[string]string)
	meta, err := api.api.Chain().Meta()

	if err != nil {
		syncBlock["result"] = "false"
		return syncBlock, err
	}

	syncBlock["startingBlock"] = fmt.Sprintf("%d", hexutil.Uint64(1))
	syncBlock["highestBlock"] = fmt.Sprintf("%d", hexutil.Uint64(meta.Height))
	syncBlock["currentBlock"] = syncBlock["highestBlock"]
	return syncBlock, nil
}

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

	count := uint(len(block.Transactions.Transactions))

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

	count := uint(len(block.Transactions.Transactions))

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

	stateLedger, err := getStateLedgerAt(api.config, api.api, blockNrOrHash)
	if err != nil {
		return nil, err
	}

	nonce := stateLedger.GetNonce(types.NewAddress(address.Bytes()))

	return (*hexutil.Uint64)(&nonce), nil
}

// GetTransactionByHash returns the transaction identified by hash.
func (api *TransactionAPI) GetTransactionByHash(hash common.Hash) (*rpctypes.RPCTransaction, error) {
	api.logger.Debugf("eth_getTransactionByHash, hash: %s", hash.String())

	ethTx, meta, err := getEthTransactionByHash(api.config, api.api, api.logger, types.NewHash(hash.Bytes()))
	if err != nil {
		return nil, err
	}

	return newRPCTransaction(ethTx, common.BytesToHash(meta.BlockHash), meta.BlockHeight, meta.Index), nil
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
	ethLogs := make([]*types3.Log, 0)
	for _, log := range receipt.EvmLogs {
		ethLog := &types3.Log{}
		data, err := json.Marshal(log)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, ethLog)
		if err != nil {
			api.logger.Errorf("unmarshal ethLog err:%s", err)
			return nil, err
		}
		api.logger.WithFields(logrus.Fields{"ethLog": ethLog}).Debug("get eth log")
		ethLogs = append(ethLogs, ethLog)
	}
	fields["logs"] = ethLogs

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

// SendRawTransaction send a raw Ethereum transaction.
func (api *TransactionAPI) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
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

	if err := checkTransaction(api.logger, tx); err != nil {
		return [32]byte{}, status.Newf(codes.InvalidArgument, "check transaction fail for %s", err.Error()).Err()
	}

	return sendTransaction(api.api, tx)
}

// ProtocolVersion returns the supported Ethereum protocol version.
// func (api *PublicEthereumAPI) ProtocolVersion() hexutil.Uint {
// 	api.logger.Debug("eth_protocolVersion")
// 	return hexutil.Uint(rpctypes.ProtocolVersion)
// }

// GetTransactionLogs returns the logs given a transaction hash.
// func (api *PublicEthereumAPI) GetTransactionLogs(txHash common.Hash) ([]*pb.EvmLog, error) {
// 	api.logger.Debugf("eth_getTransactionLogs, hash: %s", txHash.String())

// 	receipt, err := api.api.Broker().GetReceipt(types.NewHash(txHash.Bytes()))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return receipt.EvmLogs, nil
// }

func getStateLedgerAt(config *repo.Config, api api.CoreAPI, blockNrOrHash rpctypes.BlockNumberOrHash) (ledger2.StateLedger, error) {
	return api.Broker().GetStateLedger(), nil
	// todo
	// supplementary block height and block hash processing

	// if blockNr, ok := blockNrOrHash.Number(); ok {
	// 	if blockNr == rpctypes.PendingBlockNumber || blockNr == rpctypes.LatestBlockNumber {
	// 		meta, err := api.Chain().Meta()
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		blockNr = rpctypes.BlockNumber(meta.Height)
	// 	}
	// 	block, err := api.Broker().GetBlock("HEIGHT", fmt.Sprintf("%d", blockNr))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return api.Broker().GetStateLedger().(*ledger2.ComplexStateLedger).StateAt(block.BlockHeader.StateRoot)
	// }

	// if hash, ok := blockNrOrHash.Hash(); ok {
	// 	block, err := api.Broker().GetBlock("Hash", fmt.Sprintf("%d", hash))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return api.Broker().GetStateLedger().(*ledger2.ComplexStateLedger).StateAt(block.BlockHeader.StateRoot)
	// }
	//return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func checkTransaction(logger logrus.FieldLogger, tx *types2.EthTransaction) error {
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

func sendTransaction(api api.CoreAPI, tx *types2.EthTransaction) (common.Hash, error) {
	if err := tx.VerifySignature(); err != nil {
		return [32]byte{}, err
	}
	err := api.Broker().HandleTransaction(tx)
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

func getEthTransactionByHash(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger, hash *types.Hash) (*types2.EthTransaction, *pb.TransactionMeta, error) {
	var err error
	meta := &pb.TransactionMeta{}

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
			meta = &pb.TransactionMeta{}
			return nil, meta, err
		}
	}

	ethTx, ok := tx.(*types2.EthTransaction)
	if !ok {
		return nil, nil, fmt.Errorf("tx is not in eth format")
	}

	return ethTx, meta, nil
}

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
// func (api *PublicEthereumAPI) PendingTransactions() ([]*rpctypes.RPCTransaction, error) {
// 	api.logger.Debug("eth_pendingTransactions")

// 	txs := api.api.Broker().GetPendingTransactions(1000)

// 	rpcTxs := make([]*rpctypes.RPCTransaction, len(txs))

// 	for _, tx := range txs {
// 		ethTx, ok := tx.(*types2.EthTransaction)
// 		if !ok {
// 			continue
// 		}
// 		rpcTxs = append(rpcTxs, newRPCTransaction(ethTx, common.Hash{}, 0, 0))
// 	}

// 	return rpcTxs, nil
// }

func getTxByBlockInfoAndIndex(api api.CoreAPI, mode string, key string, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	block, err := api.Broker().GetBlock(mode, key)
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

	meta, err := api.Broker().GetTransactionMeta(ethTx.GetHash())
	if err != nil {
		return nil, err
	}

	return newRPCTransaction(ethTx, common.BytesToHash(meta.BlockHash), meta.BlockHeight, meta.Index), nil
}

// FormatBlock creates an ethereum block from a tendermint header and ethereum-formatted
// transactions.
func formatBlock(api api.CoreAPI, config *repo.Config, block *pb.Block, fullTx bool) (map[string]interface{}, error) {
	cumulativeGas, err := getBlockCumulativeGas(api, block, uint64(len(block.Transactions.Transactions)-1))
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
		"logsBloom":        block.BlockHeader.Bloom,
		"transactionsRoot": block.BlockHeader.TxRoot,
		"stateRoot":        block.BlockHeader.StateRoot,
		"miner":            common.Address{},
		"mixHash":          common.Hash{},
		"difficulty":       (*hexutil.Big)(big.NewInt(0)),
		"totalDifficulty":  (*hexutil.Big)(big.NewInt(0)),
		"extraData":        []byte{},
		"size":             hexutil.Uint64(block.Size()),
		"gasLimit":         hexutil.Uint64(config.GasLimit), // Static gas limit
		"gasUsed":          hexutil.Uint64(cumulativeGas),
		"timestamp":        hexutil.Uint64(block.BlockHeader.Timestamp),
		"transactions":     transactions,
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
func getBlockCumulativeGas(api api.CoreAPI, block *pb.Block, idx uint64) (uint64, error) {
	var gasUsed uint64
	txs := block.Transactions.Transactions

	for i := 0; i <= int(idx) && i < len(txs); i++ {
		receipt, err := api.Broker().GetReceipt(txs[i].GetHash())
		if err != nil {
			return 0, err
		}

		gasUsed += receipt.GetGasUsed()
	}

	return gasUsed, nil
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
