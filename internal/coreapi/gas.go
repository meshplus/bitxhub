package coreapi

import "github.com/meshplus/bitxhub/internal/coreapi/api"

type GasAPI CoreAPI

var _ api.ChainAPI = (*ChainAPI)(nil)

func (gas *GasAPI) GetGasPrice() (uint64, error) {
	return gas.bxh.Gas.GetGasPrice()
}
