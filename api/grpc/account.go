package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type Account struct {
	Type          string     `json:"type"`
	Balance       *big.Int   `json:"balance"`
	ContractCount uint64     `json:"contract_count"`
	CodeHash      types.Hash `json:"code_hash"`
}

func (cbs *ChainBrokerService) GetAccountBalance(ctx context.Context, req *pb.Address) (*pb.Response, error) {
	if !types.IsValidAddressByte([]byte(req.Address)) {
		return nil, fmt.Errorf("invalid account address: %v", req.Address)
	}

	var ret *Account
	addr := types.NewAddressByStr(req.Address)

	account := cbs.api.Account().GetAccount(addr)
	if account == nil {
		ret = &Account{
			Type:          "normal",
			Balance:       big.NewInt(0),
			ContractCount: 0,
			CodeHash:      types.Hash{},
		}
	} else {
		hash := types.NewHash(account.CodeHash())
		typ := "contract"
		if account.CodeHash() == nil || bytes.Equal(account.CodeHash(), crypto.Keccak256(nil)) {
			typ = "normal"
		}

		ret = &Account{
			Type:          typ,
			Balance:       account.GetBalance(),
			ContractCount: account.GetNonce(),
			CodeHash:      *hash,
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return nil, fmt.Errorf("marshal account error: %w", err)
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
