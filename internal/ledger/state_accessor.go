package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"

	"github.com/axiomesh/axiom-kit/types"
)

var _ StateLedger = (*StateLedgerImpl)(nil)

// GetOrCreateAccount get the account, if not exist, create a new account
func (l *StateLedgerImpl) GetOrCreateAccount(addr *types.Address) IAccount {
	account := l.GetAccount(addr)
	if account == nil {
		account = NewAccount(l.ldb, l.accountCache, addr, l.changer)
		l.changer.append(createObjectChange{account: addr})
		l.accounts[addr.String()] = account
	}

	return account
}

// GetAccount get account info using account Address, if not found, create a new account
func (l *StateLedgerImpl) GetAccount(address *types.Address) IAccount {
	addr := address.String()

	value, ok := l.accounts[addr]
	if ok {
		return value
	}

	account := NewAccount(l.ldb, l.accountCache, address, l.changer)

	if innerAccount, ok := l.accountCache.getInnerAccount(address); ok {
		account.originAccount = innerAccount
		if !bytes.Equal(innerAccount.CodeHash, nil) {
			code, okCode := l.accountCache.getCode(address)
			if !okCode {
				code = l.ldb.Get(compositeKey(codeKey, address))
			}
			account.originCode = code
			account.dirtyCode = code
		}
		l.accounts[addr] = account
		return account
	}

	if data := l.ldb.Get(compositeKey(accountKey, address)); data != nil {
		account.originAccount = &InnerAccount{Balance: big.NewInt(0)}
		if err := account.originAccount.Unmarshal(data); err != nil {
			panic(err)
		}
		if !bytes.Equal(account.originAccount.CodeHash, nil) {
			code := l.ldb.Get(compositeKey(codeKey, address))
			account.originCode = code
			account.dirtyCode = code
		}
		l.accounts[addr] = account
		return account
	}
	return nil
}

// nolint
func (l *StateLedgerImpl) setAccount(account IAccount) {
	l.accounts[account.GetAddress().String()] = account
}

// GetBalance get account balance using account Address
func (l *StateLedgerImpl) GetBalance(addr *types.Address) *big.Int {
	account := l.GetOrCreateAccount(addr)
	return account.GetBalance()
}

// SetBalance set account balance
func (l *StateLedgerImpl) SetBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	account.SetBalance(value)
}

func (l *StateLedgerImpl) SubBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	if !account.IsEmpty() {
		account.SubBalance(value)
	}
}

func (l *StateLedgerImpl) AddBalance(addr *types.Address, value *big.Int) {
	account := l.GetOrCreateAccount(addr)
	account.AddBalance(value)
}

// GetState get account state value using account Address and key
func (l *StateLedgerImpl) GetState(addr *types.Address, key []byte) (bool, []byte) {
	account := l.GetOrCreateAccount(addr)
	return account.GetState(key)
}

func (l *StateLedgerImpl) setTransientState(addr types.Address, key, value []byte) {
	l.transientStorage.Set(addr, common.BytesToHash(key), common.BytesToHash(value))
}

func (l *StateLedgerImpl) GetCommittedState(addr *types.Address, key []byte) []byte {
	account := l.GetOrCreateAccount(addr)
	if account.IsEmpty() {
		return (&types.Hash{}).Bytes()
	}
	return account.GetCommittedState(key)
}

// SetState set account state value using account Address and key
func (l *StateLedgerImpl) SetState(addr *types.Address, key []byte, v []byte) {
	account := l.GetOrCreateAccount(addr)
	account.SetState(key, v)
}

// SetCode set contract code
func (l *StateLedgerImpl) SetCode(addr *types.Address, code []byte) {
	account := l.GetOrCreateAccount(addr)
	account.SetCodeAndHash(code)
}

// GetCode get contract code
func (l *StateLedgerImpl) GetCode(addr *types.Address) []byte {
	account := l.GetOrCreateAccount(addr)
	return account.Code()
}

func (l *StateLedgerImpl) GetCodeHash(addr *types.Address) *types.Hash {
	account := l.GetOrCreateAccount(addr)
	if account.IsEmpty() {
		return &types.Hash{}
	}
	return types.NewHash(account.CodeHash())
}

