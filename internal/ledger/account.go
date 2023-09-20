package ledger

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/types"
)

var _ IAccount = (*SimpleAccount)(nil)

type SimpleAccount struct {
	Addr           *types.Address
	originAccount  *InnerAccount
	dirtyAccount   *InnerAccount
	originState    map[string][]byte
	dirtyState     map[string][]byte
	originCode     []byte
	dirtyCode      []byte
	dirtyStateHash *types.Hash
	ldb            storage.Storage
	cache          *AccountCache

	changer  *stateChanger
	suicided bool
}

func NewAccount(ldb storage.Storage, cache *AccountCache, addr *types.Address, changer *stateChanger) *SimpleAccount {
	return &SimpleAccount{
		Addr:        addr,
		originState: make(map[string][]byte),
		dirtyState:  make(map[string][]byte),
		ldb:         ldb,
		cache:       cache,
		changer:     changer,
		suicided:    false,
	}
}

func (o *SimpleAccount) GetAddress() *types.Address {
	return o.Addr
}

// GetState Get state from local cache, if not found, then get it from DB
func (o *SimpleAccount) GetState(key []byte) (bool, []byte) {
	if value, exist := o.dirtyState[string(key)]; exist {
		return value != nil, value
	}

	if value, exist := o.originState[string(key)]; exist {
		return value != nil, value
	}

	val, ok := o.cache.getState(o.Addr, string(key))
	if !ok {
		val = o.ldb.Get(composeStateKey(o.Addr, key))
	}

	o.originState[string(key)] = val

	return val != nil, val
}

func (o *SimpleAccount) GetCommittedState(key []byte) []byte {
	if value, exist := o.originState[string(key)]; exist {
		if value == nil {
			return (&types.Hash{}).Bytes()
		}
		return value
	}

	val, ok := o.cache.getState(o.Addr, string(key))
	if !ok {
		val = o.ldb.Get(composeStateKey(o.Addr, key))
	}

	o.originState[string(key)] = val

	if val == nil {
		return (&types.Hash{}).Bytes()
	}
	return val
}

// SetState Set account state
func (o *SimpleAccount) SetState(key []byte, value []byte) {
	_, prev := o.GetState(key)
	o.dirtyState[string(key)] = value
	o.changer.append(storageChange{
		account:  o.Addr,
		key:      key,
		prevalue: prev,
	})
}

func (o *SimpleAccount) setState(key []byte, value []byte) {
	o.dirtyState[string(key)] = value
}

