package coreapi

import (
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/sirupsen/logrus"
)

type BrokerAPI CoreAPI

var _ api.BrokerAPI = (*BrokerAPI)(nil)

func (b *BrokerAPI) HandleTransaction(tx *pb.Transaction) error {
	b.logger.WithFields(logrus.Fields{
		"hash": tx.TransactionHash.String(),
	}).Debugf("receive tx")

	go func() {
		if err := b.bxh.Order.Prepare(tx); err != nil {
			b.logger.Error(err)
		}
	}()

	return nil
}

func (b *BrokerAPI) GetTransaction(hash types.Hash) (*pb.Transaction, error) {
	return b.bxh.Ledger.GetTransaction(hash)
}

func (b *BrokerAPI) GetTransactionMeta(hash types.Hash) (*pb.TransactionMeta, error) {
	return b.bxh.Ledger.GetTransactionMeta(hash)
}

func (b *BrokerAPI) GetReceipt(hash types.Hash) (*pb.Receipt, error) {
	return b.bxh.Ledger.GetReceipt(hash)
}

func (b *BrokerAPI) AddPier(key string) (chan *pb.InterchainTxWrapper, error) {
	return b.bxh.Router.AddPier(key)
}

func (b *BrokerAPI) GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error {
	return b.bxh.Router.GetBlockHeader(begin, end, ch)
}

func (b *BrokerAPI) GetInterchainTxWrapper(pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrapper) error {
	return b.bxh.Router.GetInterchainTxWrapper(pid, begin, end, ch)
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
		return b.bxh.Ledger.GetBlockByHash(types.String2Hash(value))
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

func (b *BrokerAPI) RemovePier(key string) {
	b.bxh.Router.RemovePier(key)
}

func (b *BrokerAPI) OrderReady() bool {
	return b.bxh.Order.Ready()
}
