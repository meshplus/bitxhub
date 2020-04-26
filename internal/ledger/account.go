package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"sort"

	"github.com/meshplus/bitxhub-kit/types"
)

type Account struct {
	Addr           types.Address
	originAccount  *innerAccount
	dirtyAccount   *innerAccount
	originState    map[string][]byte
	dirtyState     map[string][]byte
	originCode     []byte
	dirtyCode      []byte
	dirtyStateHash types.Hash
	ldb            storage.Storage
}

type innerAccount struct {
	Nonce    uint64 `json:"nonce"`
	Balance  uint64 `json:"balance"`
	CodeHash []byte `json:"code_hash"`
}

func newAccount(ldb storage.Storage, addr types.Address) *Account {
	return &Account{
		Addr:        addr,
		originState: make(map[string][]byte),
		dirtyState:  make(map[string][]byte),
		ldb:         ldb,
	}
}

// GetState Get state from local cache, if not found, then get it from DB
func (o *Account) GetState(key []byte) (bool, []byte) {
	hexKey := hex.EncodeToString(key)

	if val, exist := o.dirtyState[hexKey]; exist {
		return val != nil, val
	}

	if val, exist := o.originState[hexKey]; exist {
		return val != nil, val
	}

	val, err := o.ldb.Get(append(o.Addr.Bytes(), key...))
	if err != nil && err != errors.ErrNotFound {
		panic(err)
	}

	o.originState[hexKey] = val

	return val != nil, val
}

// SetState Set account state
func (o *Account) SetState(key []byte, value []byte) {
	o.dirtyState[hex.EncodeToString(key)] = value
}