// SetCodeAndHash Set the contract code and hash
func (o *SimpleAccount) SetCodeAndHash(code []byte) {
	ret := crypto.Keccak256Hash(code)
	o.changer.append(codeChange{
		account:  o.Addr,
		prevcode: o.Code(),
	})
	if o.dirtyAccount == nil {
		o.dirtyAccount = CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.CodeHash = ret.Bytes()
	o.dirtyCode = code
}

func (o *SimpleAccount) setCodeAndHash(code []byte) {
	ret := crypto.Keccak256Hash(code)
	if o.dirtyAccount == nil {
		o.dirtyAccount = CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.CodeHash = ret.Bytes()
	o.dirtyCode = code
}

// Code return the contract code
func (o *SimpleAccount) Code() []byte {
	if o.dirtyCode != nil {
		return o.dirtyCode
	}

	if o.originCode != nil {
		return o.originCode
	}

	if bytes.Equal(o.CodeHash(), nil) {
		return nil
	}

	code, ok := o.cache.getCode(o.Addr)
	if !ok {
		code = o.ldb.Get(compositeKey(codeKey, o.Addr))
	}

	o.originCode = code
	o.dirtyCode = code

	return code
}

func (o *SimpleAccount) CodeHash() []byte {
	if o.dirtyAccount != nil {
		return o.dirtyAccount.CodeHash
	}
	if o.originAccount != nil {
		return o.originAccount.CodeHash
	}
	return nil
}

// SetNonce Set the nonce which indicates the contract number
func (o *SimpleAccount) SetNonce(nonce uint64) {
	o.changer.append(nonceChange{
		account: o.Addr,
		prev:    o.GetNonce(),
	})
	if o.dirtyAccount == nil {
		o.dirtyAccount = CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Nonce = nonce
}

func (o *SimpleAccount) setNonce(nonce uint64) {
	if o.dirtyAccount == nil {
		o.dirtyAccount = CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Nonce = nonce
}

// GetNonce Get the nonce from user account
func (o *SimpleAccount) GetNonce() uint64 {
	if o.dirtyAccount != nil {
		return o.dirtyAccount.Nonce
	}
	if o.originAccount != nil {
		return o.originAccount.Nonce
	}
	return 0
}

// GetBalance Get the balance from the account
func (o *SimpleAccount) GetBalance() *big.Int {
	if o.dirtyAccount != nil {
		return o.dirtyAccount.Balance
	}
	if o.originAccount != nil {
		return o.originAccount.Balance
	}
	return new(big.Int).SetInt64(0)
}

// SetBalance Set the balance to the account
func (o *SimpleAccount) SetBalance(balance *big.Int) {
	o.changer.append(balanceChange{
		account: o.Addr,
		prev:    new(big.Int).Set(o.GetBalance()),
	})
	if o.dirtyAccount == nil {
		o.dirtyAccount = CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Balance = balance
}

func (o *SimpleAccount) setBalance(balance *big.Int) {
	if o.dirtyAccount == nil {
		o.dirtyAccount = CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Balance = balance
}

func (o *SimpleAccount) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	o.SetBalance(new(big.Int).Sub(o.GetBalance(), amount))
}

func (o *SimpleAccount) AddBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	o.SetBalance(new(big.Int).Add(o.GetBalance(), amount))
}

// Query Query the value using key
func (o *SimpleAccount) Query(prefix string) (bool, [][]byte) {
	var ret [][]byte
	stored := make(map[string][]byte)

	begin, end := bytesPrefix(append(o.Addr.Bytes(), prefix...))
	it := o.ldb.Iterator(begin, end)

	for it.Next() {
		key := make([]byte, len(it.Key()))
		val := make([]byte, len(it.Value()))
		copy(key, it.Key())
		copy(val, it.Value())
		stored[string(key)] = val
	}

	for key, value := range o.dirtyState {
		if strings.HasPrefix(key, prefix) {
			stored[key] = value
		}
	}

	for _, val := range stored {
		ret = append(ret, val)
	}

	sort.Slice(ret, func(i, j int) bool {
		return bytes.Compare(ret[i], ret[j]) < 0
	})

	return len(ret) != 0, ret
}

func (o *SimpleAccount) getJournalIfModified() *blockJournalEntry {
	entry := &blockJournalEntry{Address: o.Addr}

	if InnerAccountChanged(o.originAccount, o.dirtyAccount) {
		entry.AccountChanged = true
		entry.PrevAccount = o.originAccount
	}

	if o.originCode == nil && !(o.originAccount == nil || o.originAccount.CodeHash == nil) {
		o.originCode = o.ldb.Get(compositeKey(codeKey, o.Addr))
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

func (o *SimpleAccount) getStateJournalAndComputeHash() map[string][]byte {
	prevStates := make(map[string][]byte)
	var dirtyStateKeys []string
	var dirtyStateData []byte

	for key, value := range o.dirtyState {
		origVal := o.originState[key]
		if !bytes.Equal(origVal, value) {
			prevStates[key] = origVal
			dirtyStateKeys = append(dirtyStateKeys, key)
		}
	}

	sort.Strings(dirtyStateKeys)

	for _, key := range dirtyStateKeys {
		dirtyStateData = append(dirtyStateData, key...)
		dirtyVal := o.dirtyState[key]
		dirtyStateData = append(dirtyStateData, dirtyVal...)
	}
	hash := sha256.Sum256(dirtyStateData)
	o.dirtyStateHash = types.NewHash(hash[:])

	return prevStates
}

func (o *SimpleAccount) getDirtyData() []byte {
	var dirtyData []byte

	dirtyData = append(dirtyData, o.Addr.Bytes()...)

	if o.dirtyAccount != nil {
		data, err := o.dirtyAccount.Marshal()
		if err != nil {
			panic(err)
		}
		dirtyData = append(dirtyData, data...)
	}

	return append(dirtyData, o.dirtyStateHash.Bytes()...)
}

func (o *SimpleAccount) SetSuicided(suicided bool) {
	o.suicided = suicided
}

func (o *SimpleAccount) IsEmpty() bool {
	return o.GetBalance().Sign() == 0 && o.GetNonce() == 0 && o.Code() == nil && !o.suicided
}

func (o *SimpleAccount) Suicided() bool {
	return false
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
