package grpc

import (
	"context"
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) GetInfo(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	switch req.Type {
	case pb.Request_CHAIN_STATUS:
		return GetChainStatus(cbs)
	case pb.Request_NETWORK:
		return GetNetworkMeta(cbs)
	case pb.Request_VALIDATORS:
		return GetValidators(cbs)
	default:
		return nil, fmt.Errorf("wrong query type")
	}
}
