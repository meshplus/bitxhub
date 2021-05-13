package coreapi

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/ledger"
)

type AccountAPI CoreAPI

var _ api.AccountAPI = (*AccountAPI)(nil)

func (api *AccountAPI) GetAccount(addr *types.Address) *ledger.Account {
	return api.bxh.Ledger.GetAccount(addr)
}
