package coreapi

import (
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
)

type ChainAPI CoreAPI

var _ api.ChainAPI = (*ChainAPI)(nil)

func (api *ChainAPI) Status() string {
	if api.bxh.Order.Ready() {
		return "normal"
	}

	return "abnormal"
}

func (api *ChainAPI) Meta() (*pb.ChainMeta, error) {
	return api.bxh.Ledger.GetChainMeta(), nil
}
