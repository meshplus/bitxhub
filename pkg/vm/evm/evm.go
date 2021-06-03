package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/meshplus/bitxhub/pkg/vm"
	types2 "github.com/meshplus/eth-kit/types"
)

type EtherVM struct {
	ctx *vm.Context
}

func New(ctx *vm.Context) (*EtherVM, error) {
	return &EtherVM{
		ctx: ctx,
	}, nil
}

func NewMessage(tx *types2.EthTransaction) types.Message {
	from := common.BytesToAddress(tx.GetFrom().Bytes())
	to := common.BytesToAddress(tx.GetTo().Bytes())
	nonce := tx.GetNonce()
	amount := tx.GetAmount()
	gas := tx.GetGas()
	gasPrice := tx.GetGasPrice()
	data := tx.GetPayload()
	accessList := tx.GetInner().GetAccessList()
	return types.NewMessage(from, &to, nonce, amount, gas, gasPrice, data, accessList, true)
}

// func NewEVM(blockCtx vm.BlockContext, txCtx vm.TxContext, statedb vm.StateDB, chainConfig *params.ChainConfig, vmConfig vm.Config) *vm.evm {
// 	return vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, vmConfig)
// }
