package vmledger

import (
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-core/validator/validatorlib"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/wasmerio/wasmer-go/wasmer"
)

type GasLimit struct {
	limit uint64

	sync.RWMutex
}

func (g *GasLimit) GetLimit() uint64 {
	g.Lock()
	defer g.Unlock()

	return g.limit
}

func (g *GasLimit) SetLimit(limit uint64) {
	g.Lock()
	defer g.Unlock()

	g.limit = limit
}

func usegas(env interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	gasPrice := uint64(args[0].I64())
	gasLimit := env.(*wasmlib.WasmEnv).Ctx["gaslimit"].(*validatorlib.GasLimit)

	gasL := gasLimit.GetLimit()
	if gasL < gasPrice {
		return []wasmer.Value{}, fmt.Errorf("run out of gas limit")
	}
	remain := gasL - gasPrice
	fmt.Println(remain)

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
