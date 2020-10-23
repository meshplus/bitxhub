package grpc

import (
	"context"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) GetReceipt(ctx context.Context, req *pb.TransactionHashMsg) (*pb.Receipt, error) {
	hash := types.NewHashByStr(req.TxHash)

	return cbs.api.Broker().GetReceipt(hash)
}
