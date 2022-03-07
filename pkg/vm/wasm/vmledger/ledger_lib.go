package vmledger

import (
	"github.com/bytecodealliance/wasmtime-go"
	"github.com/meshplus/bitxhub-core/wasm/wasmlib"
)

const (
	ACCOUNT        = "account"
	LEDGER         = "ledger"
	ALLOC_MEM      = "allocate"
	TX_HASH        = "tx_hash"
	CURRENT_HEIGHT = "current_height"
	CALLER         = "caller"
	CURRENT_CALLER = "current_caller"
)

func NewLedgerWasmLibs(context map[string]interface{}, store *wasmtime.Store) []*wasmlib.ImportLib {
	return ImportLedgerLib(context, store)
}
