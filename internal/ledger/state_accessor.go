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
	"github.com/meshplus/eth-kit/ledger"
)

var _ ledger.StateLedger = (*SimpleLedger)(nil)

// GetOrCreateAccount get the account, if not exist, create a new account
func (l *SimpleLedger) GetOrCreateAccount(addr *types.Address) ledger.IAccount {
	account := l.GetAccount(addr)
	if account == nil {
		account = newAccount(l.ldb, l.accountCache, addr, l.changer)
		l.changer.append(createObjectChange{account: addr})

		l.lock.Lock()
		l.accounts[addr.String()] = account
		l.lock.Unlock()
	}

	return account
}

// GetAccount get account info using account Address, if not found, create a new account
func (l *SimpleLedger) GetAccount(address *types.Address) ledger.IAccount {
	addr := address.String()

	l.lock.RLock()
	value, ok := l.accounts[addr]
	l.lock.RUnlock()
	if ok {
		return value
	}

	account := newAccount(l.ldb, l.accountCache, address, l.changer)

	if innerAccount, ok := l.accountCache.getInnerAccount(address); ok {
		account.originAccount = innerAccount
		if !bytes.Equal(innerAccount.CodeHash, nil) {
			code, okCode := l.accountCache.getCode(address)
			if !okCode {
				code = l.ldb.Get(compositeKey(codeKey, address))
			}
			account.originCode = code
		}
		l.lock.Lock()
		l.accounts[addr] = account
		l.lock.Unlock()
		return account
	}

	if data := l.ldb.Get(compositeKey(accountKey, address)); data != nil {
		account.originAccount = &ledger.InnerAccount{Balance: big.NewInt(0)}
		if err := account.originAccount.Unmarshal(data); err != nil {
			panic(err)
		}
		if !bytes.Equal(account.originAccount.CodeHash, nil) {
			code := l.ldb.Get(compositeKey(codeKey, address))
			account.originCode = code
		}
		l.lock.Lock()
		l.accounts[addr] = account
		l.lock.Unlock()
		return account
	}

	return nil
}

func (l *SimpleLedger) setAccount(account ledger.IAccount) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.accounts[account.GetAddress().String()] = account
}

// GetBalanec get account balance using account Address
func (l *SimpleLedger) GetBalance(addr *types.Address) *big.Int {
	account := l.GetOrCreateAccount(addr)
	return account.GetBalance()
}

// SetBalance set account balance
func (l *SimpleLedger) SetBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	account.SetBalance(value)
}

func (l *SimpleLedger) SubBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	if !account.IsEmpty() {
		account.SubBalance(value)
	}
}

func (l *SimpleLedger) AddBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	account.AddBalance(value)
}

// GetState get account state value using account Address and key
func (l *SimpleLedger) GetState(addr *types.Address, key []byte) (bool, []byte) {
	account := l.GetOrCreateAccount(addr)
	return account.GetState(key)
}

func (l *SimpleLedger) GetCommittedState(addr *types.Address, key []byte) []byte {
	account := l.GetOrCreateAccount(addr)
	if account.IsEmpty() {
		return (&types.Hash{}).Bytes()
	}
	return account.GetCommittedState(key)
}

// SetState set account state value using account Address and key
func (l *SimpleLedger) SetState(addr *types.Address, key []byte, v []byte) {
	account := l.GetOrCreateAccount(addr)
	account.SetState(key, v)
}

// AddState add account state value using account Address and key
func (l *SimpleLedger) AddState(addr *types.Address, key []byte, v []byte) {
	account := l.GetOrCreateAccount(addr)
	account.AddState(key, v)
}

// SetCode set contract code
func (l *SimpleLedger) SetCode(addr *types.Address, code []byte) {
	account := l.GetOrCreateAccount(addr)
	account.SetCodeAndHash(code)
}

// GetCode get contract code
func (l *SimpleLedger) GetCode(addr *types.Address) []byte {
	account := l.GetOrCreateAccount(addr)
	return account.Code()
}

