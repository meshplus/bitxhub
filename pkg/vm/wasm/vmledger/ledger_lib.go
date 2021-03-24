package vmledger

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
	imports.importLedger()

	return imports.imports, nil
}
