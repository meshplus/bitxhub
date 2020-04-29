package wasm

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-kit/wasm"
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

	if ctx.Callee == (types.Address{}) {
		return wasmVM, nil
	}

	contractByte := ctx.Ledger.GetCode(ctx.Callee)

	w, err := wasm.New(contractByte, imports, instances)
	if err != nil {
		return nil, err
	}

	wasmVM.w = w

	return wasmVM, nil
}

func EmptyImports() (*wasmer.Imports, error) {
	return wasmer.NewImports(), nil
}

// Run let the wasm vm excute or deploy the smart contract which depends on whether the callee is empty
func (w *WasmVM) Run(input []byte) (ret []byte, err error) {
	if w.ctx.Callee == (types.Address{}) {
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
		Hash: types.Bytes2Hash(w.ctx.TransactionData.Payload),
	}
	wasmByte, err := json.Marshal(wasmStruct)
	if err != nil {
		return nil, err
	}
	w.ctx.Ledger.SetCode(contractAddr, wasmByte)

	w.ctx.Ledger.SetNonce(w.ctx.Caller, contractNonce+1)

	return contractAddr.Bytes(), nil
}

func createAddress(b types.Address, nonce uint64) types.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	hashBytes := sha256.Sum256(data)

	return types.Bytes2Address(hashBytes[12:])
}
