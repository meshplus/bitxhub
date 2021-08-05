package vmledger

import (
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/wasmerio/wasmer-go/wasmer"
)

type GasLimit struct {
	limit int64

	sync.RWMutex
}

func (g *GasLimit) GetLimit() int64 {
	g.Lock()
	defer g.Unlock()

	return g.limit
}

func (g *GasLimit) SetLimit(limit int64) {
	g.Lock()
	defer g.Unlock()

	g.limit = limit
}

func usegas(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	gasPrice := args[0].I64()
	gasLimit := env.(*wasmlib.WasmEnv).Ctx["gaslimit"].(*GasLimit)

	remain := gasLimit.GetLimit() - gasPrice
	fmt.Println(remain)
	if remain < 0 {
		return []wasmer.Value{}, fmt.Errorf("run out of gas limit")
	}
	gasLimit.SetLimit(remain)
	return []wasmer.Value{}, nil
}

func (im *Imports) importGasLib(store *wasmer.Store, wasmEnv *wasmlib.WasmEnv) {
	useGasFunc := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I64),
			wasmer.NewValueTypes(),
		),
		wasmEnv,
		usegas,
	)
	im.imports.Register(
		"metering",
		map[string]wasmer.IntoExtern{
			"usegas": useGasFunc,
		},
	)
}
