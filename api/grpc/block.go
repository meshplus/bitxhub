package grpc

import (
	"context"
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
	types2 "github.com/meshplus/eth-kit/types"
)

func (cbs *ChainBrokerService) GetInterchainTxWrappers(req *pb.GetInterchainTxWrappersRequest, server pb.ChainBroker_GetInterchainTxWrappersServer) error {
	meta, err := cbs.api.Chain().Meta()
	if err != nil {
		return fmt.Errorf("get chain meta from ledger failed: %w", err)
	}

	if meta.Height < req.End {
		req.End = meta.Height
	}

	ch := make(chan *pb.InterchainTxWrappers, req.End-req.Begin+1)
	if err := cbs.api.Broker().GetInterchainTxWrappers(req.Pid, req.Begin, req.End, ch); err != nil {
		return fmt.Errorf("get interchain tx wrappers from router failed: %w", err)
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
				return fmt.Errorf("send interchain tx wrappers failed: %w", err)
			}
		}
	}
}

func (cbs *ChainBrokerService) GetBlockHeader(req *pb.GetBlockHeaderRequest, server pb.ChainBroker_GetBlockHeaderServer) error {
	meta, err := cbs.api.Chain().Meta()
	if err != nil {
		return fmt.Errorf("get chain meta from ledger failed: %w", err)
	}

	if meta.Height < req.End {
		req.End = meta.Height
	}

	ch := make(chan *pb.BlockHeader, req.End-req.Begin+1)
	if err := cbs.api.Broker().GetBlockHeader(req.Begin, req.End, ch); err != nil {
		return fmt.Errorf("get block header from router failed: %w", err)
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
				return fmt.Errorf("send block header failed: %w", err)
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
		return nil, fmt.Errorf("get blocks failed: %w", err)
	}

	return &pb.GetBlocksResponse{
		Blocks: blocks,
	}, nil
}

func (cbs *ChainBrokerService) GetHappyBlocks(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetHappyBlocksResponse, error) {
	blocks, err := cbs.api.Broker().GetBlocks(req.Start, req.End)
	if err != nil {
		return nil, fmt.Errorf("get blocks failed: %w", err)
	}
	happyBlocks := make([]*pb.HappyBlock, 0, len(blocks))
	for _, block := range blocks {
		bxhTxs := make([]*pb.BxhTransaction, 0)
		ethTxs := make([][]byte, 0)
		index := make([]uint64, 0, len(block.Transactions.Transactions))
		for _, tx := range block.Transactions.Transactions {
			if bxhTx, ok := tx.(*pb.BxhTransaction); ok {
				bxhTxs = append(bxhTxs, bxhTx)
				index = append(index, 0)
			} else if ethTx, ok := tx.(*types2.EthTransaction); ok {
				ethTxs = append(ethTxs, ethTx.GetHash().Bytes())
				index = append(index, 1)
			} else {
				return nil, fmt.Errorf("unsupport tx type")
			}
		}
		happyBlocks = append(happyBlocks, &pb.HappyBlock{
			BlockHeader: block.BlockHeader,
			BxhTxs:      bxhTxs,
			EthTxs:      ethTxs,
			Index:       index,
			BlockHash:   block.BlockHash,
			Signature:   block.Signature,
			Extra:       block.Extra,
		})
	}

	return &pb.GetHappyBlocksResponse{
		Blocks: happyBlocks,
	}, nil
}

func (cbs *ChainBrokerService) GetBlockHeaders(ctx context.Context, req *pb.GetBlockHeadersRequest) (*pb.GetBlockHeadersResponse, error) {
	headers, err := cbs.api.Broker().GetBlockHeaders(req.Start, req.End)
	if err != nil {
		return nil, fmt.Errorf("get block headers failed: %w", err)
	}

	return &pb.GetBlockHeadersResponse{
		BlockHeaders: headers,
	}, nil
}
