package coreapi

import "github.com/axiomesh/axiom/internal/coreapi/api"

type GasAPI CoreAPI

var _ api.ChainAPI = (*ChainAPI)(nil)

func (gas *GasAPI) GetGasPrice() (uint64, error) {
	return gas.bxh.Gas.GetGasPrice()
}
