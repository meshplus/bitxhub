package wasm

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/vm"
)

var _ vm.VM = (*WasmVM)(nil)

// Wasm represents the wasm vm in BitXHub
type WasmVM struct {
	// contract context
	ctx *vm.Context
}

// Contract represents the smart contract structure used in the wasm vm
type Contract struct {
	// contract byte
	Code []byte

	// contract hash
	Hash types.Hash
}

func New(ctx *vm.Context) (*WasmVM, error) {
	wasmVM := &WasmVM{
		ctx: ctx,
	}
	return wasmVM, nil
}

func (w *WasmVM) Run(input []byte) (ret []byte, err error) {
	if w.ctx.Callee == nil || bytes.Equal(w.ctx.Callee.Bytes(), (&types.Address{}).Bytes()) {
		return w.deploy()
	}

	return []byte(""), nil
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
