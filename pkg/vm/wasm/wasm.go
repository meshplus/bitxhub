package wasm

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/meshplus/bitxhub-core/wasm"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/vm"
	"github.com/meshplus/bitxhub/pkg/vm/wasm/vmledger"
	"github.com/sirupsen/logrus"
)

const GasXVMDeploy = 21000 * 10

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

func NewStore() *wasmtime.Store {
	return wasm.NewStore()
}

// New creates a wasm vm instance
func New(ctx *vm.Context, libs []*wasmlib.ImportLib, context map[string]interface{}, store *wasmtime.Store) (*WasmVM, error) {
	wasmVM := &WasmVM{
		ctx: ctx,
	}

	if ctx.Callee == nil || bytes.Equal(ctx.Callee.Bytes(), (&types.Address{}).Bytes()) {
		return wasmVM, nil
	}

	contractByte := ctx.Ledger.GetCode(ctx.Callee)
	if contractByte == nil {
		return nil, fmt.Errorf("this rule address %s does not exist", ctx.Callee)
	}
	contract := &wasm.Contract{}
	if err := json.Unmarshal(contractByte, contract); err != nil {
		return nil, fmt.Errorf("contract byte not correct")
	}
	w, err := wasm.NewWithStore(contract.Code, context, libs, store)
	if err != nil {
		return nil, fmt.Errorf("init wasm failed: %w", err)
	}

	w.SetContext(vmledger.LEDGER, ctx.Ledger)
	w.SetContext(vmledger.ACCOUNT, ctx.Ledger.GetOrCreateAccount(ctx.Callee))
	w.SetContext(vmledger.CURRENT_HEIGHT, ctx.CurrentHeight)
	w.SetContext(vmledger.TX_HASH, ctx.Tx.GetHash().String())
	w.SetContext(vmledger.CALLER, ctx.Caller.String())
	w.SetContext(vmledger.CURRENT_CALLER, ctx.CurrentCaller.String())
	wasmVM.w = w

	return wasmVM, nil
}

// Run let the wasm vm excute or deploy the smart contract which depends on whether the callee is empty
func (w *WasmVM) Run(input []byte, gasLimit uint64) (ret []byte, gasUsed uint64, err error) {
	if w.ctx.Callee == nil || bytes.Equal(w.ctx.Callee.Bytes(), (&types.Address{}).Bytes()) {
		return w.deploy()
	}

	return w.w.Execute(input, gasLimit)
}

func (w *WasmVM) deploy() ([]byte, uint64, error) {
	w.ctx.Logger.WithFields(logrus.Fields{}).Info("Rule is deploying")
	if len(w.ctx.TransactionData.Payload) == 0 {
		return nil, 0, fmt.Errorf("contract cannot be empty")
	}
	context := make(map[string]interface{})
	store := wasm.NewStore()
	libs := vmledger.NewLedgerWasmLibs(context, store)
	_, err := wasm.NewWithStore(w.ctx.TransactionData.Payload, context, libs, store)
	if err != nil {
		w.ctx.Logger.WithFields(logrus.Fields{}).Error("new instance:", err)
		return nil, 0, err
	}
	contractNonce := w.ctx.Ledger.GetNonce(w.ctx.Caller)

	contractAddr := createAddress(w.ctx.Caller, contractNonce)
	wasmStruct := &Contract{
		Code: w.ctx.TransactionData.Payload,
		Hash: *types.NewHash(w.ctx.TransactionData.Payload),
	}
	wasmByte, err := json.Marshal(wasmStruct)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal wasm struct error: %w", err)
	}
	w.ctx.Ledger.SetCode(contractAddr, wasmByte)

	w.ctx.Ledger.SetNonce(w.ctx.Caller, contractNonce+1)

	return contractAddr.Bytes(), GasXVMDeploy, nil
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