func (l *StateLedgerImpl) GetCodeSize(addr *types.Address) int {
	account := l.GetOrCreateAccount(addr)
	if !account.IsEmpty() {
		if code := account.Code(); code != nil {
			return len(code)
		}
	}
	return 0
}

func (l *StateLedgerImpl) AddRefund(gas uint64) {
	l.changer.append(refundChange{prev: l.refund})
	l.refund += gas
}

func (l *StateLedgerImpl) SubRefund(gas uint64) {
	l.changer.append(refundChange{prev: l.refund})
	if gas > l.refund {
		panic(fmt.Sprintf("Refund counter below zero (gas: %d > refund: %d)", gas, l.refund))
	}
	l.refund -= gas
}

func (l *StateLedgerImpl) GetRefund() uint64 {
	return l.refund
}

// GetNonce get account nonce
func (l *StateLedgerImpl) GetNonce(addr *types.Address) uint64 {
	account := l.GetOrCreateAccount(addr)
	return account.GetNonce()
}

// SetNonce set account nonce
func (l *StateLedgerImpl) SetNonce(addr *types.Address, nonce uint64) {
	account := l.GetOrCreateAccount(addr)
	account.SetNonce(nonce)
}

// QueryByPrefix query value using key
func (l *StateLedgerImpl) QueryByPrefix(addr *types.Address, prefix string) (bool, [][]byte) {
	account := l.GetOrCreateAccount(addr)
	return account.Query(prefix)
}

func (l *StateLedgerImpl) Clear() {
	l.accounts = make(map[string]IAccount)
}

// FlushDirtyData gets dirty accounts and computes block journal
func (l *StateLedgerImpl) FlushDirtyData() (map[string]IAccount, *types.Hash) {
	dirtyAccounts := make(map[string]IAccount)
	var dirtyAccountData []byte
	var journals []*blockJournalEntry
	var sortedAddr []string
	accountData := make(map[string][]byte)

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
	l.blockJournals[blockJournal.ChangedHash.String()] = blockJournal
	l.prevJnlHash = blockJournal.ChangedHash
	l.Clear()
	if err := l.accountCache.add(dirtyAccounts); err != nil {
		panic(err)
	}

	return dirtyAccounts, blockJournal.ChangedHash
}

