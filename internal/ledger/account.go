package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"sort"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/storage"
)

type Account struct {
	Addr           types.Address
	originAccount  *innerAccount
	dirtyAccount   *innerAccount
	originState    sync.Map
	dirtyState     sync.Map
	originCode     []byte
	dirtyCode      []byte
	dirtyStateHash types.Hash
	ldb            storage.Storage
	cache          *AccountCache
	lock           sync.RWMutex
}

type innerAccount struct {
	Nonce    uint64 `json:"nonce"`
	Balance  uint64 `json:"balance"`
	CodeHash []byte `json:"code_hash"`
}

func newAccount(ldb storage.Storage, cache *AccountCache, addr types.Address) *Account {
	return &Account{
		Addr:  addr,
		ldb:   ldb,
		cache: cache,
	}
}

// GetState Get state from local cache, if not found, then get it from DB
func (o *Account) GetState(key []byte) (bool, []byte) {
	if val, exist := o.dirtyState.Load(string(key)); exist {
		value := val.([]byte)
		return value != nil, value
	}

	if val, exist := o.originState.Load(string(key)); exist {
		value := val.([]byte)
		return value != nil, value
	}

	val, ok := o.cache.getState(o.Addr.Hex(), string(key))
	if !ok {
		val = o.ldb.Get(composeStateKey(o.Addr, key))
	}

	o.originState.Store(string(key), val)

	return val != nil, val
}

// SetState Set account state
func (o *Account) SetState(key []byte, value []byte) {
	o.GetState(key)
	o.dirtyState.Store(string(key), value)
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
	o.lock.Lock()
	defer o.lock.Unlock()

	if o.dirtyCode != nil {
		return o.dirtyCode
	}

	if o.originCode != nil {
		return o.originCode
	}

	if bytes.Equal(o.CodeHash(), nil) {
		return nil
	}

	code, ok := o.cache.getCode(o.Addr.Hex())
	if !ok {
		code = o.ldb.Get(compositeKey(codeKey, o.Addr.Hex()))
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
	stored := make(map[string][]byte)

	cached := o.cache.query(o.Addr.Hex(), prefix)

	begin, end := bytesPrefix(append(o.Addr.Bytes(), prefix...))
	it := o.ldb.Iterator(begin, end)

	for it.Next() {
		key := make([]byte, len(it.Key()))
		val := make([]byte, len(it.Value()))
		copy(key, it.Key())
		copy(val, it.Value())
		stored[string(key)] = val
	}

	for key, val := range cached {
		stored[key] = val
	}

	for _, val := range stored {
		ret = append(ret, val)
	}

	sort.Slice(ret, func(i, j int) bool {
		return bytes.Compare(ret[i], ret[j]) < 0
	})

	return len(ret) != 0, ret
}

func (o *Account) getJournalIfModified() *journal {
	entry := &journal{Address: o.Addr}

	if innerAccountChanged(o.originAccount, o.dirtyAccount) {
		entry.AccountChanged = true
		entry.PrevAccount = o.originAccount
	}

	if o.originCode == nil && !(o.originAccount == nil || o.originAccount.CodeHash == nil) {
		o.originCode = o.ldb.Get(compositeKey(codeKey, o.Addr.Hex()))
	}

	if !bytes.Equal(o.originCode, o.dirtyCode) {
		entry.CodeChanged = true
		entry.PrevCode = o.originCode
	}

	prevStates := o.getStateJournalAndComputeHash()
	if len(prevStates) != 0 {
		entry.PrevStates = prevStates
	}

	if entry.AccountChanged || entry.CodeChanged || len(entry.PrevStates) != 0 {
		return entry
	}

	return nil
}

func (o *Account) getStateJournalAndComputeHash() map[string][]byte {
	prevStates := make(map[string][]byte)
	var dirtyStateKeys []string
	var dirtyStateData []byte

	o.dirtyState.Range(func(key, value interface{}) bool {
		origVal, ok := o.originState.Load(key)
		var origValBytes []byte
		if ok {
			origValBytes = origVal.([]byte)
		}
		valBytes := value.([]byte)
		if !bytes.Equal(origValBytes, valBytes) {
			prevStates[key.(string)] = origValBytes
			dirtyStateKeys = append(dirtyStateKeys, key.(string))
		}
		return true
	})

	sort.Strings(dirtyStateKeys)

	for _, key := range dirtyStateKeys {
		dirtyStateData = append(dirtyStateData, key...)
		dirtyVal, _ := o.dirtyState.Load(key)
		dirtyStateData = append(dirtyStateData, dirtyVal.([]byte)...)
	}
	o.dirtyStateHash = sha256.Sum256(dirtyStateData)

	return prevStates
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
	// If account1 is nil, the account does not change whatever account0 is.
	if account1 == nil {
		return false
	}

	// If account already exists, account0 is not nil. We should compare account0 and account1 to get the result.
	if account0 != nil &&
		account0.Nonce == account1.Nonce &&
		account0.Balance == account1.Balance &&
		bytes.Equal(account0.CodeHash, account1.CodeHash) {
		return false
	}

	return true
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