// SetCodeAndHash Set the contract code and hash
func (o *Account) SetCodeAndHash(code []byte) {
	ret := sha256.Sum256(code)
	if o.dirtyAccount == nil {
		o.dirtyAccount = copyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.CodeHash = ret[:]
	o.dirtyCode = code
}

// Code return the contract code
func (o *Account) Code() []byte {
	if o.dirtyCode != nil {
		return o.dirtyCode
	}

	if o.originCode != nil {
		return o.originCode
	}

	if bytes.Equal(o.CodeHash(), nil) {
		return nil
	}

	code, err := o.ldb.Get(compositeKey(codeKey, o.Addr.Hex()))
	if err != nil && err != errors.ErrNotFound {
		panic(err)
	}

	o.originCode = code
	o.dirtyCode = code

	return code
}

func (o *Account) CodeHash() []byte {
	if o.dirtyAccount != nil {
		return o.dirtyAccount.CodeHash
	}
	if o.originAccount != nil {
		return o.originAccount.CodeHash
	}
	return nil
}

// SetNonce Set the nonce which indicates the contract number
func (o *Account) SetNonce(nonce uint64) {
	if o.dirtyAccount == nil {
		o.dirtyAccount = copyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Nonce = nonce
}

// GetNonce Get the nonce from user account
func (o *Account) GetNonce() uint64 {
	if o.dirtyAccount != nil {
		return o.dirtyAccount.Nonce
	}
	if o.originAccount != nil {
		return o.originAccount.Nonce
	}
	return 0
}

// GetBalance Get the balance from the account
func (o *Account) GetBalance() uint64 {
	if o.dirtyAccount != nil {
		return o.dirtyAccount.Balance
	}
	if o.originAccount != nil {
		return o.originAccount.Balance
	}
	return 0
}

// SetBalance Set the balance to the account
func (o *Account) SetBalance(balance uint64) {
	if o.dirtyAccount == nil {
		o.dirtyAccount = copyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Balance = balance
}

// Query Query the value using key
func (o *Account) Query(prefix string) (bool, [][]byte) {
	var ret [][]byte
	begin, end := bytesPrefix(append(o.Addr.Bytes(), prefix...))
	it := o.ldb.Iterator(begin, end)

	for it.Next() {
		val := make([]byte, len(it.Value()))
		copy(val, it.Value())
		ret = append(ret, val)
	}

	return len(ret) != 0, ret
}

func (o *Account) getJournalIfModified(ldbBatch storage.Batch) []journalEntry {
	var entries []journalEntry

	if !innerAccountChanged(o.originAccount, o.dirtyAccount) {
		data, err := o.dirtyAccount.Marshal()
		if err != nil {
			panic(err)
		}
		ldbBatch.Put(compositeKey(accountKey, o.Addr.Hex()), data)
		entries = append(entries, accountChange{address: o.Addr, prevAccount: o.originAccount})
	}

	if bytes.Compare(o.originCode, o.dirtyCode) != 0 {
		if o.dirtyCode != nil {
			ldbBatch.Put(compositeKey(codeKey, o.Addr.Hex()), o.dirtyCode)
		} else {
			ldbBatch.Delete(compositeKey(codeKey, o.Addr.Hex()))
		}
		entries = append(entries, codeChange{address: o.Addr, prevCode: o.originCode})
	}

	stateJournal := o.getStateJournalAndComputeHash(ldbBatch)
	if len(stateJournal.prevStates) != 0 {
		entries = append(entries, stateJournal)
	}

	return entries
}

func (o *Account) getStateJournalAndComputeHash(ldbBatch storage.Batch) stateChange {
	stateJournal := stateChange{
		address:    o.Addr,
		prevStates: make(map[string][]byte),
	}
	var dirtyStateKeys []string
	var dirtyStateData []byte

	for key, val := range o.dirtyState {
		origVal := o.originState[key]
		if bytes.Compare(origVal, val) != 0 {
			dirtyStateKeys = append(dirtyStateKeys, key)
			byteKey, err := hex.DecodeString(key)
			if err != nil {
				panic(err)
			}

			if val != nil {
				ldbBatch.Put(append(o.Addr.Bytes(), byteKey...), val)
			} else {
				ldbBatch.Delete(append(o.Addr.Bytes(), byteKey...))
			}
			stateJournal.prevStates[key] = origVal
		}
	}

	sort.Strings(dirtyStateKeys)

	for _, key := range dirtyStateKeys {
		dirtyStateData = append(dirtyStateData, key...)
		dirtyStateData = append(dirtyStateData, o.dirtyState[key]...)
	}
	o.dirtyStateHash = sha256.Sum256(dirtyStateData)

	return stateJournal
}

func (o *Account) getDirtyData() []byte {
	var dirtyData []byte

	dirtyData = append(dirtyData, o.Addr.Bytes()...)

	if o.dirtyAccount != nil {
		data, err := o.dirtyAccount.Marshal()
		if err != nil {
			panic(err)
		}
		dirtyData = append(dirtyData, data...)
	}

	return append(dirtyData, o.dirtyStateHash[:]...)
}

func innerAccountChanged(account0 *innerAccount, account1 *innerAccount) bool {
	if account1 == nil ||
		account0 != nil &&
			account0.Nonce == account1.Nonce &&
			account0.Balance == account1.Balance &&
			bytes.Compare(account0.CodeHash, account1.CodeHash) == 0 {
		return true
	}

	return false
}

// Marshal Marshal the account into byte
func (o *innerAccount) Marshal() ([]byte, error) {
	obj := &innerAccount{
		Nonce:    o.Nonce,
		Balance:  o.Balance,
		CodeHash: o.CodeHash,
	}

	return json.Marshal(obj)
}

// Unmarshal Unmarshal the account byte into structure
func (o *innerAccount) Unmarshal(data []byte) error {
	return json.Unmarshal(data, o)
}

func copyOrNewIfEmpty(o *innerAccount) *innerAccount {
	if o == nil {
		return &innerAccount{}
	}

	return &innerAccount{
		Nonce:    o.Nonce,
		Balance:  o.Balance,
		CodeHash: o.CodeHash,
	}
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