// Commit the state
func (l *StateLedgerImpl) Commit(height uint64, accounts map[string]IAccount, stateRoot *types.Hash) error {
	ldbBatch := l.ldb.NewBatch()

	for _, acc := range accounts {
		account := acc.(*SimpleAccount)
		if account.Suicided() {
			if data := l.ldb.Get(compositeKey(accountKey, account.Addr)); data != nil {
				ldbBatch.Delete(compositeKey(accountKey, account.Addr))
			}
			continue
		}
		if InnerAccountChanged(account.originAccount, account.dirtyAccount) {
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

		for key, valBytes := range account.dirtyState {
			origValBytes := account.originState[key]

			if !bytes.Equal(origValBytes, valBytes) {
				if valBytes != nil {
					ldbBatch.Put(composeStateKey(account.Addr, []byte(key)), valBytes)
				} else {
					ldbBatch.Delete(composeStateKey(account.Addr, []byte(key)))
				}
			}
		}
	}

	blockJournal, ok := l.blockJournals[stateRoot.String()]
	if !ok {
		return fmt.Errorf("cannot get block journal for block %d", height)
	}

	data, err := json.Marshal(blockJournal)
	if err != nil {
		return fmt.Errorf("marshal block journal error: %w", err)
	}

	ldbBatch.Put(compositeKey(journalKey, height), data)
	ldbBatch.Put(compositeKey(journalKey, maxHeightStr), marshalHeight(height))

	if l.minJnlHeight == 0 {
		l.minJnlHeight = height
		ldbBatch.Put(compositeKey(journalKey, minHeightStr), marshalHeight(height))
	}

	ldbBatch.Commit()

	l.maxJnlHeight = height

	if height > 10 {
		if err := l.removeJournalsBeforeBlock(height - 10); err != nil {
			return fmt.Errorf("remove journals before block %d failed: %w", height-10, err)
		}
	}
	l.blockJournals = make(map[string]*BlockJournal)

	return nil
}

// Version returns the current version
func (l *StateLedgerImpl) Version() uint64 {
	return l.maxJnlHeight
}

func (l *StateLedgerImpl) RollbackState(height uint64) error {
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

func (l *StateLedgerImpl) Suicide(addr *types.Address) bool {
	account := l.GetOrCreateAccount(addr)
	l.changer.append(suicideChange{
		account:     addr,
		prev:        account.Suicided(),
		prevbalance: new(big.Int).Set(account.GetBalance()),
	})
	account.SetSuicided(true)
	account.SetBalance(new(big.Int))

	return true
}

func (l *StateLedgerImpl) HasSuicide(addr *types.Address) bool {
	account := l.GetOrCreateAccount(addr)
	if account.IsEmpty() {
		return false
	}
	return account.Suicided()
}

func (l *StateLedgerImpl) Exist(addr *types.Address) bool {
	return !l.GetOrCreateAccount(addr).IsEmpty()
}

func (l *StateLedgerImpl) Empty(addr *types.Address) bool {
	return l.GetOrCreateAccount(addr).IsEmpty()
}

func (l *StateLedgerImpl) Snapshot() int {
	id := l.nextRevisionId
	l.nextRevisionId++
	l.validRevisions = append(l.validRevisions, revision{id: id, changerIndex: l.changer.length()})
	return id
}

func (l *StateLedgerImpl) RevertToSnapshot(revid int) {
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

func (l *StateLedgerImpl) ClearChangerAndRefund() {
	if len(l.changer.changes) > 0 {
		l.changer = NewChanger()
		l.refund = 0
	}
	l.validRevisions = l.validRevisions[:0]
	l.nextRevisionId = 0
}

func (l *StateLedgerImpl) AddAddressToAccessList(addr types.Address) {
	if l.accessList.AddAddress(addr) {
		l.changer.append(accessListAddAccountChange{address: &addr})
	}
}

func (l *StateLedgerImpl) AddSlotToAccessList(addr types.Address, slot types.Hash) {
	addrMod, slotMod := l.accessList.AddSlot(addr, slot)
	if addrMod {
		l.changer.append(accessListAddAccountChange{address: &addr})
	}
	if slotMod {
		l.changer.append(accessListAddSlotChange{
			address: &addr,
			slot:    &slot,
		})
	}
}

func (l *StateLedgerImpl) PrepareAccessList(sender types.Address, dst *types.Address, precompiles []types.Address, list AccessTupleList) {
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

func (l *StateLedgerImpl) AddressInAccessList(addr types.Address) bool {
	return l.accessList.ContainsAddress(addr)
}

func (l *StateLedgerImpl) SlotInAccessList(addr types.Address, slot types.Hash) (bool, bool) {
	return l.accessList.Contains(addr, slot)
}

func (l *StateLedgerImpl) AddPreimage(hash types.Hash, preimage []byte) {
	if _, ok := l.preimages[hash]; !ok {
		l.changer.append(addPreimageChange{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		l.preimages[hash] = pi
	}
}

func (l *StateLedgerImpl) PrepareBlock(hash *types.Hash, height uint64) {
	l.logs = NewEvmLogs()
	l.logs.bhash = hash
	l.blockHeight = height
}

func (l *StateLedgerImpl) AddLog(log *types.EvmLog) {
	if log.TransactionHash == nil {
		log.TransactionHash = l.thash
	}

	log.TransactionIndex = uint64(l.txIndex)

	l.changer.append(addLogChange{txHash: log.TransactionHash})

	log.BlockHash = l.logs.bhash
	log.LogIndex = uint64(l.logs.logSize)
	if _, ok := l.logs.logs[*log.TransactionHash]; !ok {
		l.logs.logs[*log.TransactionHash] = make([]*types.EvmLog, 0)
	}

	l.logs.logs[*log.TransactionHash] = append(l.logs.logs[*log.TransactionHash], log)
	l.logs.logSize++
}

func (l *StateLedgerImpl) GetLogs(hash types.Hash, height uint64, blockHash *types.Hash) []*types.EvmLog {
	logs := l.logs.logs[hash]
	for _, l := range logs {
		l.BlockNumber = height
		l.BlockHash = blockHash
	}
	return logs
}

func (l *StateLedgerImpl) Logs() []*types.EvmLog {
	var logs []*types.EvmLog
	for _, lgs := range l.logs.logs {
		logs = append(logs, lgs...)
	}
	return logs
}
