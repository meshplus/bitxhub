package grpc

import (
	"context"
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) CheckMasterPier(ctx context.Context, req *pb.Address) (*pb.Response, error) {
	resp := &pb.CheckPierResponse{}
	ret, err := cbs.checkMasterPier(req.Address)
	if err != nil {
		return nil, fmt.Errorf("check %s master pier failed: %w", req.Address, err)
	}
	if ret {
		resp.Status = pb.CheckPierResponse_HAS_MASTER
	} else {
		resp.Status = pb.CheckPierResponse_NO_MASTER
	}
	resp.Address = req.Address

	data, err := resp.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal check pier response error: %w", err)
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) SetMasterPier(ctx context.Context, req *pb.PierInfo) (*pb.Response, error) {
	ret, err := cbs.checkMasterPier(req.Address)
	if err != nil {
		return nil, fmt.Errorf("check %s master pier failed: %w", req.Address, err)
	}
	if ret {
		return nil, fmt.Errorf("master pier already exist")
	}
	err = cbs.api.Network().PierManager().Piers().SetMaster(req.Address, req.Index, req.Timeout)
	if err != nil {
		return nil, fmt.Errorf("set %s master pier failed: %w", req.Address, err)
	}
	resp := &pb.CheckPierResponse{
		Address: req.Address,
	}

	data, err := resp.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal check pier response error: %w", err)
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) HeartBeat(ctx context.Context, req *pb.PierInfo) (*pb.Response, error) {
	err := cbs.api.Network().PierManager().Piers().HeartBeat(req.Address, req.Index)
	if err != nil {
		return nil, fmt.Errorf("send heart beat to %s with index %d failed: %w", req.Address, req.Index, err)
	}

	resp := &pb.CheckPierResponse{
		Address: req.Address,
	}

	data, err := resp.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal check pier response error: %w", err)
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) checkMasterPier(address string) (bool, error) {
	pmgr := cbs.api.Network().PierManager()
	if pmgr.Piers().HasPier(address) {
		// cbs.logger.Infoln("native master")
		return pmgr.Piers().CheckMaster(address), nil
	} else {
		// cbs.logger.Infoln("remote master")
		return pmgr.AskPierMaster(address)
	}
}
