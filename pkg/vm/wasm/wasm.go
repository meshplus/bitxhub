package wasm

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-core/wasm"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

var (
	errorLackOfMethod = fmt.Errorf("wasm execute: lack of method name")
)

var _ vm.VM = (*WasmVM)(nil)

// Wasm represents the wasm vm in BitXHub
type WasmVM struct {
	// contract context
	ctx *vm.Context

	// wasm
	w *wasm.Wasm
}

// Contract represents the smart contract structure used in the wasm vm
type Contract struct {
	// contract byte
	Code []byte

	// contract hash
	Hash types.Hash
}

// New creates a wasm vm instance
func New(ctx *vm.Context, imports *wasmer.Imports, instances map[string]wasmer.Instance) (*WasmVM, error) {
	wasmVM := &WasmVM{
		ctx: ctx,
	}

	if ctx.Callee == nil || bytes.Equal(ctx.Callee.Bytes(), (&types.Address{}).Bytes()) {
		return wasmVM, nil
	}

	contractByte := ctx.Ledger.GetCode(ctx.Callee)

	syncInstances := sync.Map{}
	for k, instance := range instances {
		syncInstances.Store(k, instance)
	}

	w, err := wasm.New(contractByte, imports, &syncInstances)
	if err != nil {
		return nil, err
	}

	w.SetContext(wasm.ACCOUNT, ctx.Ledger.GetOrCreateAccount(ctx.Callee))

	_, ok := w.Instance.Exports["allocate"]
	if !ok {
		return nil, fmt.Errorf("no allocate method")
	}
	w.SetContext(wasm.ALLOC_MEM, w.Instance.Exports["allocate"])
	wasmVM.w = w

	return wasmVM, nil
}

func EmptyImports() (*wasmer.Imports, error) {
	return wasmer.NewImports(), nil
}

// Run let the wasm vm excute or deploy the smart contract which depends on whether the callee is empty
func (w *WasmVM) Run(input []byte) (ret []byte, err error) {
	if w.ctx.Callee == nil || bytes.Equal(w.ctx.Callee.Bytes(), (&types.Address{}).Bytes()) {
		return w.deploy()
	}

	return w.w.Execute(input)
}

func (w *WasmVM) deploy() ([]byte, error) {
	if len(w.ctx.TransactionData.Payload) == 0 {
		return nil, fmt.Errorf("contract cannot be empty")
	}
	contractNonce := w.ctx.Ledger.GetNonce(w.ctx.Caller)

	contractAddr := createAddress(w.ctx.Caller, contractNonce)
	wasmStruct := &Contract{
		Code: w.ctx.TransactionData.Payload,
		Hash: *types.NewHash(w.ctx.TransactionData.Payload),
	}
	wasmByte, err := json.Marshal(wasmStruct)
	if err != nil {
		return nil, err
	}
	w.ctx.Ledger.SetCode(contractAddr, wasmByte)

	w.ctx.Ledger.SetNonce(w.ctx.Caller, contractNonce+1)

	return contractAddr.Bytes(), nil
}

func createAddress(b *types.Address, nonce uint64) *types.Address {
	var data []byte
	nonceBytes := make([]byte, 8)

	binary.LittleEndian.PutUint64(nonceBytes, nonce)
	data = append(data, b.Bytes()...)
	data = append(data, nonceBytes...)
	hashBytes := sha256.Sum256(data)

	return types.NewAddress(hashBytes[12:])
}
