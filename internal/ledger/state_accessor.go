package ledger

import (
	"sort"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

var _ Ledger = (*ChainLedger)(nil)

const (
	accountKey = "account-"
)

// GetOrCreateAccount get the account, if not exist, create a new account
func (l *ChainLedger) GetOrCreateAccount(addr types.Address) *Account {
	h := addr.Hex()
	value, ok := l.accounts[h]
	if ok {
		return value
	}

	obj := newAccount(l, l.ldb, addr)

	l.accounts[h] = obj

	return obj
}

// GetAccount get account info using account address
func (l *ChainLedger) GetAccount(addr types.Address) *Account {
	h := addr.Hex()
	value, ok := l.accounts[h]
	if ok {
		return value
	}

	account := &Account{}
	_, data := l.tree.Get(compositeKey(accountKey, addr.Hex()))
	if data != nil {
		if err := account.Unmarshal(data); err != nil {
			panic(err)
		}
	}

	return account
}

// GetBalanec get account balance using account address
func (l *ChainLedger) GetBalance(addr types.Address) uint64 {
	account := l.GetOrCreateAccount(addr)

	return account.Balance
}

// SetBalance set account balance
func (l *ChainLedger) SetBalance(addr types.Address, value uint64) {
	h := addr.Hex()
	account := l.GetOrCreateAccount(addr)
	account.Balance = value

	l.accounts[h] = account
	l.modifiedAccount[h] = true
}

// GetState get account state value using account address and key
func (l *ChainLedger) GetState(addr types.Address, key []byte) (bool, []byte) {
	account := l.GetOrCreateAccount(addr)

	return account.GetState(key)
}

// SetState set account state value using account address and key
func (l *ChainLedger) SetState(addr types.Address, key []byte, v []byte) {
	h := addr.Hex()
	account := l.GetOrCreateAccount(addr)
	account.SetState(key, v)

	l.accounts[h] = account
	l.modifiedAccount[h] = true
}

// SetCode set contract code
func (l *ChainLedger) SetCode(addr types.Address, code []byte) {
	h := addr.Hex()
	account := l.GetOrCreateAccount(addr)
	account.SetCodeAndHash(code)

	l.accounts[h] = account
	l.modifiedAccount[h] = true
}

// GetCode get contract code
func (l *ChainLedger) GetCode(addr types.Address) []byte {
	account := l.GetOrCreateAccount(addr)
	return account.Code()
}

// GetNonce get account nonce
func (l *ChainLedger) GetNonce(addr types.Address) uint64 {
	account := l.GetOrCreateAccount(addr)
	return account.GetNonce()
}

// SetNonce set account nonce
func (l *ChainLedger) SetNonce(addr types.Address, nonce uint64) {
	h := addr.Hex()
	account := l.GetOrCreateAccount(addr)
	account.SetNonce(nonce)

	l.accounts[h] = account
	l.modifiedAccount[h] = true
}

// QueryByPrefix query value using key
func (l *ChainLedger) QueryByPrefix(addr types.Address, prefix string) (bool, [][]byte) {
	account := l.GetOrCreateAccount(addr)
	return account.Query(prefix)
}

func (l *ChainLedger) Clear() {
	l.events = make(map[string][]*pb.Event, 10)
	l.accounts = make(map[string]*Account)
	l.modifiedAccount = make(map[string]bool)
}

// Commit commit the state
func (l *ChainLedger) Commit() (types.Hash, error) {
	sk := make([]string, 0, len(l.modifiedAccount))
	for id := range l.modifiedAccount {
		sk = append(sk, id)
	}

	sort.Strings(sk)

	for _, id := range sk {
		obj := l.accounts[id]
		hash, err := obj.Commit()
		if err != nil {
			return types.Hash{}, err
		}

		obj.SetStateRoot(hash)

		data, err := obj.Marshal()
		if err != nil {
			return types.Hash{}, err
		}

		l.tree.Set(compositeKey(accountKey, id), data)
	}

	hash, height, err := l.tree.SaveVersion()
	if err != nil {
		return types.Hash{}, err
	}

	l.height = uint64(height)

	l.Clear()

	return types.Bytes2Hash(hash), nil
}

// Version returns the current version
func (l *ChainLedger) Version() uint64 {
	return uint64(l.tree.Version())
}
