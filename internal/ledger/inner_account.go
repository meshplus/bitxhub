package ledger

import (
	"bytes"
	"encoding/json"
	"math/big"
)

type InnerAccount struct {
	Nonce    uint64   `json:"nonce"`
	Balance  *big.Int `json:"balance"`
	CodeHash []byte   `json:"code_hash"`
}

// Marshal Marshal the account into byte
func (o *InnerAccount) Marshal() ([]byte, error) {
	obj := &InnerAccount{
		Nonce:    o.Nonce,
		Balance:  o.Balance,
		CodeHash: o.CodeHash,
	}

	return json.Marshal(obj)
}

// Unmarshal Unmarshal the account byte into structure
func (o *InnerAccount) Unmarshal(data []byte) error {
	return json.Unmarshal(data, o)
}

func InnerAccountChanged(account0 *InnerAccount, account1 *InnerAccount) bool {
	// If account1 is nil, the account does not change whatever account0 is.
	if account1 == nil {
		return false
	}

	// If account already exists, account0 is not nil. We should compare account0 and account1 to get the result.
	if account0 != nil &&
		account0.Nonce == account1.Nonce &&
		account0.Balance.Cmp(account1.Balance) == 0 &&
		bytes.Equal(account0.CodeHash, account1.CodeHash) {
		return false
	}

	return true
}

func CopyOrNewIfEmpty(o *InnerAccount) *InnerAccount {
	if o == nil {
		return &InnerAccount{Balance: big.NewInt(0)}
	}

	return &InnerAccount{
		Nonce:    o.Nonce,
		Balance:  o.Balance,
		CodeHash: o.CodeHash,
	}
}
