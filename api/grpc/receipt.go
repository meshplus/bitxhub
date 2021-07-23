package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (cbs *ChainBrokerService) GetReceipt(ctx context.Context, req *pb.TransactionHashMsg) (*pb.Receipt, error) {
	hash := types.NewHashByStr(req.TxHash)
	if hash == nil {
		return nil, status.Newf(codes.InvalidArgument, fmt.Sprintf("invalid format of receipt hash %s for querying receipt", hash)).Err()
	}
	r, err := cbs.api.Broker().GetReceipt(hash)
	if err != nil {
		if errors.Is(storage.ErrorNotFound, err) {
			return nil, status.Newf(codes.NotFound, fmt.Sprintf("cannot found receipt for %s", hash.String()), err.Error()).Err()
		}
		return nil, status.Newf(codes.Internal, "internal handling error", err.Error()).Err()
	}
	return r, nil
}
