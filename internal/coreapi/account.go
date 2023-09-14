package coreapi

import (
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/coreapi/api"
	"github.com/axiomesh/axiom/internal/ledger"
)

type AccountAPI CoreAPI

var _ api.AccountAPI = (*AccountAPI)(nil)

func (api *AccountAPI) GetAccount(addr *types.Address) ledger.IAccount {
	return api.axiom.ViewLedger.StateLedger.GetOrCreateAccount(addr)
}
