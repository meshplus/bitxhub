package grpc

import (
	"context"
	"encoding/json"

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
	addresses := cbs.genesis.Addresses
	v, err := json.Marshal(addresses)
	if err != nil {
		return nil, err
	}
	return &pb.Response{
		Data: v,
	}, nil
}
