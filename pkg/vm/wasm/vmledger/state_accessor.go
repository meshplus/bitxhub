package vmledger

import (
	"fmt"
	"math/big"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/meshplus/bitxhub-core/wasm"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	ledger1 "github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/eth-kit/ledger"
)

func getBalance(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "get_balance",
		Func: func() int64 {
			account := context[ACCOUNT].(ledger.IAccount)
			return account.GetBalance().Int64()
		},
	}
}

func setBalance(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "set_balance",
		Func: func(balance int64) {
			account := context[ACCOUNT].(ledger.IAccount)
			account.SetBalance(new(big.Int).SetUint64(uint64(balance)))
		},
	}
}

func getState(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "get_state",
		Func: func(caller *wasmtime.Caller, key_ptr int32) int32 {
			mem := caller.GetExport("memory").Memory().UnsafeData(store)
			account := context[ACCOUNT].(ledger.IAccount)
			argmap := context[wasm.CONTEXT_ARGMAP].(map[int32]int32)
			key := mem[key_ptr : key_ptr+argmap[key_ptr]]
			ok, value := account.GetState(key)
			if !ok {
				context[wasm.ERROR] = fmt.Errorf("no state")
				return -1
			}
			inputPointer, err := setBytes(caller, store, value)
			if err != nil {
				context[wasm.ERROR] = err
				return -1
			}
			return inputPointer
		},
	}
}

func setState(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "set_state",
		Func: func(caller *wasmtime.Caller, key_ptr int32, value_ptr int32) {
			mem := caller.GetExport("memory").Memory().UnsafeData(store)
			account := context[ACCOUNT].(ledger.IAccount)
			argmap := context[wasm.CONTEXT_ARGMAP].(map[int32]int32)
			key := mem[key_ptr : key_ptr+argmap[key_ptr]]
			value := mem[value_ptr : value_ptr+argmap[value_ptr]]
			account.SetState(key, value)
		},
	}
}

func addState(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "add_state",
		Func: func(caller *wasmtime.Caller, key_ptr int32, value_ptr int32) {
			mem := caller.GetExport("memory").Memory().UnsafeData(store)
			account := context[ACCOUNT].(ledger.IAccount)
			argmap := context[wasm.CONTEXT_ARGMAP].(map[int32]int32)
			key := mem[key_ptr : key_ptr+argmap[key_ptr]]
			value := mem[value_ptr : value_ptr+argmap[value_ptr]]
			account.AddState(key, value)
		},
	}
}

func addEvent(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "add_event",
		Func: func(caller *wasmtime.Caller, value_ptr int32) {
			mem := caller.GetExport("memory").Memory().UnsafeData(store)
			ledger := context[LEDGER].(*ledger1.Ledger)
			txHash := context[TX_HASH].(string)
			argmap := context[wasm.CONTEXT_ARGMAP].(map[int32]int32)
			value := mem[value_ptr : value_ptr+argmap[value_ptr]]
			event := &pb.Event{
				TxHash:    types.NewHashByStr(txHash),
				Data:      value,
				EventType: pb.Event_WASM,
			}
			ledger.AddEvent(event)
		},
	}
}

func getCurrentHeight(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "get_current_height",
		Func: func(caller *wasmtime.Caller) int64 {
			return int64(context[CURRENT_HEIGHT].(uint64))
		},
	}
}

func getTxHash(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "get_tx_hash",
		Func: func(caller *wasmtime.Caller) int32 {
			txHash := context[TX_HASH].(string)
			inputPointer, err := setString(caller, store, txHash)
			if err != nil {
				context[wasm.ERROR] = err
				return -1
			}
			return inputPointer
		},
	}
}

func getCaller(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "get_caller",
		Func: func(caller *wasmtime.Caller) int32 {
			contractCaller := context[CALLER].(string)
			inputPointer, err := setString(caller, store, contractCaller)
			if err != nil {
				context[wasm.ERROR] = err
				return -1
			}
			return inputPointer
		},
	}
}

func getCurrentCaller(context map[string]interface{}, store *wasmtime.Store) *wasmlib.ImportLib {
	return &wasmlib.ImportLib{
		Module: "env",
		Name:   "get_current_caller",
		Func: func(caller *wasmtime.Caller) int32 {
			currentCaller := context[CURRENT_CALLER].(string)
			inputPointer, err := setString(caller, store, currentCaller)
			if err != nil {
				context[wasm.ERROR] = err
				return -1
			}
			return inputPointer
		},
	}
}

func setString(caller *wasmtime.Caller, store *wasmtime.Store, str string) (int32, error) {
	alloc := caller.GetExport("allocate").Func()
	if alloc == nil {
		return -1, fmt.Errorf("not found allocate method")
	}
	lengthOfStr := len(str)
	allocResult, err := alloc.Call(store, lengthOfStr+1)
	if err != nil {
		return -1, err
	}
	inputPointer := allocResult.(int32)
	mem := caller.GetExport("memory").Memory().UnsafeData(store)
	memory := mem[inputPointer:]

	var i int
	for i = 0; i < lengthOfStr; i++ {
		memory[i] = str[i]
	}

	memory[i] = 0
	fmt.Println(inputPointer)

	return inputPointer, nil
}

func setBytes(caller *wasmtime.Caller, store *wasmtime.Store, str []byte) (int32, error) {
	alloc := caller.GetExport("allocate").Func()
	if alloc == nil {
		return -1, fmt.Errorf("not found allocate method")
	}
	lengthOfStr := len(str)
	allocResult, err := alloc.Call(store, lengthOfStr+1)
	if err != nil {
		return -1, err
	}
	inputPointer := allocResult.(int32)
	mem := caller.GetExport("memory").Memory().UnsafeData(store)
	memory := mem[inputPointer:]

	var i int
	for i = 0; i < lengthOfStr; i++ {
		memory[i] = str[i]
	}

	memory[i] = 0

	return inputPointer, nil
}

func ImportLedgerLib(context map[string]interface{}, store *wasmtime.Store) []*wasmlib.ImportLib {
	var libs []*wasmlib.ImportLib
	libs = append(libs, getBalance(context, store))
	libs = append(libs, setBalance(context, store))
	libs = append(libs, getState(context, store))
	libs = append(libs, setState(context, store))
	libs = append(libs, addState(context, store))
	libs = append(libs, addEvent(context, store))
	libs = append(libs, getCurrentHeight(context, store))
	libs = append(libs, getTxHash(context, store))
	libs = append(libs, getCaller(context, store))
	libs = append(libs, getCurrentCaller(context, store))

	return libs
}
