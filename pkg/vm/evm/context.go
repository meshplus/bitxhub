package evm

import (
	"math/big"

	"github.com/meshplus/eth-kit/ledger"

	"github.com/ethereum/go-ethereum/params"
	vm "github.com/meshplus/eth-kit/evm"
)

type EvmBlockContext struct {
	BlkCtx   vm.BlockContext
	chainCfg *params.ChainConfig
	vmConfig vm.Config
}

func NewEvmBlockContext(number uint64, timestamp uint64, db ledger.StateLedger, db2 ledger.ChainLedger, admin string) *EvmBlockContext {
	blkCtx := vm.NewEVMBlockContext(number, timestamp, db, db2, admin)

	return &EvmBlockContext{
		BlkCtx:   blkCtx,
		chainCfg: &params.ChainConfig{ChainID: new(big.Int).SetInt64(1)},
		vmConfig: vm.Config{},
	}
}
