package coreapi

import (
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	ledger2 "github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/eth-kit/ledger"
)

type AccountAPI CoreAPI

var _ api.AccountAPI = (*AccountAPI)(nil)

func (api *AccountAPI) GetAccount(addr *types.Address) ledger.IAccount {
	switch l := api.bxh.Ledger.StateLedger.(type) {
	case *ledger2.SimpleLedger:
		return l.GetOrCreateAccount(addr)
	case *ledger.ComplexStateLedger:
		return l.Copy().GetOrCreateAccount(addr)
	}
	return nil
}
