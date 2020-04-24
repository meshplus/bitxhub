package ledger

import "fmt"

const (
	ledgerTreePrefix   = "ChainLedger"
	blockKey           = "block-"
	blockHashKey       = "block-hash-"
	receiptKey         = "receipt-"
	transactionKey     = "tx-"
	transactionMetaKey = "tx-meta-"
	chainMetaKey       = "chain-meta"
	accountKey         = "account-"
	codeKey            = "code-"
)

func compositeKey(prefix string, value interface{}) []byte {
	return append([]byte(prefix), []byte(fmt.Sprintf("%v", value))...)
}
