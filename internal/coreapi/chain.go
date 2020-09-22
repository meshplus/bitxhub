package coreapi

import (
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
)

type ChainAPI CoreAPI

var _ api.ChainAPI = (*ChainAPI)(nil)

func (api *ChainAPI) Status() string {
	if api.bxh.Order.Ready() {
		return "normal"
	}

	return "abnormal"
}

func (api *ChainAPI) Meta() (*pb.ChainMeta, error) {
	return api.bxh.Ledger.GetChainMeta(), nil
}

func (api *ChainAPI) TPS(begin, end uint64) (uint64, error) {
	total := 0
	startTime := int64(0)
	endTime := int64(0)

	if begin >= end {
		return 0, fmt.Errorf("begin number should be smaller than end number")
	}

	for i := begin; i <= end; i++ {
		block, err := api.bxh.Ledger.GetBlock(i)
		if err != nil {
			return 0, err
		}

		total += len(block.Transactions)
		if i == begin {
			startTime = block.BlockHeader.Timestamp
		}
		if i == end {
			endTime = block.BlockHeader.Timestamp
		}
	}

	elapsed := endTime - startTime

	if elapsed <= 0 {
		return 0, fmt.Errorf("incorrect block timestamp")
	}

	return uint64(total) * 1e9 / uint64(elapsed), nil
}
