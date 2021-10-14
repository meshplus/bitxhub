package router

import (
	"encoding/json"
	"fmt"

	appchain_mgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

func (router *InterchainRouter) queryAllAppchains() (map[string]*appchain_mgr.Appchain, error) {
	ret := make(map[string]*appchain_mgr.Appchain, 0)
	ok, value := router.ledger.Copy().QueryByPrefix(constant.AppchainMgrContractAddr.Address(), appchain_mgr.Prefix)
	if !ok {
		return ret, nil
	}

	for _, data := range value {
		chain := &appchain_mgr.Appchain{}
		if err := json.Unmarshal(data, chain); err != nil {
			router.logger.Errorf("unmarshal appchain error:%v", err)
			return nil, fmt.Errorf("unmarshal appchain error:%v", err)
		}
		// TODO
		//if chain.ChainType == appchain_mgr.RelaychainType {
		//	continue
		//}
		ret[chain.ID] = chain
	}
	return ret, nil
}

func (router *InterchainRouter) generateUnionInterchainTxWrappers(ret map[string]*pb.InterchainTxWrapper, block *pb.Block, meta *pb.InterchainMeta) *pb.InterchainTxWrappers {
	wrappers := make([]*pb.InterchainTxWrapper, 0)
	emptyWrapper := &pb.InterchainTxWrapper{
		Height:  block.Height(),
		L2Roots: meta.L2Roots,
	}

	appchains, err := router.queryAllAppchains()
	if err != nil {
		wrappers = append(wrappers, emptyWrapper)
		return &pb.InterchainTxWrappers{
			InterchainTxWrappers: wrappers,
		}
	}
	for pid, _ := range ret {
		if _, ok := appchains[pid]; ok {
			delete(ret, pid)
		}
	}

	for _, interchainTxWrapper := range ret {
		wrappers = append(wrappers, interchainTxWrapper)
	}
	if len(wrappers) == 0 {
		wrappers = append(wrappers, emptyWrapper)
	}
	return &pb.InterchainTxWrappers{
		InterchainTxWrappers: wrappers,
	}
}
