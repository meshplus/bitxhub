package coreapi

import "github.com/axiomesh/axiom/internal/coreapi/api"

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
