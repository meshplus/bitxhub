package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

var _ Ledger = (*ChainLedger)(nil)

// GetOrCreateAccount get the account, if not exist, create a new account
func (l *ChainLedger) GetOrCreateAccount(addr *types.Address) *Account {
	l.lock.RLock()
	value, ok := l.accounts[addr.String()]
	l.lock.RUnlock()
	if ok {
		return value
	}

	l.lock.Lock()
	defer l.lock.Unlock()
	if value, ok := l.accounts[addr.String()]; ok {
		return value
	}
	account := l.GetAccount(addr)
	l.accounts[addr.String()] = account

	return account
}

// GetAccount get account info using account Address, if not found, create a new account
func (l *ChainLedger) GetAccount(address *types.Address) *Account {
	account := newAccount(l.ldb, l.accountCache, address, l.changer)

	if innerAccount, ok := l.accountCache.getInnerAccount(address); ok {
		account.originAccount = innerAccount
		return account
	}

	if data := l.ldb.Get(compositeKey(accountKey, address)); data != nil {
		account.originAccount = &innerAccount{Balance: big.NewInt(0)}
		if err := account.originAccount.Unmarshal(data); err != nil {
			panic(err)
		}
		return account
	}

	l.changer.append(createObjectChange{account: address})
	return account
}

func (l *ChainLedger) setAccount(account *Account) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.accounts[account.Addr.String()] = account
}

// GetBalanec get account balance using account Address
func (l *ChainLedger) GetBalance(addr *types.Address) *big.Int {
	account := l.GetOrCreateAccount(addr)
	return account.GetBalance()
}

// SetBalance set account balance
func (l *ChainLedger) SetBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	account.SetBalance(value)
}

func (l *ChainLedger) SubBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	if !account.isEmpty() {
		account.SubBalance(value)
	}
}

func (l *ChainLedger) AddBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	account.AddBalance(value)
}

// GetState get account state value using account Address and key
func (l *ChainLedger) GetState(addr *types.Address, key []byte) (bool, []byte) {
	account := l.GetOrCreateAccount(addr)
	return account.GetState(key)
}

func (l *ChainLedger) GetCommittedState(addr *types.Address, key []byte) []byte {
	account := l.GetOrCreateAccount(addr)
	if account.isEmpty() {
		return (&types.Hash{}).Bytes()
	}
	return account.GetCommittedState(key)
}

// SetState set account state value using account Address and key
func (l *ChainLedger) SetState(addr *types.Address, key []byte, v []byte) {
	account := l.GetOrCreateAccount(addr)
	_, prev := account.GetState(key)

	account.SetState(key, v)
	l.changer.append(storageChange{
		account:  addr,
		key:      key,
		prevalue: prev,
	})
}

// AddState add account state value using account Address and key
func (l *ChainLedger) AddState(addr *types.Address, key []byte, v []byte) {
	account := l.GetOrCreateAccount(addr)
	account.AddState(key, v)
}

// SetCode set contract code
func (l *ChainLedger) SetCode(addr *types.Address, code []byte) {
	account := l.GetOrCreateAccount(addr)
	account.SetCodeAndHash(code)
}

// GetCode get contract code
func (l *ChainLedger) GetCode(addr *types.Address) []byte {
	account := l.GetOrCreateAccount(addr)
	return account.Code()
}

func (l *ChainLedger) GetCodeHash(addr *types.Address) *types.Hash {
	account := l.GetOrCreateAccount(addr)
	if account.isEmpty() {
		return &types.Hash{}
	}
	return types.NewHash(account.CodeHash())
}

func (l *ChainLedger) GetCodeSize(addr *types.Address) int {
	account := l.GetOrCreateAccount(addr)
	if !account.isEmpty() {
		if code := account.Code(); code != nil {
			return len(code)
		}
	}
	return 0
}

func (l *ChainLedger) AddRefund(gas uint64) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.changer.append(refundChange{prev: l.refund})
	l.refund += gas
}

func (l *ChainLedger) SubRefund(gas uint64) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.changer.append(refundChange{prev: l.refund})
	if gas > l.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, l.refund))
	}
	l.refund -= gas
}

func (l *ChainLedger) GetRefund() uint64 {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.refund
}

// GetNonce get account nonce
func (l *ChainLedger) GetNonce(addr *types.Address) uint64 {
	account := l.GetOrCreateAccount(addr)
	return account.GetNonce()
}

// SetNonce set account nonce
func (l *ChainLedger) SetNonce(addr *types.Address, nonce uint64) {
	account := l.GetOrCreateAccount(addr)
	account.SetNonce(nonce)
}

// QueryByPrefix query value using key
func (l *ChainLedger) QueryByPrefix(addr *types.Address, prefix string) (bool, [][]byte) {
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
	dirtyAccountData = append(dirtyAccountData, l.prevJnlHash.Bytes()...)
	journalHash := sha256.Sum256(dirtyAccountData)

	blockJournal := &BlockJournal{
		Journals:    journals,
		ChangedHash: types.NewHash(journalHash[:]),
	}

	l.prevJnlHash = blockJournal.ChangedHash
	l.Clear()
	l.accountCache.add(dirtyAccounts)

	return dirtyAccounts, blockJournal
}