func (l *SimpleLedger) GetCodeHash(addr *types.Address) *types.Hash {
	account := l.GetOrCreateAccount(addr)
	if account.IsEmpty() {
		return &types.Hash{}
	}
	return types.NewHash(account.CodeHash())
}

func (l *SimpleLedger) GetCodeSize(addr *types.Address) int {
	account := l.GetOrCreateAccount(addr)
	if !account.IsEmpty() {
		if code := account.Code(); code != nil {
			return len(code)
		}
	}
	return 0
}

func (l *SimpleLedger) AddRefund(gas uint64) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.changer.append(refundChange{prev: l.refund})
	l.refund += gas
}

func (l *SimpleLedger) SubRefund(gas uint64) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.changer.append(refundChange{prev: l.refund})
	if gas > l.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, l.refund))
	}
	l.refund -= gas
}

func (l *SimpleLedger) GetRefund() uint64 {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.refund
}

// GetNonce get account nonce
func (l *SimpleLedger) GetNonce(addr *types.Address) uint64 {
	account := l.GetOrCreateAccount(addr)
	return account.GetNonce()
}

// SetNonce set account nonce
func (l *SimpleLedger) SetNonce(addr *types.Address, nonce uint64) {
	account := l.GetOrCreateAccount(addr)
	account.SetNonce(nonce)
}

// QueryByPrefix query value using key
func (l *SimpleLedger) QueryByPrefix(addr *types.Address, prefix string) (bool, [][]byte) {
	account := l.GetOrCreateAccount(addr)
	return account.Query(prefix)
}

func (l *SimpleLedger) Clear() {
	l.events = sync.Map{}

	l.lock.Lock()
	l.accounts = make(map[string]ledger.IAccount)
	l.lock.Unlock()
}

// FlushDirtyData gets dirty accounts and computes block journal
func (l *SimpleLedger) FlushDirtyData() (map[string]ledger.IAccount, *types.Hash) {
	dirtyAccounts := make(map[string]ledger.IAccount)
	var dirtyAccountData []byte
	var journals []*blockJournalEntry
	var sortedAddr []string
	accountData := make(map[string][]byte)

	l.lock.RLock()
	for addr, acc := range l.accounts {
		account := acc.(*SimpleAccount)
		journal := account.getJournalIfModified()
		if journal != nil {
			journals = append(journals, journal)
			sortedAddr = append(sortedAddr, addr)
			accountData[addr] = account.getDirtyData()
			dirtyAccounts[addr] = account
		}
	}
	l.lock.RUnlock()

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
	l.blockJournals.Store(blockJournal.ChangedHash.String(), blockJournal)
	l.prevJnlHash = blockJournal.ChangedHash
	l.Clear()
	l.accountCache.add(dirtyAccounts)

	return dirtyAccounts, blockJournal.ChangedHash
}

