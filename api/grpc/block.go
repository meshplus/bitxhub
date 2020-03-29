package grpc

import (
	"context"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) SyncMerkleWrapper(req *pb.SyncMerkleWrapperRequest, server pb.ChainBroker_SyncMerkleWrapperServer) error {
	c, err := cbs.api.Broker().AddPier(req.AppchainId)
	if err != nil {
		return err
	}

	for {
		select {
		case <-cbs.ctx.Done():
			break
		case bw, ok := <-c:
			if !ok {
				return nil
			}

			bs, err := bw.Marshal()
			if err != nil {
				cbs.api.Broker().RemovePier(req.AppchainId)
				break
			}

			if err := server.Send(&pb.Response{
				Data: bs,
			}); err != nil {
				cbs.api.Broker().RemovePier(req.AppchainId)
				break
			}
		}
	}
}

func (cbs *ChainBrokerService) GetMerkleWrapper(req *pb.GetMerkleWrapperRequest, server pb.ChainBroker_GetMerkleWrapperServer) error {
	ch := make(chan *pb.MerkleWrapper, req.End-req.Begin+1)
	if err := cbs.api.Broker().GetMerkleWrapper(req.Pid, req.Begin, req.End, ch); err != nil {
		return err
	}

	for {
		select {
		case <-cbs.ctx.Done():
			break
		case w := <-ch:
			data, err := w.Marshal()
			if err != nil {
				return err
			}

			if err := server.Send(&pb.Response{
				Data: data,
			}); err != nil {
				return err
			}

			if w.BlockHeader.Number == req.End {
				return nil
			}

		}
	}
}

func (cbs *ChainBrokerService) GetBlock(ctx context.Context, req *pb.GetBlockRequest) (*pb.Block, error) {
	return cbs.api.Broker().GetBlock(req.Type.String(), req.Value)
}

func (cbs *ChainBrokerService) GetBlocks(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlocksResponse, error) {
	blocks, err := cbs.api.Broker().GetBlocks(req.Offset, req.Length)
	if err != nil {
		return nil, err
	}

	return &pb.GetBlocksResponse{
		Blocks: blocks,
	}, nil
}
