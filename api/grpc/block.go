package grpc

import (
	"context"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) GetInterchainTxWrappers(req *pb.GetInterchainTxWrappersRequest, server pb.ChainBroker_GetInterchainTxWrappersServer) error {
	meta, err := cbs.api.Chain().Meta()
	if err != nil {
		return err
	}

	if meta.Height < req.End {
		req.End = meta.Height
	}

	ch := make(chan *pb.InterchainTxWrappers, req.End-req.Begin+1)
	if err := cbs.api.Broker().GetInterchainTxWrappers(req.Pid, req.Begin, req.End, ch); err != nil {
		return err
	}

	for {
		select {
		case <-cbs.ctx.Done():
			break
		case bw, ok := <-ch:
			if !ok {
				return nil
			}

			if err := server.Send(bw); err != nil {
				return err
			}
		}
	}
}

func (cbs *ChainBrokerService) GetBlockHeader(req *pb.GetBlockHeaderRequest, server pb.ChainBroker_GetBlockHeaderServer) error {
	meta, err := cbs.api.Chain().Meta()
	if err != nil {
		return err
	}

	if meta.Height < req.End {
		req.End = meta.Height
	}

	ch := make(chan *pb.BlockHeader, req.End-req.Begin+1)
	if err := cbs.api.Broker().GetBlockHeader(req.Begin, req.End, ch); err != nil {
		return err
	}

	for {
		select {
		case <-cbs.ctx.Done():
			break
		case w, ok := <-ch:
			// if channel is unexpected closed, return
			if !ok {
				return nil
			}

			if err := server.Send(w); err != nil {
				return err
			}

			if w.Number == req.End {
				return nil
			}

		}
	}
}

func (cbs *ChainBrokerService) GetBlock(ctx context.Context, req *pb.GetBlockRequest) (*pb.Block, error) {
	return cbs.api.Broker().GetBlock(req.Type.String(), req.Value)
}

func (cbs *ChainBrokerService) GetBlocks(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlocksResponse, error) {
	blocks, err := cbs.api.Broker().GetBlocks(req.Start, req.End)
	if err != nil {
		return nil, err
	}

	return &pb.GetBlocksResponse{
		Blocks: blocks,
	}, nil
}
