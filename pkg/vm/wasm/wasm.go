package wasm

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-core/wasm"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/vm"
	metering "github.com/meshplus/go-wasm-metering"
	"github.com/sirupsen/logrus"
	"github.com/wasmerio/wasmer-go/wasmer"
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

// New creates a wasm vm instance
func New(ctx *vm.Context, imports wasmlib.WasmImport, instances map[string]*wasmer.Instance) (*WasmVM, error) {
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
	w.SetContext("currentHeight", ctx.CurrentHeight)
	w.SetContext("txHash", ctx.Tx.GetHash().String())
	w.SetContext("caller", ctx.Caller.String())
	w.SetContext("currentCaller", ctx.CurrentCaller.String())

	// alloc, err := w.Instance.Exports.GetFunction("allocate")
	// if err != nil {
	// 	return nil, err
	// }
	// w.SetContext(wasm.ALLOC_MEM, alloc)
	wasmVM.w = w

	return wasmVM, nil
}

func EmptyImports() (wasmlib.WasmImport, error) {
	return wasm.NewEmptyImports(), nil
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

	var (
		metaChan = make(chan []byte)
		err      error
	)

	go func(err error) {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("%v", e)
			}
		}()

		meteredCode, _, err := metering.MeterWASM(w.ctx.TransactionData.Payload, &metering.Options{}, w.ctx.Logger)
		metaChan <- meteredCode
	}(err)
	meteredCode := <-metaChan
	if err != nil {
		return nil, 0, err
	}

	contractNonce := w.ctx.Ledger.GetNonce(w.ctx.Caller)

	contractAddr := createAddress(w.ctx.Caller, contractNonce)
	wasmStruct := &Contract{
		Code: meteredCode,
		Hash: *types.NewHash(w.ctx.TransactionData.Payload),
	}
	wasmByte, err := json.Marshal(wasmStruct)
	if err != nil {
		return nil, 0, err
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
