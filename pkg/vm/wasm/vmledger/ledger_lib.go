package vmledger

import (
	"github.com/meshplus/bitxhub-core/validator/validatorlib"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
	"github.com/wasmerio/wasmer-go/wasmer"
)

type Imports struct {
	imports wasmlib.WasmImport
}

func New() wasmlib.WasmImport {
	imports := &Imports{
		imports: validatorlib.New(),
	}
	return imports
}

func (imports *Imports) ImportLib(wasmEnv *wasmlib.WasmEnv) {
	imports.imports.ImportLib(wasmEnv)
	imports.importLedgerLib(wasmEnv.Store, wasmEnv)
	imports.importLedgerContants(wasmEnv.Store, wasmEnv)
}

func (imports *Imports) GetImportObject() *wasmer.ImportObject {
	return imports.imports.GetImportObject()
}
