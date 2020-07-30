package grpc

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) GetAssetExchangeSigns(ctx context.Context, req *pb.AssetExchangeSignsRequest) (*pb.Response, error) {
	var (
		wg     = sync.WaitGroup{}
		result = make(map[string][]byte)
	)

	wg.Add(1)
	go func(result map[string][]byte) {
		for k, v := range cbs.api.Broker().FetchAssetExchangeSignsFromOtherPeers(req.Id) {
			result[k] = v
		}
		wg.Done()
	}(result)

	address, sign, err := cbs.api.Broker().GetAssetExchangeSign(req.Id)
	wg.Wait()

	result[address] = sign
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	resp := &pb.Response{
		Data: data,
	}

	return resp, err
}
