package adaptor

import (
	"github.com/samber/lo"
)

type RBFTTXPoolAdaptor struct {
}

func NewRBFTTXPoolAdaptor() (*RBFTTXPoolAdaptor, error) {
	return &RBFTTXPoolAdaptor{}, nil
}

// TODO: implement it, check by ledger
func (a *RBFTTXPoolAdaptor) IsRequestsExist(txs [][]byte) []bool {
	return lo.Map(txs, func(item []byte, index int) bool {
		return false
	})
}

func (a *RBFTTXPoolAdaptor) CheckSigns(txs [][]byte) {}
