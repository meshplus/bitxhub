package coreapi

import (
	"fmt"
	"sync"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"go.uber.org/atomic"
)

type ChainAPI CoreAPI

var _ api.ChainAPI = (*ChainAPI)(nil)

func (api *ChainAPI) Status() string {
	err := api.bxh.Order.Ready()
	if err != nil {
		return "abnormal"
	}

	return "normal"
}

func (api *ChainAPI) Meta() (*pb.ChainMeta, error) {
	return api.bxh.Ledger.GetChainMeta(), nil
}

func (api *ChainAPI) TPS(begin, end uint64) (uint64, error) {
	var (
		errCount  atomic.Int64
		total     atomic.Uint64
		startTime int64
		endTime   int64
		wg        sync.WaitGroup
	)

	if int(begin) <= 0 {
		return 0, fmt.Errorf("begin number should be greater than zero")
	}

	if int(begin) >= int(end) {
		return 0, fmt.Errorf("begin number should be smaller than end number")
	}

	wg.Add(int(end - begin + 1))
	for i := begin + 1; i <= end-1; i++ {
		go func(height uint64, wg *sync.WaitGroup) {
			defer wg.Done()
			count, err := api.bxh.Ledger.GetTransactionCount(height)
			if err != nil {
				errCount.Inc()
			} else {
				total.Add(count)
			}
		}(i, &wg)
	}

	go func() {
		defer wg.Done()
		block, err := api.bxh.Ledger.GetBlock(begin)
		if err != nil {
			errCount.Inc()
		} else {
			total.Add(uint64(len(block.Transactions.Transactions)))
			startTime = block.BlockHeader.Timestamp
		}
	}()
	go func() {
		defer wg.Done()
		block, err := api.bxh.Ledger.GetBlock(end)
		if err != nil {
			errCount.Inc()
		} else {
			total.Add(uint64(len(block.Transactions.Transactions)))
			endTime = block.BlockHeader.Timestamp
		}
	}()
	wg.Wait()

	if errCount.Load() != 0 {
		return 0, fmt.Errorf("error during get block TPS")
	}

	elapsed := (endTime - startTime) / int64(time.Second)

	if elapsed <= 0 {
		return 0, fmt.Errorf("incorrect block timestamp")
	}
	return total.Load() / uint64(elapsed), nil
}
