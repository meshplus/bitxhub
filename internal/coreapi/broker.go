package coreapi

import (
	"errors"
	"fmt"
	"strconv"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/coreapi/api"
	"github.com/axiomesh/axiom/internal/executor/system"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/ledger"
	vm "github.com/axiomesh/eth-kit/evm"
)

type BrokerAPI CoreAPI

func (b *BrokerAPI) GetTotalPendingTxCount() uint64 {
	return b.axiom.Order.GetTotalPendingTxCount()
}

var _ api.BrokerAPI = (*BrokerAPI)(nil)

func (b *BrokerAPI) HandleTransaction(tx *types.Transaction) error {
	if tx.GetHash() == nil {
		return errors.New("transaction hash is nil")
	}

	b.logger.WithFields(logrus.Fields{
		"hash": tx.GetHash().String(),
	}).Debugf("Receive tx")

	if err := b.axiom.Order.Prepare(tx); err != nil {
		return fmt.Errorf("order prepare for tx %s failed: %w", tx.GetHash().String(), err)
	}

	return nil
}

func (b *BrokerAPI) GetTransaction(hash *types.Hash) (*types.Transaction, error) {
	return b.axiom.ViewLedger.ChainLedger.GetTransaction(hash)
}

func (b *BrokerAPI) GetTransactionMeta(hash *types.Hash) (*types.TransactionMeta, error) {
	return b.axiom.ViewLedger.ChainLedger.GetTransactionMeta(hash)
}

func (b *BrokerAPI) GetReceipt(hash *types.Hash) (*types.Receipt, error) {
	return b.axiom.ViewLedger.ChainLedger.GetReceipt(hash)
}

func (b *BrokerAPI) GetBlock(mode string, value string) (*types.Block, error) {
	switch mode {
	case "HEIGHT":
		height, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("wrong block number: %s", value)
		}
		return b.axiom.ViewLedger.ChainLedger.GetBlock(height)
	case "HASH":
		hash := types.NewHashByStr(value)
		if hash == nil {
			return nil, errors.New("invalid format of block hash for querying block")
		}
		return b.axiom.ViewLedger.ChainLedger.GetBlockByHash(hash)
	default:
		return nil, fmt.Errorf("wrong args about getting block: %s", mode)
	}
}

func (b *BrokerAPI) GetBlocks(start uint64, end uint64) ([]*types.Block, error) {
	meta := b.axiom.ViewLedger.ChainLedger.GetChainMeta()

	var blocks []*types.Block
	if meta.Height < end {
		end = meta.Height
	}
	for i := start; i > 0 && i <= end; i++ {
		b, err := b.GetBlock("HEIGHT", strconv.Itoa(int(i)))
		if err != nil {
			continue
		}
		blocks = append(blocks, b)
	}

	return blocks, nil
}

func (b *BrokerAPI) GetBlockHeaders(start uint64, end uint64) ([]*types.BlockHeader, error) {
	meta := b.axiom.ViewLedger.ChainLedger.GetChainMeta()

	var blockHeaders []*types.BlockHeader
	if meta.Height < end {
		end = meta.Height
	}
	for i := start; i > 0 && i <= end; i++ {
		b, err := b.GetBlock("HEIGHT", strconv.Itoa(int(i)))
		if err != nil {
			continue
		}
		blockHeaders = append(blockHeaders, b.BlockHeader)
	}

	return blockHeaders, nil
}

func (b *BrokerAPI) OrderReady() error {
	return b.axiom.Order.Ready()
}

func (b *BrokerAPI) GetPendingTxCountByAccount(account string) uint64 {
	return b.axiom.Order.GetPendingTxCountByAccount(account)
}

func (b *BrokerAPI) GetPoolTransaction(hash *types.Hash) *types.Transaction {
	return b.axiom.Order.GetPendingTxByHash(hash)
}

func (b *BrokerAPI) GetViewStateLedger() ledger.StateLedger {
	return b.axiom.ViewLedger.StateLedger
}

func (b *BrokerAPI) GetEvm(mes *vm.Message, vmConfig *vm.Config) (*vm.EVM, error) {
	if vmConfig == nil {
		vmConfig = new(vm.Config)
	}
	txContext := vm.NewEVMTxContext(mes)
	return b.axiom.BlockExecutor.NewEvmWithViewLedger(txContext, *vmConfig)
}

func (b *BrokerAPI) GetSystemContract(addr *ethcommon.Address) (common.SystemContract, bool) {
	if addr == nil {
		return nil, false
	}
	return system.GetSystemContract(types.NewAddress(addr.Bytes()))
}
