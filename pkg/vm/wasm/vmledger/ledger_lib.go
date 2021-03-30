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
	err := imports.importLedger()
	if err != nil {
		return nil, err
	}

	return imports.imports, nil
}
