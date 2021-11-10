package vmledger

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/meshplus/bitxhub-core/wasm"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	ledger1 "github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/eth-kit/ledger"
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

func addEvent(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	value_ptr := args[0].I64()
	value_len := args[1].I64()
	ctx := env.(*wasmlib.WasmEnv).Ctx
	txHash := ctx["txHash"].(string)
	mem, err := env.(*wasmlib.WasmEnv).Instance.Exports.GetMemory("memory")
	if err != nil {
		return []wasmer.Value{}, err
	}
	ledger := ctx[wasm.LEDGER].(*ledger1.Ledger)
	value := mem.Data()[value_ptr : value_ptr+value_len]
	event := &pb.Event{
		TxHash:    types.NewHashByStr(txHash),
		Data:      value,
		EventType: pb.Event_WASM,
	}
	ledger.AddEvent(event)
	return []wasmer.Value{}, nil
}

func getCurrentHeight(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	ctx := env.(*wasmlib.WasmEnv).Ctx
	currentHeight := ctx["currentHeight"].(uint64)
	strInt64 := strconv.FormatUint(currentHeight, 10)
	id16, _ := strconv.Atoi(strInt64)
	return []wasmer.Value{wasmer.NewI32(id16)}, nil
}

func getTxHash(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	ctx := env.(*wasmlib.WasmEnv).Ctx
	instance := env.(*wasmlib.WasmEnv).Instance
	txHash := ctx["txHash"].(string)

	return setString(instance, txHash)
}

func getCaller(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	ctx := env.(*wasmlib.WasmEnv).Ctx
	instance := env.(*wasmlib.WasmEnv).Instance
	caller := ctx["caller"].(string)

	return setString(instance, caller)
}

func getCurrentCaller(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	ctx := env.(*wasmlib.WasmEnv).Ctx
	instance := env.(*wasmlib.WasmEnv).Instance
	currentCaller := ctx["currentCaller"].(string)

	return setString(instance, currentCaller)
}

func setString(instance *wasmer.Instance, str string) ([]wasmer.Value, error) {
	alloc, err := instance.Exports.GetFunction("allocate")
	if err != nil {
		return []wasmer.Value{wasmer.NewI64(0)}, err
	}
	if alloc == nil {
		return []wasmer.Value{wasmer.NewI64(0)}, fmt.Errorf("not found allocate method")
	}

	allocResult, err := alloc(len(str))
	if err != nil {
		return []wasmer.Value{wasmer.NewI64(0)}, err
	}
	inputPointer := allocResult.(int32)

	store, _ := instance.Exports.GetMemory("memory")
	memory := store.Data()[inputPointer:]

	var i int
	for i = 0; i < len(str); i++ {
		memory[i] = str[i]
	}

	memory[i] = 0

	return []wasmer.Value{wasmer.NewI32(inputPointer)}, nil
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
	addEventFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I64, wasmer.I64),
			wasmer.NewValueTypes(),
		),
		wasmEnv,
		addEvent,
	)
	im.imports.GetImportObject().Register(
		"env",
		map[string]wasmer.IntoExtern{
			"get_balance": getBalanceFunc,
			"set_balance": setBalanceFunc,
			"get_state":   getStateFunc,
			"set_state":   setStateFunc,
			"add_state":   addStateFunc,
			"add_event":   addEventFunc,
		},
	)
}

func (im *Imports) importLedgerContants(store *wasmer.Store, wasmEnv *wasmlib.WasmEnv) {
	getCurrentHeightFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(),
			wasmer.NewValueTypes(wasmer.I32),
		),
		wasmEnv,
		getCurrentHeight,
	)
	getTxHashFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(),
			wasmer.NewValueTypes(wasmer.I32),
		),
		wasmEnv,
		getTxHash,
	)
	getCallerFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(),
			wasmer.NewValueTypes(wasmer.I32),
		),
		wasmEnv,
		getCaller,
	)
	getCurrentCallerFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(),
			wasmer.NewValueTypes(wasmer.I32),
		),
		wasmEnv,
		getCurrentCaller,
	)
	im.imports.GetImportObject().Register(
		"env",
		map[string]wasmer.IntoExtern{
			"get_height":         getCurrentHeightFunc,
			"get_tx_hash":        getTxHashFunc,
			"get_caller":         getCallerFunc,
			"get_current_caller": getCurrentCallerFunc,
		},
	)
}
