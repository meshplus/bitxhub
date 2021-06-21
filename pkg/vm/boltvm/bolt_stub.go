package boltvm

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/pkg/vm"
	vm1 "github.com/meshplus/eth-kit/evm"
	"github.com/sirupsen/logrus"
)

var _ boltvm.Stub = (*BoltStubImpl)(nil)

type BoltStubImpl struct {
	bvm *BoltVM
	ctx *vm.Context
	ve  validator.Engine
}

func (b *BoltStubImpl) Caller() string {
	return b.ctx.Caller.String()
}

func (b *BoltStubImpl) Callee() string {
	return b.ctx.Callee.String()
}

func (b *BoltStubImpl) CurrentCaller() string {
	return b.ctx.CurrentCaller.String()
}

func (b *BoltStubImpl) Logger() logrus.FieldLogger {
	return b.ctx.Logger
}

// GetTxHash returns the transaction hash
func (b *BoltStubImpl) GetTxHash() *types.Hash {
	hash := b.ctx.TransactionHash
	return hash
}

func (b *BoltStubImpl) GetTxIndex() uint64 {
	return b.ctx.TransactionIndex
}

func (b *BoltStubImpl) Has(key string) bool {
	exist, _ := b.ctx.Ledger.GetState(b.ctx.Callee, []byte(key))
	return exist
}

func (b *BoltStubImpl) Get(key string) (bool, []byte) {
	return b.ctx.Ledger.GetState(b.ctx.Callee, []byte(key))
}

func (b *BoltStubImpl) Delete(key string) {
	b.ctx.Ledger.SetState(b.ctx.Callee, []byte(key), nil)
}

func (b *BoltStubImpl) GetObject(key string, ret interface{}) bool {
	ok, data := b.Get(key)
	if !ok {
		return ok
	}

	err := json.Unmarshal(data, ret)
	return err == nil
}

func (b *BoltStubImpl) Set(key string, value []byte) {
	b.ctx.Ledger.SetState(b.ctx.Callee, []byte(key), value)
}

func (b *BoltStubImpl) Add(key string, value []byte) {
	b.ctx.Ledger.AddState(b.ctx.Callee, []byte(key), value)
}

func (b *BoltStubImpl) SetObject(key string, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	b.Set(key, data)
}

func (b *BoltStubImpl) AddObject(key string, value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	b.Add(key, data)
}

func (b *BoltStubImpl) Query(prefix string) (bool, [][]byte) {
	return b.ctx.Ledger.QueryByPrefix(b.ctx.Callee, prefix)
}

func (b *BoltStubImpl) PostEvent(event interface{}) {
	b.postEvent(false, event)
}

func (b *BoltStubImpl) PostInterchainEvent(event interface{}) {
	b.postEvent(true, event)
}

func (b *BoltStubImpl) postEvent(interchain bool, event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}

	b.ctx.Ledger.AddEvent(&pb.Event{
		Interchain: interchain,
		Data:       data,
		TxHash:     b.GetTxHash(),
	})
}

func (b *BoltStubImpl) CrossInvoke(address, method string, args ...*pb.Arg) *boltvm.Response {
	addr := types.NewAddressByStr(address)

	payload := &pb.InvokePayload{
		Method: method,
		Args:   args,
	}

	ctx := &vm.Context{
		Caller:           b.bvm.ctx.Caller,
		Callee:           addr,
		CurrentCaller:    b.bvm.ctx.Callee,
		Ledger:           b.bvm.ctx.Ledger,
		TransactionIndex: b.bvm.ctx.TransactionIndex,
		TransactionHash:  b.bvm.ctx.TransactionHash,
		Logger:           b.bvm.ctx.Logger,
	}

	data, err := payload.Marshal()
	if err != nil {
		return boltvm.Error(err.Error())
	}
	bvm := New(ctx, b.ve, nil, b.bvm.contracts)
	ret, err := bvm.Run(data)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(ret)
}

func (b *BoltStubImpl) CrossInvokeEVM(address string, data []byte) *boltvm.Response {
	addr := types.NewAddressByStr(address)
	ctx := b.bvm.ctx

	tx := &pb.BxhTransaction{
		Version:         ctx.Tx.Version,
		From:            ctx.Tx.From,
		To:              addr,
		Timestamp:       ctx.Tx.Timestamp,
		TransactionHash: ctx.Tx.TransactionHash,
		Payload:         data,
		Nonce:           ctx.Tx.Nonce,
		Signature:       ctx.Tx.Signature,
		Extra:           ctx.Tx.Extra,
	}
	gp := new(core.GasPool).AddGas(10000000)
	msg := ledger.NewMessageFromBxh(tx)
	statedb := ctx.Ledger.StateLedger
	statedb.PrepareEVM(common.BytesToHash(ctx.TransactionHash.Bytes()), int(ctx.TransactionIndex))
	snapshot := statedb.Snapshot()
	txContext := vm1.NewEVMTxContext(msg)
	b.bvm.evm.Reset(txContext, statedb)
	result, err := vm1.ApplyMessage(b.bvm.evm, msg, gp)
	if err != nil {
		statedb.RevertToSnapshot(snapshot)
		ctx.Ledger.ClearChangerAndRefund()
		return boltvm.Error(err.Error())
	}
	if result.Failed() {
		return boltvm.Error(string(append([]byte(result.Err.Error()), result.Revert()...)))
	}
	ret := result.Return()
	ctx.Ledger.Finalise(false)
	return boltvm.Success(ret)
}

func (b *BoltStubImpl) ValidationEngine() validator.Engine {
	return b.ve
}

func (b *BoltStubImpl) GetAccount(address string) (bool, interface{}) {
	addr := types.NewAddressByStr(address)
	account := b.ctx.Ledger.GetAccount(addr)

	return account != nil, account
}