// Commit commit the state
func (l *ChainLedger) Commit(height uint64, accounts map[string]*Account, blockJournal *BlockJournal) error {
	ldbBatch := l.ldb.NewBatch()

	for _, account := range accounts {
		if account.suicided {
			if data := l.ldb.Get(compositeKey(accountKey, account.Addr)); data != nil {
				ldbBatch.Delete(compositeKey(accountKey, account.Addr))
			}
			continue
		}
		if innerAccountChanged(account.originAccount, account.dirtyAccount) {
			data, err := account.dirtyAccount.Marshal()
			if err != nil {
				panic(err)
			}
			ldbBatch.Put(compositeKey(accountKey, account.Addr), data)
		}

		if !bytes.Equal(account.originCode, account.dirtyCode) {
			if account.dirtyCode != nil {
				ldbBatch.Put(compositeKey(codeKey, account.Addr), account.dirtyCode)
			} else {
				ldbBatch.Delete(compositeKey(codeKey, account.Addr))
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
		l.prevJnlHash = &types.Hash{}
		l.minJnlHeight = 0
	}
	l.maxJnlHeight = height

	return nil
}

func (l *ChainLedger) Suiside(addr *types.Address) bool {
	account := l.GetAccount(addr)
	l.changer.append(suicideChange{
		account:     addr,
		prev:        account.suicided,
		prevbalance: new(big.Int).Set(account.GetBalance()),
	})
	account.markSuicided()
	account.SetBalance(new(big.Int))

	return true
}

func (l *ChainLedger) HasSuiside(addr *types.Address) bool {
	account := l.GetOrCreateAccount(addr)
	if account.isEmpty() {
		return false
	}
	return account.suicided
}

func (l *ChainLedger) Exist(addr *types.Address) bool {
	return !l.GetOrCreateAccount(addr).isEmpty()
}

func (l *ChainLedger) Empty(addr *types.Address) bool {
	return l.GetOrCreateAccount(addr).isEmpty()
}

func (l *ChainLedger) Snapshot() int {
	id := l.nextRevisionId
	l.nextRevisionId++
	l.validRevisions = append(l.validRevisions, revision{id, l.changer.length()})
	return id
}

func (l *ChainLedger) RevertToSnapshot(revid int) {
	idx := sort.Search(len(l.validRevisions), func(i int) bool {
		return l.validRevisions[i].id >= revid
	})
	if idx == len(l.validRevisions) || l.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannod be reverted", revid))
	}
	snapshot := l.validRevisions[idx].changerIndex

	l.changer.revert(l, snapshot)
	l.validRevisions = l.validRevisions[:idx]
}

func (l *ChainLedger) ClearChangerAndRefund() {
	if len(l.changer.changes) > 0 {
		l.changer = newChanger()
		l.refund = 0
	}
	l.validRevisions = l.validRevisions[:0]
	l.nextRevisionId = 0
}

func (l *ChainLedger) AddAddressToAccessList(addr types.Address) {
	if l.accessList.AddAddress(addr) {
		l.changer.append(accessListAddAccountChange{&addr})
	}
}

func (l *ChainLedger) AddSlotToAccessList(addr types.Address, slot types.Hash) {
	addrMod, slotMod := l.accessList.AddSlot(addr, slot)
	if addrMod {
		l.changer.append(accessListAddAccountChange{&addr})
	}
	if slotMod {
		l.changer.append(accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}
}

func (l *ChainLedger) PrepareAccessList(sender types.Address, dst *types.Address, precompiles []types.Address, list AccessList) {
	l.AddAddressToAccessList(sender)

	if dst != nil {
		l.AddAddressToAccessList(*dst)
	}

	for _, addr := range precompiles {
		l.AddAddressToAccessList(addr)
	}
	for _, el := range list {
		l.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			l.AddSlotToAccessList(el.Address, key)
		}
	}
}

func (l *ChainLedger) AddressInAccessList(addr types.Address) bool {
	return l.accessList.ContainsAddress(addr)
}

func (l *ChainLedger) SlotInAccessList(addr types.Address, slot types.Hash) (bool, bool) {
	return l.accessList.Contains(addr, slot)
}

func (l *ChainLedger) AddPreimage(hash types.Hash, preimage []byte) {
	if _, ok := l.preimages[hash]; !ok {
		l.changer.append(addPreimageChange{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		l.preimages[hash] = pi
	}
}

func (l *ChainLedger) PrepareBlock(hash *types.Hash) {
	l.logs = NewEvmLogs()
	l.logs.bhash = hash
}

func (l *ChainLedger) AddLog(log *pb.EvmLog) {
	l.changer.append(addLogChange{txHash: l.logs.thash})

	log.TxHash = l.logs.thash
	log.BlockHash = l.logs.bhash
	log.TxIndex = uint64(l.logs.txIndex)
	log.Index = uint64(l.logs.logSize)
	l.logs.logs[*l.logs.thash] = append(l.logs.logs[*l.logs.thash], log)
	l.logs.logSize++
}

func (l *ChainLedger) GetLogs(hash types.Hash) []*pb.EvmLog {
	return l.logs.logs[hash]
}

func (l *ChainLedger) Logs() []*pb.EvmLog {
	var logs []*pb.EvmLog
	for _, lgs := range l.logs.logs {
		logs = append(logs, lgs...)
	}
	return logs
}
