package grpc

import (
	"context"
	"sync"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

func (cbs *ChainBrokerService) GetMultiSigns(ctx context.Context, req *pb.GetMultiSignsRequest) (*pb.SignResponse, error) {
	var (
		wg     = sync.WaitGroup{}
		result = make(map[string][]byte)
	)

	wg.Add(1)
	go func(result map[string][]byte) {
		for k, v := range cbs.api.Broker().FetchSignsFromOtherPeers(req.Content, req.Type) {
			result[k] = v
		}
		wg.Done()
	}(result)

	address, sign, err := cbs.api.Broker().GetSign(req.Content, req.Type)
	wg.Wait()

	if err != nil {
		cbs.logger.WithFields(logrus.Fields{
			"id":  req.Content,
			"err": err.Error(),
		}).Warnf("Get sign on current node")
	} else {
		result[address] = sign
	}

	return &pb.SignResponse{
		Sign: result,
	}, nil

}
