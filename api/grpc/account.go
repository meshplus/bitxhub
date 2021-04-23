package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
	if !types.IsValidAddressByte([]byte(req.Address)) {
		return nil, fmt.Errorf("invalid account address: %v", req.Address)
	}

	addr := types.NewAddressByStr(req.Address)

	account := cbs.api.Account().GetAccount(addr)

	hash := types.NewHash(account.CodeHash())

	typ := "normal"

	if account.CodeHash() != nil {
		typ = "contract"
	}

	ret := &Account{
		Type:          typ,
		Balance:       account.GetBalance(),
		ContractCount: account.GetNonce(),
		CodeHash:      *hash,
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) GetPendingNonceByAccount(ctx context.Context, req *pb.Address) (*pb.Response, error) {
	if !types.IsValidAddressByte([]byte(req.Address)) {
		return nil, fmt.Errorf("invalid account address: %v", req.Address)
	}
	nonce := cbs.api.Broker().GetPendingNonceByAccount(req.Address)
	return &pb.Response{
		Data: []byte(strconv.FormatUint(nonce, 10)),
	}, nil
}
