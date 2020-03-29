package grpc

import (
	"context"
	"encoding/json"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type Account struct {
	Type          string     `json:"type"`
	Balance       uint64     `json:"balance"`
	ContractCount uint64     `json:"contract_count"`
	CodeHash      types.Hash `json:"code_hash"`
}

func (cbs *ChainBrokerService) GetAccountBalance(ctx context.Context, req *pb.Address) (*pb.Response, error) {
	addr := types.String2Address(req.Address)

	account := cbs.api.Account().GetAccount(addr)

	hash := types.Bytes2Hash(account.CodeHash)

	typ := "normal"

	if account.CodeHash != nil {
		typ = "contract"
	}

	ret := &Account{
		Type:          typ,
		Balance:       account.Balance,
		ContractCount: account.Nonce,
		CodeHash:      hash,
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}
