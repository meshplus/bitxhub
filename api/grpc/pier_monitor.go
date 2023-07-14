package grpc

import (
	"context"
	"fmt"
	"github.com/meshplus/bitxhub-model/constant"

	"github.com/meshplus/bitxhub-model/pb"
)

func (cbs *ChainBrokerService) CheckMasterPier(ctx context.Context, req *pb.Address) (*pb.Response, error) {
	resp := &pb.CheckPierResponse{}
	index := cbs.checkMasterPier(req.Address)

	if index != constant.NoMaster {
		resp.Status = pb.CheckPierResponse_HAS_MASTER
	} else {
		resp.Status = pb.CheckPierResponse_NO_MASTER
	}
	resp.Address = req.Address
	resp.Index = index

	data, err := resp.Marshal()
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) SetMasterPier(ctx context.Context, req *pb.PierInfo) (*pb.Response, error) {
	index := cbs.checkMasterPier(req.Address)
	if index < req.Index && index != constant.NoMaster {
		return nil, fmt.Errorf("master pier already exist: %s", index)
	}

	current, err := cbs.api.Network().PierManager().Piers().SetMaster(req.Address, req.Index, req.Timeout)
	if err != nil {
		cbs.logger.Errorf("failed to become master, current %s, err %s", current, err.Error())
		return nil, err
	}
	resp := &pb.CheckPierResponse{
		Address: req.Address,
		Index:   current,
	}

	data, err := resp.Marshal()
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) HeartBeat(ctx context.Context, req *pb.PierInfo) (*pb.Response, error) {

	current, err := cbs.api.Network().PierManager().Piers().HeartBeat(req.Address, req.Index)
	if err != nil {
		return nil, err
	}

	var masters []string
	masters, err = cbs.api.Network().PierManager().AskPierMaster(req.Address)
	if err != nil {
		cbs.logger.Error(err)
		return nil, err
	}

	resp := &pb.CheckPierResponse{
		Address: req.Address,
		Index:   current,
	}

	if len(masters) == 0 {
		resp.Index = current
	} else {
		if current == constant.NoMaster {
			resp.Index = masters[0]
		} else {
			resp.Index = minStr(current, masters[0])
		}
	}

	cbs.logger.Infof("heartbeat finish, selected master: %s", resp.Index)

	data, err := resp.Marshal()
	if err != nil {
		return nil, err
	}

	return &pb.Response{
		Data: data,
	}, nil
}

func (cbs *ChainBrokerService) checkMasterPier(address string) (id string) {
	pmgr := cbs.api.Network().PierManager()
	if pmgr.Piers().HasPier(address) {
		id = pmgr.Piers().CheckMaster(address)
	} else {
		masters, err := pmgr.AskPierMaster(address)
		if err != nil || len(masters) == 0 {
			cbs.logger.Errorf("err: %v, masters: %v", err, masters)
			return constant.NoMaster
		}
		id = masters[0]
	}
	return
}

func minStr(a, b string) string {
	if a < b {
		return a
	}
	return b
}