// Commit commit the state
func (l *SimpleLedger) Commit(height uint64, accounts map[string]ledger.IAccount, stateRoot *types.Hash) error {
	ldbBatch := l.ldb.NewBatch()

	for _, acc := range accounts {
		account := acc.(*SimpleAccount)
		if account.Suicided() {
			if data := l.ldb.Get(compositeKey(accountKey, account.Addr)); data != nil {
				ldbBatch.Delete(compositeKey(accountKey, account.Addr))
			}
			continue
		}
		if ledger.InnerAccountChanged(account.originAccount, account.dirtyAccount) {
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
				if account.dirtyAccount != nil {
					if !bytes.Equal(account.originAccount.CodeHash, account.dirtyAccount.CodeHash) {
						if bytes.Equal(account.dirtyAccount.CodeHash, nil) {
							ldbBatch.Delete(compositeKey(codeKey, account.Addr))
						}
					}
				}
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

	value, ok := l.blockJournals.Load(stateRoot.String())
	if !ok {
		return fmt.Errorf("cannot get block journal for block %d", height)
	}

	blockJournal := value.(*BlockJournal)
	data, err := json.Marshal(blockJournal)
	if err != nil {
		return fmt.Errorf("marshal block journal error: %w", err)
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

	if height > 10 {
		if err := l.removeJournalsBeforeBlock(height - 10); err != nil {
			return fmt.Errorf("remove journals before block %d failed: %w", height-10, err)
		}
	}
	l.blockJournals = sync.Map{}

	return nil
}

// Version returns the current version
func (l *SimpleLedger) Version() uint64 {
	l.journalMutex.RLock()
	defer l.journalMutex.RUnlock()

	return l.maxJnlHeight
}

func (l *SimpleLedger) RollbackState(height uint64) error {
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
			revertJournal(journal, batch)
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

func (l *SimpleLedger) Suiside(addr *types.Address) bool {
	account := l.GetAccount(addr).(*SimpleAccount)
	l.changer.append(suicideChange{
		account:     addr,
		prev:        account.Suicided(),
		prevbalance: new(big.Int).Set(account.GetBalance()),
	})
	account.SetSuicided(true)
	account.SetBalance(new(big.Int))

	return true
}

func (l *SimpleLedger) HasSuiside(addr *types.Address) bool {
	account := l.GetOrCreateAccount(addr)
	if account.IsEmpty() {
		return false
	}
	return account.Suicided()
}

func (l *SimpleLedger) Exist(addr *types.Address) bool {
	return !l.GetOrCreateAccount(addr).IsEmpty()
}

func (l *SimpleLedger) Empty(addr *types.Address) bool {
	return l.GetOrCreateAccount(addr).IsEmpty()
}

func (l *SimpleLedger) Snapshot() int {
	id := l.nextRevisionId
	l.nextRevisionId++
	l.validRevisions = append(l.validRevisions, revision{id, l.changer.length()})
	return id
}

func (l *SimpleLedger) RevertToSnapshot(revid int) {
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

func (l *SimpleLedger) ClearChangerAndRefund() {
	if len(l.changer.changes) > 0 {
		l.changer = newChanger()
		l.refund = 0
	}
	l.validRevisions = l.validRevisions[:0]
	l.nextRevisionId = 0
}

func (l *SimpleLedger) AddAddressToAccessList(addr types.Address) {
	if l.accessList.AddAddress(addr) {
		l.changer.append(accessListAddAccountChange{&addr})
	}
}

func (l *SimpleLedger) AddSlotToAccessList(addr types.Address, slot types.Hash) {
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

func (l *SimpleLedger) PrepareAccessList(sender types.Address, dst *types.Address, precompiles []types.Address, list ledger.AccessTupleList) {
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

func (l *SimpleLedger) AddressInAccessList(addr types.Address) bool {
	return l.accessList.ContainsAddress(addr)
}

func (l *SimpleLedger) SlotInAccessList(addr types.Address, slot types.Hash) (bool, bool) {
	return l.accessList.Contains(addr, slot)
}

func (l *SimpleLedger) AddPreimage(hash types.Hash, preimage []byte) {
	if _, ok := l.preimages[hash]; !ok {
		l.changer.append(addPreimageChange{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		l.preimages[hash] = pi
	}
}

func (l *SimpleLedger) PrepareBlock(hash *types.Hash, height uint64) {
	l.logs = NewEvmLogs()
	l.logs.bhash = hash
	l.blockHeight = height
}

func (l *SimpleLedger) AddLog(log *pb.EvmLog) {
	l.changer.append(addLogChange{txHash: l.logs.thash})

	log.TxHash = l.logs.thash
	log.BlockHash = l.logs.bhash
	log.TxIndex = uint64(l.logs.txIndex)
	log.Index = uint64(l.logs.logSize)
	l.logs.logs[*l.logs.thash] = append(l.logs.logs[*l.logs.thash], log)
	l.logs.logSize++
}

func (l *SimpleLedger) GetLogs(hash types.Hash) []*pb.EvmLog {
	return l.logs.logs[hash]
}

func (l *SimpleLedger) Logs() []*pb.EvmLog {
	var logs []*pb.EvmLog
	for _, lgs := range l.logs.logs {
		logs = append(logs, lgs...)
	}
	return logs
}
