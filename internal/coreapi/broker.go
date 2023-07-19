package coreapi

import (
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	vm "github.com/meshplus/eth-kit/evm"
	"github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

type BrokerAPI CoreAPI

var _ api.BrokerAPI = (*BrokerAPI)(nil)

func (b *BrokerAPI) HandleTransaction(tx pb.Transaction) error {
	if tx.GetHash() == nil {
		return fmt.Errorf("transaction hash is nil")
	}

	b.logger.WithFields(logrus.Fields{
		"hash": tx.GetHash().String(),
	}).Debugf("Receive tx")

	if err := b.bxh.Order.Prepare(tx); err != nil {
		b.logger.Errorf("order prepare for tx %s failed: %s", tx.GetHash().String(), err.Error())
		return fmt.Errorf("order prepare for tx %s failed: %w", tx.GetHash().String(), err)
	}

	return nil
}

func (b *BrokerAPI) HandleView(tx pb.Transaction) (*pb.Receipt, error) {
	if tx.GetHash() == nil {
		return nil, fmt.Errorf("transaction hash is nil")
	}

	b.logger.WithFields(logrus.Fields{
		"hash": tx.GetHash().String(),
	}).Debugf("Receive view")

	receipts := b.bxh.ViewExecutor.ApplyReadonlyTransactions([]pb.Transaction{tx})

	return receipts[0], nil
}

func (b *BrokerAPI) GetTransaction(hash *types.Hash) (pb.Transaction, error) {
	return b.bxh.Ledger.GetTransaction(hash)
}

func (b *BrokerAPI) GetTransactionMeta(hash *types.Hash) (*pb.TransactionMeta, error) {
	return b.bxh.Ledger.GetTransactionMeta(hash)
}

func (b *BrokerAPI) GetReceipt(hash *types.Hash) (*pb.Receipt, error) {
	return b.bxh.Ledger.GetReceipt(hash)
}

func (b *BrokerAPI) GetBlock(mode string, value string) (*pb.Block, error) {
	switch mode {
	case "HEIGHT":
		height, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("wrong block number: %s", value)
		}
		return b.bxh.Ledger.GetBlock(height)
	case "HASH":
		hash := types.NewHashByStr(value)
		if hash == nil {
			return nil, fmt.Errorf("invalid format of block hash for querying block")
		}
		return b.bxh.Ledger.GetBlockByHash(hash)
	default:
		return nil, fmt.Errorf("wrong args about getting block: %s", mode)
	}
}

func (b *BrokerAPI) GetBlocks(start uint64, end uint64) ([]*pb.Block, error) {
	meta := b.bxh.Ledger.GetChainMeta()

	var blocks []*pb.Block
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

func (b *BrokerAPI) GetBlockHeaders(start uint64, end uint64) ([]*pb.BlockHeader, error) {
	meta := b.bxh.Ledger.GetChainMeta()

	var blockHeaders []*pb.BlockHeader
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
	return b.bxh.Order.Ready()
}

func (b BrokerAPI) GetPendingNonceByAccount(account string) uint64 {
	return b.bxh.Order.GetPendingNonceByAccount(account)
}

func (b BrokerAPI) GetPendingTransactions(max int) []pb.Transaction {
	// TODO
	return nil
}

func (b BrokerAPI) GetPoolTransaction(hash *types.Hash) pb.Transaction {
	return b.bxh.Order.GetPendingTxByHash(hash)
}

func (b BrokerAPI) GetStateLedger() ledger.StateLedger {
	return b.bxh.Ledger.StateLedger
}

func (b BrokerAPI) GetEvm(mes *vm.Message, vmConfig *vm.Config) *vm.EVM {
	if vmConfig == nil {
		vmConfig = new(vm.Config)
	}
	txContext := vm.NewEVMTxContext(mes)
	return b.bxh.BlockExecutor.GetEvm(txContext, *vmConfig)
}
