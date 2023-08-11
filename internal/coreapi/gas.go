package coreapi

import (
	"github.com/axiomesh/axiom/internal/coreapi/api"
)

type GasAPI CoreAPI

var _ api.ChainAPI = (*ChainAPI)(nil)

func (gas *GasAPI) GetGasPrice() (uint64, error) {
	lastestHeight := gas.bxh.Ledger.ChainLedger.GetChainMeta().Height
	block, err := gas.bxh.Ledger.ChainLedger.GetBlock(lastestHeight)
	if err != nil {
		return 0, err
	}
	return uint64(block.BlockHeader.GasPrice), nil
}

func (gas *GasAPI) GetCurrentGasPrice(blockHeight uint64) (uint64, error) {
	// since all block header records the gas price of the next block, so here we need to dec to get the current block's gas price, genesis block will return its own gas price
	if blockHeight != 1 {
		blockHeight--
	}
	block, err := gas.bxh.Ledger.ChainLedger.GetBlock(blockHeight)
	if err != nil {
		return 0, err
	}
	return uint64(block.BlockHeader.GasPrice), nil
}
