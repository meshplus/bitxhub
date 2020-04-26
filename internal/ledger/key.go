package ledger

import "fmt"

const (
	blockKey           = "block-"
	blockHashKey       = "block-hash-"
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
