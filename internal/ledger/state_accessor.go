package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"sort"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
)

var _ Ledger = (*ChainLedger)(nil)

// GetOrCreateAccount get the account, if not exist, create a new account
func (l *ChainLedger) GetOrCreateAccount(addr types.Address) *Account {
	l.lock.Lock()
	defer l.lock.Unlock()

	h := addr.Hex()
	value, ok := l.accounts[h]
	if ok {
		return value
	}

	account := l.GetAccount(addr)
	l.accounts[h] = account

	return account
}

// GetAccount get account info using account Address, if not found, create a new account
func (l *ChainLedger) GetAccount(addr types.Address) *Account {
	account := newAccount(l.ldb, l.accountCache, addr)

	if innerAccount, ok := l.accountCache.getInnerAccount(addr.Hex()); ok {
		account.originAccount = innerAccount
		return account
	}

	if data := l.ldb.Get(compositeKey(accountKey, addr.Hex())); data != nil {
		account.originAccount = &innerAccount{}
		if err := account.originAccount.Unmarshal(data); err != nil {
			panic(err)
		}
	}

	return account
}

// GetBalanec get account balance using account Address
func (l *ChainLedger) GetBalance(addr types.Address) uint64 {
	account := l.GetOrCreateAccount(addr)
	return account.GetBalance()
}

// SetBalance set account balance
func (l *ChainLedger) SetBalance(addr types.Address, value uint64) {
	account := l.GetOrCreateAccount(addr)
	account.SetBalance(value)
}

// GetState get account state value using account Address and key
func (l *ChainLedger) GetState(addr types.Address, key []byte) (bool, []byte) {
	account := l.GetOrCreateAccount(addr)
	return account.GetState(key)
}

// SetState set account state value using account Address and key
func (l *ChainLedger) SetState(addr types.Address, key []byte, v []byte) {
	account := l.GetOrCreateAccount(addr)
	account.SetState(key, v)
}

// SetCode set contract code
func (l *ChainLedger) SetCode(addr types.Address, code []byte) {
	account := l.GetOrCreateAccount(addr)
	account.SetCodeAndHash(code)
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
	account := l.GetOrCreateAccount(addr)
	account.SetNonce(nonce)
}

// QueryByPrefix query value using key
func (l *ChainLedger) QueryByPrefix(addr types.Address, prefix string) (bool, [][]byte) {
	account := l.GetOrCreateAccount(addr)
	return account.Query(prefix)
}

func (l *ChainLedger) Clear() {
	l.events = sync.Map{}
	l.accounts = make(map[string]*Account)
}

// FlushDirtyDataAndComputeJournal gets dirty accounts and computes block journal
func (l *ChainLedger) FlushDirtyDataAndComputeJournal() (map[string]*Account, *BlockJournal) {
	dirtyAccounts := make(map[string]*Account)
	var dirtyAccountData []byte
	var journals []*journal
	var sortedAddr []string
	accountData := make(map[string][]byte)

	for addr, account := range l.accounts {
		journal := account.getJournalIfModified()
		if journal != nil {
			journals = append(journals, journal)
			sortedAddr = append(sortedAddr, addr)
			accountData[addr] = account.getDirtyData()
			dirtyAccounts[addr] = account
		}
	}

	sort.Strings(sortedAddr)
	for _, addr := range sortedAddr {
		dirtyAccountData = append(dirtyAccountData, accountData[addr]...)
	}
	dirtyAccountData = append(dirtyAccountData, l.prevJnlHash[:]...)
	journalHash := sha256.Sum256(dirtyAccountData)

	blockJournal := &BlockJournal{
		Journals:    journals,
		ChangedHash: journalHash,
	}

	l.prevJnlHash = journalHash
	l.Clear()
	l.accountCache.add(dirtyAccounts)

	return dirtyAccounts, blockJournal
}

// Commit commit the state
func (l *ChainLedger) Commit(height uint64, accounts map[string]*Account, blockJournal *BlockJournal) error {
	ldbBatch := l.ldb.NewBatch()

	for _, account := range accounts {
		if innerAccountChanged(account.originAccount, account.dirtyAccount) {
			data, err := account.dirtyAccount.Marshal()
			if err != nil {
				panic(err)
			}
			ldbBatch.Put(compositeKey(accountKey, account.Addr.Hex()), data)
		}

		if !bytes.Equal(account.originCode, account.dirtyCode) {
			if account.dirtyCode != nil {
				ldbBatch.Put(compositeKey(codeKey, account.Addr.Hex()), account.dirtyCode)
			} else {
				ldbBatch.Delete(compositeKey(codeKey, account.Addr.Hex()))
			}
		}

		account.dirtyState.Range(func(key, value interface{}) bool {
			valBytes := value.([]byte)
			origVal, ok := account.originState.Load(key)
			var origValBytes []byte
			if ok {
				origValBytes = origVal.([]byte)
			}

			if !bytes.Equal(origValBytes, valBytes) {
				if valBytes != nil {
					ldbBatch.Put(composeStateKey(account.Addr, []byte(key.(string))), valBytes)
				} else {
					ldbBatch.Delete(composeStateKey(account.Addr, []byte(key.(string))))
				}
			}

			return true
		})
	}

	data, err := json.Marshal(blockJournal)
	if err != nil {
		return err
	}

	ldbBatch.Put(compositeKey(journalKey, height), data)
	ldbBatch.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(height))

	l.journalMutex.Lock()

	if l.minJnlHeight == 0 {
		l.minJnlHeight = height
		ldbBatch.Put(compositeKey(journalKey, minHeightStr), marshalHeight(height))
	}

	ldbBatch.Commit()

	l.maxJnlHeight = height

	l.journalMutex.Unlock()

	l.accountCache.remove(accounts)

	return nil
}

// Version returns the current version
func (l *ChainLedger) Version() uint64 {
	l.journalMutex.RLock()
	defer l.journalMutex.RUnlock()

	return l.maxJnlHeight
}

func (l *ChainLedger) rollbackState(height uint64) error {
	l.journalMutex.Lock()
	defer l.journalMutex.Unlock()

	if l.maxJnlHeight < height {
		return ErrorRollbackToHigherNumber
	}

	if l.minJnlHeight > height && !(l.minJnlHeight == 1 && height == 0) {
		return ErrorRollbackTooMuch
	}

	if l.maxJnlHeight == height {
		return nil
	}

	// clean cache account
	l.Clear()
	l.accountCache.clear()

	for i := l.maxJnlHeight; i > height; i-- {
		batch := l.ldb.NewBatch()

		blockJournal := getBlockJournal(i, l.ldb)
		if blockJournal == nil {
			return ErrorRollbackWithoutJournal
		}

		for _, journal := range blockJournal.Journals {
			journal.revert(batch)
		}

		batch.Delete(compositeKey(journalKey, i))
		batch.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(i-1))
		batch.Commit()
	}

	if height != 0 {
		journal := getBlockJournal(height, l.ldb)
		l.prevJnlHash = journal.ChangedHash
	} else {
		l.prevJnlHash = types.Hash{}
		l.minJnlHeight = 0
	}
	l.maxJnlHeight = height

	return nil
}
