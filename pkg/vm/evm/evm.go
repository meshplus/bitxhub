package evm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/vm"
)

type EtherVM struct {
	ctx *vm.Context
}

func New(ctx *vm.Context) (*EtherVM, error) {
	return &EtherVM{
		ctx: ctx,
	}, nil
}

func NewMessage(tx *pb.EthTransaction) types.Message {
	from := common.BytesToAddress(tx.GetFrom().Bytes())
	to := common.BytesToAddress(tx.GetTo().Bytes())
	nonce := tx.GetNonce()
	amount := new(big.Int).SetUint64(tx.GetAmount())
	gas := tx.GetGas()
	gasPrice := tx.GetGasPrice()
	data := tx.GetPayload()
	accessList := tx.AccessList()
	return types.NewMessage(from, &to, nonce, amount, gas, gasPrice, data, accessList, true)
}

// func NewEVM(blockCtx vm.BlockContext, txCtx vm.TxContext, statedb vm.StateDB, chainConfig *params.ChainConfig, vmConfig vm.Config) *vm.evm {
// 	return vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, vmConfig)
// }
