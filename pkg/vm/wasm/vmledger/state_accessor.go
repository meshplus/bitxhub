package vmledger

import (
	"fmt"
	"math/big"

	"github.com/meshplus/eth-kit/ledger"

	"github.com/meshplus/bitxhub-core/wasm"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/wasmerio/wasmer-go/wasmer"
)

func getBalance(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	account := env.(*wasmlib.WasmEnv).Ctx[wasm.ACCOUNT].(ledger.IAccount)
	return []wasmer.Value{wasmer.NewI64(account.GetBalance().Int64())}, nil
}

func setBalance(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	account := env.(*wasmlib.WasmEnv).Ctx[wasm.ACCOUNT].(ledger.IAccount)
	account.SetBalance(new(big.Int).SetUint64(uint64(args[0].I64())))
	return []wasmer.Value{}, nil
}

func getState(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	key_ptr := args[0].I64()
	ctx := env.(*wasmlib.WasmEnv).Ctx
	mem, err := env.(*wasmlib.WasmEnv).Instance.Exports.GetMemory("memory")
	if err != nil {
		return []wasmer.Value{wasmer.NewI32(-1)}, nil
	}
	data := ctx[wasm.CONTEXT_ARGMAP].(map[int]int)
	// alloc := ctx["allocate"].(wasmer.NativeFunction)
	alloc, err := env.(*wasmlib.WasmEnv).Instance.Exports.GetFunction("allocate")
	if err != nil {
		return []wasmer.Value{wasmer.NewI32(-1)}, nil
	}
	account := ctx[wasm.ACCOUNT].(ledger.IAccount)
	key := mem.Data()[key_ptr : key_ptr+int64(data[int(key_ptr)])]
	ok, value := account.GetState(key)
	if !ok {
		fmt.Println("===========================================")
		return []wasmer.Value{wasmer.NewI32(-1)}, nil
	}
	lengthOfBytes := len(value)

	allocResult, err := alloc(lengthOfBytes)
	if err != nil {
		return []wasmer.Value{}, err
	}
	inputPointer := allocResult.(int32)
	memory := mem.Data()[inputPointer:]

	var i int
	for i = 0; i < lengthOfBytes; i++ {
		memory[i] = value[i]
	}

	memory[i] = 0
	data[int(inputPointer)] = len(value)

	return []wasmer.Value{wasmer.NewI32(inputPointer)}, nil
}

func setState(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	key_ptr := args[0].I64()
	value_ptr := args[1].I64()
	ctx := env.(*wasmlib.WasmEnv).Ctx
	mem, err := env.(*wasmlib.WasmEnv).Instance.Exports.GetMemory("memory")
	if err != nil {
		return []wasmer.Value{}, err
	}
	data := ctx[wasm.CONTEXT_ARGMAP].(map[int]int)
	account := ctx[wasm.ACCOUNT].(ledger.IAccount)
	key := mem.Data()[key_ptr : key_ptr+int64(data[int(key_ptr)])]
	value := mem.Data()[value_ptr : value_ptr+int64(data[int(value_ptr)])]
	account.SetState(key, value)
	return []wasmer.Value{}, nil
}

func addState(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	key_ptr := args[0].I64()
	value_ptr := args[1].I64()
	ctx := env.(*wasmlib.WasmEnv).Ctx
	mem, err := env.(*wasmlib.WasmEnv).Instance.Exports.GetMemory("memory")
	if err != nil {
		return []wasmer.Value{}, err
	}
	data := ctx[wasm.CONTEXT_ARGMAP].(map[int]int)
	account := ctx[wasm.ACCOUNT].(ledger.IAccount)
	key := mem.Data()[key_ptr : key_ptr+int64(data[int(key_ptr)])]
	value := mem.Data()[value_ptr : value_ptr+int64(data[int(value_ptr)])]
	account.AddState(key, value)
	return []wasmer.Value{}, nil
}

func (im *Imports) importLedgerLib(store *wasmer.Store, wasmEnv *wasmlib.WasmEnv) {
	getBalanceFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(),
			wasmer.NewValueTypes(wasmer.I64),
		),
		wasmEnv,
		getBalance,
	)
	setBalanceFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I64),
			wasmer.NewValueTypes(),
		),
		wasmEnv,
		setBalance,
	)
	getStateFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I64),
			wasmer.NewValueTypes(wasmer.I32),
		),
		wasmEnv,
		getState,
	)
	setStateFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I64, wasmer.I64),
			wasmer.NewValueTypes(),
		),
		wasmEnv,
		setState,
	)
	addStateFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I64, wasmer.I64),
			wasmer.NewValueTypes(),
		),
		wasmEnv,
		addState,
	)
	im.imports.GetImportObject().Register(
		"env",
		map[string]wasmer.IntoExtern{
			"get_balance": getBalanceFunc,
		},
	)
	im.imports.GetImportObject().Register(
		"env",
		map[string]wasmer.IntoExtern{
			"set_balance": setBalanceFunc,
		},
	)
	im.imports.GetImportObject().Register(
		"env",
		map[string]wasmer.IntoExtern{
			"get_state": getStateFunc,
		},
	)
	im.imports.GetImportObject().Register(
		"env",
		map[string]wasmer.IntoExtern{
			"set_state": setStateFunc,
		},
	)
	im.imports.GetImportObject().Register(
		"env",
		map[string]wasmer.IntoExtern{
			"add_state": addStateFunc,
		},
	)
}
