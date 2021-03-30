package vmledger

// #include <stdlib.h>
//
// extern long long get_balance(void *context);
// extern void set_balance(void *context, long long value);
// extern int32_t get_state(void *context, long long key_ptr);
// extern void set_state(void *context, long long key_ptr, long long value_ptr);
// extern void add_state(void *context, long long key_ptr, long long value_ptr);
import "C"
import (
	"unsafe"

	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

//export get_balance
func get_balance(context unsafe.Pointer) int64 {
	ctx := wasmer.IntoInstanceContext(context)
	ctxMap := ctx.Data().(map[string]interface{})
	account := ctxMap["account"].(*ledger.Account)
	return int64(account.GetBalance())
}

//export set_balance
func set_balance(context unsafe.Pointer, value int64) {
	ctx := wasmer.IntoInstanceContext(context)
	ctxMap := ctx.Data().(map[string]interface{})
	account := ctxMap["account"].(*ledger.Account)
	account.SetBalance(uint64(value))
}

//export get_state
func get_state(context unsafe.Pointer, key_ptr int64) int32 {
	ctx := wasmer.IntoInstanceContext(context)
	ctxMap := ctx.Data().(map[string]interface{})
	data := ctxMap["argmap"].(map[int]int)
	alloc := ctxMap["allocate"].(func(...interface{}) (wasmer.Value, error))
	account := ctxMap["account"].(*ledger.Account)
	memory := ctx.Memory()
	key := memory.Data()[key_ptr : key_ptr+int64(data[int(key_ptr)])]
	ok, value := account.GetState(key)
	if !ok {
		return -1
	}
	lengthOfBytes := len(value)

	allocResult, err := alloc(lengthOfBytes)
	if err != nil {
		return -1
	}
	inputPointer := allocResult.ToI32()
	mem := memory.Data()[inputPointer:]

	var i int
	for i = 0; i < lengthOfBytes; i++ {
		mem[i] = value[i]
	}

	mem[i] = 0
	data[int(inputPointer)] = len(value)

	return inputPointer
}

//export set_state
func set_state(context unsafe.Pointer, key_ptr int64, value_ptr int64) {
	ctx := wasmer.IntoInstanceContext(context)
	ctxMap := ctx.Data().(map[string]interface{})
	data := ctxMap["argmap"].(map[int]int)
	account := ctxMap["account"].(*ledger.Account)
	memory := ctx.Memory()
	key := memory.Data()[key_ptr : key_ptr+int64(data[int(key_ptr)])]
	value := memory.Data()[value_ptr : value_ptr+int64(data[int(value_ptr)])]
	account.SetState(key, value)
}

//export add_state
func add_state(context unsafe.Pointer, key_ptr int64, value_ptr int64) {
	ctx := wasmer.IntoInstanceContext(context)
	ctxMap := ctx.Data().(map[string]interface{})
	data := ctxMap["argmap"].(map[int]int)
	account := ctxMap["account"].(*ledger.Account)
	memory := ctx.Memory()
	key := memory.Data()[key_ptr : key_ptr+int64(data[int(key_ptr)])]
	value := memory.Data()[value_ptr : value_ptr+int64(data[int(value_ptr)])]
	account.AddState(key, value)
}

func (im *Imports) importLedger() error {
	var err error
	im.imports, err = im.imports.Append("get_balance", get_balance, C.get_balance)
	if err != nil {
		return err
	}
	im.imports, err = im.imports.Append("set_balance", set_balance, C.set_balance)
	if err != nil {
		return err
	}
	im.imports, err = im.imports.Append("get_state", get_state, C.get_state)
	if err != nil {
		return err
	}
	im.imports, err = im.imports.Append("set_state", set_state, C.set_state)
	if err != nil {
		return err
	}
	im.imports, err = im.imports.Append("add_state", add_state, C.add_state)
	if err != nil {
		return err
	}

	return nil
}
