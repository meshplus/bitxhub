package grpc

import (
	"context"
	"encoding/binary"
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

func (cbs *ChainBrokerService) GetTPS(ctx context.Context, req *pb.GetTPSRequest) (*pb.Response, error) {
	tps, err := cbs.api.Chain().TPS(req.Begin, req.End)
	if err != nil {
		return nil, fmt.Errorf("get tps between %d and %d failed: %w", req.Begin, req.End, err)
	}

	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, tps)

	return &pb.Response{Data: data}, nil
}

func (cbs *ChainBrokerService) GetChainID(ctx context.Context, empty *pb.Empty) (*pb.Response, error) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, cbs.config.Genesis.ChainID)

	return &pb.Response{Data: data}, nil
}
