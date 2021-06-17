package ledger

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/eth-kit/ledger"
)

var _ ledger.IAccount = (*SimpleAccount)(nil)

type SimpleAccount struct {
	Addr           *types.Address
	originAccount  *ledger.InnerAccount
	dirtyAccount   *ledger.InnerAccount
	originState    sync.Map
	dirtyState     sync.Map
	originCode     []byte
	dirtyCode      []byte
	dirtyStateHash *types.Hash
	ldb            storage.Storage
	cache          *AccountCache
	lock           sync.RWMutex

	changer  *stateChanger
	suicided bool
}

func newAccount(ldb storage.Storage, cache *AccountCache, addr *types.Address, changer *stateChanger) *SimpleAccount {
	return &SimpleAccount{
		Addr:     addr,
		ldb:      ldb,
		cache:    cache,
		changer:  changer,
		suicided: false,
	}
}

func (o *SimpleAccount) GetAddress() *types.Address {
	return o.Addr
}

// GetState Get state from local cache, if not found, then get it from DB
func (o *SimpleAccount) GetState(key []byte) (bool, []byte) {
	if val, exist := o.dirtyState.Load(string(key)); exist {
		value := val.([]byte)
		return value != nil, value
	}

	if val, exist := o.originState.Load(string(key)); exist {
		value := val.([]byte)
		return value != nil, value
	}

	val, ok := o.cache.getState(o.Addr, string(key))
	if !ok {
		val = o.ldb.Get(composeStateKey(o.Addr, key))
	}

	o.originState.Store(string(key), val)

	return val != nil, val
}

func (o *SimpleAccount) GetCommittedState(key []byte) []byte {
	if val, exist := o.originState.Load(string(key)); exist {
		value := val.([]byte)
		if val != nil {
			return (&types.Hash{}).Bytes()
		}
		return value
	}

	val, ok := o.cache.getState(o.Addr, string(key))
	if !ok {
		val = o.ldb.Get(composeStateKey(o.Addr, key))
	}

	o.originState.Store(string(key), val)

	if val != nil {
		return (&types.Hash{}).Bytes()
	}
	return val
}

// SetState Set account state
func (o *SimpleAccount) SetState(key []byte, value []byte) {
	_, prev := o.GetState(key)
	o.dirtyState.Store(string(key), value)

	o.changer.append(storageChange{
		account:  o.Addr,
		key:      key,
		prevalue: prev,
	})
}

func (o *SimpleAccount) setState(key []byte, value []byte) {
	o.dirtyState.Store(string(key), value)
}

// AddState Add account state
func (o *SimpleAccount) AddState(key []byte, value []byte) {
	o.dirtyState.Store(string(key), value)
}

// SetCodeAndHash Set the contract code and hash
func (o *SimpleAccount) SetCodeAndHash(code []byte) {
	ret := crypto.Keccak256Hash(code)
	o.changer.append(codeChange{
		account:  o.Addr,
		prevcode: o.Code(),
	})
	if o.dirtyAccount == nil {
		o.dirtyAccount = ledger.CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.CodeHash = ret.Bytes()
	o.dirtyCode = code
}

func (o *SimpleAccount) setCodeAndHash(code []byte) {
	ret := crypto.Keccak256Hash(code)
	if o.dirtyAccount == nil {
		o.dirtyAccount = ledger.CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.CodeHash = ret.Bytes()
	o.dirtyCode = code
}

// Code return the contract code
func (o *SimpleAccount) Code() []byte {
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
		o.dirtyAccount = ledger.CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Nonce = nonce
}

func (o *SimpleAccount) setNonce(nonce uint64) {
	if o.dirtyAccount == nil {
		o.dirtyAccount = ledger.CopyOrNewIfEmpty(o.originAccount)
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
		o.dirtyAccount = ledger.CopyOrNewIfEmpty(o.originAccount)
	}
	o.dirtyAccount.Balance = balance
}

func (o *SimpleAccount) setBalance(balance *big.Int) {
	if o.dirtyAccount == nil {
		o.dirtyAccount = ledger.CopyOrNewIfEmpty(o.originAccount)
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

	cached := o.cache.query(o.Addr, prefix)

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

	o.dirtyState.Range(func(key, value interface{}) bool {
		if strings.HasPrefix(key.(string), prefix) {
			stored[key.(string)] = value.([]byte)
		}
		return true
	})

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

	if ledger.InnerAccountChanged(o.originAccount, o.dirtyAccount) {
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
	return o.GetBalance().Sign() == 0 && o.GetNonce() == 0 && o.Code() == nil && o.suicided == false
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
