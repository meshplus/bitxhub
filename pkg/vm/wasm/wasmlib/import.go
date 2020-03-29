package wasmlib

import (
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

type Imports struct {
	imports *wasmer.Imports
}

func New() (*wasmer.Imports, error) {
	imports := &Imports{
		imports: wasmer.NewImports(),
	}
	imports.importECDSA()
	imports.importFabricV13()

	return imports.imports, nil
}
