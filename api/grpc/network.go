package grpc

import (
	"context"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) GetNetworkMeta(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	data, err := cbs.api.Network().PeerInfo()
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}
