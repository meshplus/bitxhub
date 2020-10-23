package grpc

import (
	"context"
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) GetReceipt(ctx context.Context, req *pb.TransactionHashMsg) (*pb.Receipt, error) {
	hash := types.NewHashByStr(req.TxHash)
	if hash == nil {
		return nil, fmt.Errorf("invalid format of receipt hash for querying receipt")
	}
	return cbs.api.Broker().GetReceipt(hash)
}
