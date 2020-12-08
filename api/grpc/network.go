package grpc

import (
	"context"

	"github.com/meshplus/bitxhub-model/pb"
)

func GetNetworkMeta(cbs *ChainBrokerService) (*pb.Response, error) {
	data, err := cbs.api.Network().PeerInfo()
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) DelVPNode(ctx context.Context, req *pb.DelVPNodeRequest) (*pb.Response, error) {
	data, err := cbs.api.Network().DelVPNode(req.Pid)
	if err != nil {
		return nil, err
	}
	return &pb.Response{
		Data: data,
	}, nil
}
