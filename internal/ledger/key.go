package ledger

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
)

const (
	blockKey           = "block-"
	blockHashKey       = "block-hash-"
	blockTxSetKey      = "block-tx-set-"
	interchainMetaKey  = "interchain-meta-"
	receiptKey         = "receipt-"
	transactionKey     = "tx-"
	transactionMetaKey = "tx-meta-"
	chainMetaKey       = "chain-meta"
	accountKey         = "account-"
	codeKey            = "code-"
	journalKey         = "journal-"
)

func compositeKey(prefix string, value interface{}) []byte {
	return append([]byte(prefix), []byte(fmt.Sprintf("%v", value))...)
}

func composeStateKey(addr *types.Address, key []byte) []byte {
	return append(addr.Bytes(), key...)
}
