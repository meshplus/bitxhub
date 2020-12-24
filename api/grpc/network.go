package grpc

import (
	"github.com/meshplus/bitxhub-model/pb"
)

func GetNetworkMeta(cbs *ChainBrokerService) (*pb.Response, error) {
	if cbs.config.Solo {
		return &pb.Response{}, nil
	}
	data, err := cbs.api.Network().PeerInfo()
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}
