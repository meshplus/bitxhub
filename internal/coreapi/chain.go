package coreapi

import (
	"fmt"
	"runtime"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/pkg/utils"
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

func (api *ChainAPI) TPS(begin, end uint64) (*pb.GetTPSResponse, error) {
	var (
		errCount  atomic.Int64
		total     atomic.Uint64
		startTime int64
		endTime   int64
	)

	pool := utils.NewGoPool(runtime.GOMAXPROCS(runtime.NumCPU()))

	meta := api.bxh.Ledger.GetChainMeta()
	latestHeight := meta.Height
	if int(begin) <= 0 {
		return nil, fmt.Errorf("begin number should be greater than zero")
	}

	if int(begin) >= int(end) {
		return nil, fmt.Errorf("begin number should be smaller than end number")
	}

	if end > latestHeight {
		return nil, fmt.Errorf("end number:[%d] should be smaller than latest block number:[%d]", end, latestHeight)
	}

	// calculate all tx counts
	for i := begin + 1; i <= end-1; i++ {
		pool.Add()
		go func(height uint64, pool *utils.Pool) {
			defer pool.Done()
			count, err := api.bxh.Ledger.GetTransactionCount(height)
			if err != nil {
				errCount.Inc()
			} else {
				total.Add(count)
			}
		}(i, pool)
	}

	// get begin block
	pool.Add()
	go func(pool *utils.Pool) {
		defer pool.Done()
		block, err := api.bxh.Ledger.GetBlock(begin, false)
		if err != nil {
			errCount.Inc()
		} else {
			total.Add(uint64(len(block.Transactions.Transactions)))
			startTime = block.BlockHeader.Timestamp
		}
	}(pool)

	// get end block
	pool.Add()
	go func(pool *utils.Pool) {
		defer pool.Done()
		block, err := api.bxh.Ledger.GetBlock(end, false)
		if err != nil {
			errCount.Inc()
		} else {
			total.Add(uint64(len(block.Transactions.Transactions)))
			endTime = block.BlockHeader.Timestamp
		}
	}(pool)

	pool.Wait()

	if errCount.Load() != 0 {
		return nil, fmt.Errorf("error during get block TPS")
	}

	elapsed := float32(endTime-startTime) / float32(time.Second)

	if elapsed <= 0 {
		return nil, fmt.Errorf("incorrect block timestamp")
	}
	tps := float32(total.Load()) / elapsed
	return &pb.GetTPSResponse{
		TxCount: total.Load(),
		Tps:     tps,
	}, nil
}
