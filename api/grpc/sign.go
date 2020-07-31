package grpc

import (
	"context"
	"sync"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

func (cbs *ChainBrokerService) GetAssetExchangeSigns(ctx context.Context, req *pb.AssetExchangeSignsRequest) (*pb.SignResponse, error) {
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

	if err != nil {
		cbs.logger.WithFields(logrus.Fields{
			"id":  req.Id,
			"err": err.Error(),
		}).Warnf("Get asset exchange sign on current node")
	} else {
		result[address] = sign
	}

	return &pb.SignResponse{
		Sign: result,
	}, nil
}
