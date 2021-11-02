package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) GetChainMeta(ctx context.Context, req *pb.Request) (*pb.ChainMeta, error) {
	return cbs.api.Chain().Meta()
}

func GetChainStatus(cbs *ChainBrokerService) (*pb.Response, error) {
	return &pb.Response{
		Data: []byte(cbs.api.Chain().Status()),
	}, nil
}

func GetValidators(cbs *ChainBrokerService) (*pb.Response, error) {
	admins := cbs.genesis.Admins
	addresses := make([]string, 0)
	for _, admin := range admins {
		addresses = append(addresses, admin.Address)
	}

	v, err := json.Marshal(addresses)
	if err != nil {
		return nil, fmt.Errorf("marshal admin adresses error: %w", err)
	}
	return &pb.Response{
		Data: v,
	}, nil
}
