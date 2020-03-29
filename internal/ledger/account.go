package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/tendermint/iavl"
	db "github.com/tendermint/tm-db"
)

type Account struct {
	Nonce     uint64     `json:"nonce"`
	Balance   uint64     `json:"balance"`
	Version   int64      `json:"version"`
	StateRoot types.Hash `json:"merkle_root"`
	CodeHash  []byte     `json:"code_hash"`

	tree *iavl.MutableTree
	code []byte
}

func newAccount(l *ChainLedger, ldb db.DB, addr types.Address) *Account {
	account := &Account{}
	_, data := l.tree.Get(compositeKey(accountKey, addr.Hex()))
	if data != nil {
		if err := account.Unmarshal(data); err != nil {
			panic(err)
		}
	}

	p := []byte(fmt.Sprintf("account-%s", addr.Hex()))
	tree := iavl.NewMutableTree(db.NewPrefixDB(ldb, p), defaultIAVLCacheSize)
	_, err := tree.Load()
	if err != nil {
		panic(err)
	}
	account.tree = tree

	return account
}

// GetState Get state from the tree
func (o *Account) GetState(key []byte) (bool, []byte) {
	_, v := o.tree.Get(key)

	return v != nil, v
}

// SetState Set account state
func (o *Account) SetState(key []byte, value []byte) {
	o.tree.Set(key, value)
}

// SetCodeAndHash Set the contract code and hash
func (o *Account) SetCodeAndHash(code []byte) {
	ret := sha256.Sum256(code)
	o.CodeHash = ret[:]
	o.SetState(o.CodeHash, code)
	o.code = code
}

// Code return the contract code
func (o *Account) Code() []byte {
	if o.code != nil {
		return o.code
	}

	if bytes.Equal(o.CodeHash, nil) {
		return nil
	}

	ok, code := o.GetState(o.CodeHash)
	if !ok {
		return nil
	}
	o.code = code

	return code
}

// SetNonce Set the nonce which indicates the contract number
func (o *Account) SetNonce(nonce uint64) {
	o.Nonce = nonce
}

// GetNonce Get the nonce from user account
func (o *Account) GetNonce() uint64 {
	return o.Nonce
}

// SetStateRoot Set the state root hash
func (o *Account) SetStateRoot(hash types.Hash) {
	o.StateRoot = hash
}

// Query Query the value using key
func (o *Account) Query(prefix string) (bool, [][]byte) {
	var ret [][]byte
	begin, end := bytesPrefix([]byte(prefix))
	o.tree.IterateRange(begin, end, false, func(key []byte, value []byte) bool {
		ret = append(ret, value)
		return false
	})

	return len(ret) != 0, ret
}

// Commit Commit the result
func (o *Account) Commit() (types.Hash, error) {
	hash, ver, err := o.tree.SaveVersion()
	if err != nil {
		return types.Hash{}, err
	}

	o.Version = ver

	return types.Bytes2Hash(hash), nil
}

// Marshal Marshal the account into byte
func (o *Account) Marshal() ([]byte, error) {
	obj := &Account{
		Nonce:     o.Nonce,
		Balance:   o.Balance,
		CodeHash:  o.CodeHash,
		Version:   o.Version,
		StateRoot: o.StateRoot,
	}

	return json.Marshal(obj)
}

// Unmarshal Unmarshal the account byte into structure
func (o *Account) Unmarshal(data []byte) error {
	return json.Unmarshal(data, o)
}

func bytesPrefix(prefix []byte) ([]byte, []byte) {
	var limit []byte
	for i := len(prefix) - 1; i >= 0; i-- {
		c := prefix[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, prefix)
			limit[i] = c + 1
			break
		}
	}
	return prefix, limit
}
