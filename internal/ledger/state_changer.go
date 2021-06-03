package ledger

import (
	"math/big"
	"sync"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/eth-kit/ledger"
)

type stateChange interface {
	// revert undoes the state changes by this entry
	revert(*SimpleLedger)

	// dirted returns the address modified by this state entry
	dirtied() *types.Address
}

type stateChanger struct {
	changes []stateChange
	dirties map[types.Address]int // dirty address and the number of changes

	lock sync.RWMutex
}

func newChanger() *stateChanger {
	return &stateChanger{
		dirties: make(map[types.Address]int),
	}
}

func (s *stateChanger) append(change stateChange) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.changes = append(s.changes, change)
	if addr := change.dirtied(); addr != nil {
		s.dirties[*addr]++
	}
}

func (s *stateChanger) revert(ledger *SimpleLedger, snapshot int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i := len(s.changes) - 1; i >= snapshot; i-- {
		s.changes[i].revert(ledger)

		if addr := s.changes[i].dirtied(); addr != nil {
			if s.dirties[*addr]--; s.dirties[*addr] == 0 {
				delete(s.dirties, *addr)
			}
		}
	}

	s.changes = s.changes[:snapshot]
}

func (s *stateChanger) dirty(addr types.Address) {
	s.dirties[addr]++
}

func (s *stateChanger) length() int {
	return len(s.changes)
}

type (
	createObjectChange struct {
		account *types.Address
	}
	resetObjectChange struct {
		prev ledger.IAccount
	}
	suicideChange struct {
		account     *types.Address
		prev        bool
		prevbalance *big.Int
	}
	balanceChange struct {
		account *types.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *types.Address
		prev    uint64
	}
	storageChange struct {
		account       *types.Address
		key, prevalue []byte
	}
	codeChange struct {
		account  *types.Address
		prevcode []byte
	}
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txHash *types.Hash
	}
	addPreimageChange struct {
		hash types.Hash
	}
	touchChange struct {
		account *types.Address
	}
	accessListAddAccountChange struct {
		address *types.Address
	}
	accessListAddSlotChange struct {
		address *types.Address
		slot    *types.Hash
	}
)

func (ch createObjectChange) revert(l *SimpleLedger) {
	delete(l.accounts, ch.account.String())
	l.accountCache.rmAccount(ch.account)
}

func (ch createObjectChange) dirtied() *types.Address {
	return ch.account
}

func (ch resetObjectChange) revert(l *SimpleLedger) {
	l.setAccount(ch.prev)
}

func (ch resetObjectChange) dirtied() *types.Address {
	return nil
}

func (ch suicideChange) revert(l *SimpleLedger) {
	acc := l.GetOrCreateAccount(ch.account)
	account := acc.(*SimpleAccount)
	account.suicided = ch.prev
	account.setBalance(ch.prevbalance)
}

func (ch suicideChange) dirtied() *types.Address {
	return ch.account
}

func (ch touchChange) revert(l *SimpleLedger) {
}

func (ch touchChange) dirtied() *types.Address {
	return ch.account
}

func (ch balanceChange) revert(l *SimpleLedger) {
	l.GetOrCreateAccount(ch.account).(*SimpleAccount).setBalance(ch.prev)
}

func (ch balanceChange) dirtied() *types.Address {
	return ch.account
}

func (ch nonceChange) revert(l *SimpleLedger) {
	l.GetOrCreateAccount(ch.account).(*SimpleAccount).setNonce(ch.prev)
}

func (ch nonceChange) dirtied() *types.Address {
	return ch.account
}

func (ch codeChange) revert(l *SimpleLedger) {
	l.GetOrCreateAccount(ch.account).(*SimpleAccount).setCodeAndHash(ch.prevcode)
}

func (ch codeChange) dirtied() *types.Address {
	return ch.account
}

func (ch storageChange) revert(l *SimpleLedger) {
	l.GetOrCreateAccount(ch.account).(*SimpleAccount).setState(ch.key, ch.prevalue)
}

func (ch storageChange) dirtied() *types.Address {
	return ch.account
}

func (ch refundChange) revert(l *SimpleLedger) {
	l.refund = ch.prev
}

func (ch refundChange) dirtied() *types.Address {
	return nil
}

func (ch addPreimageChange) revert(l *SimpleLedger) {
	delete(l.preimages, ch.hash)
}

func (ch addPreimageChange) dirtied() *types.Address {
	return nil
}

func (ch accessListAddAccountChange) revert(l *SimpleLedger) {
	l.accessList.DeleteAddress(*ch.address)
}

func (ch accessListAddAccountChange) dirtied() *types.Address {
	return nil
}

func (ch accessListAddSlotChange) revert(l *SimpleLedger) {
	l.accessList.DeleteSlot(*ch.address, *ch.slot)
}

func (ch accessListAddSlotChange) dirtied() *types.Address {
	return nil
}

func (ch addLogChange) revert(l *SimpleLedger) {
	logs := l.logs.logs[*ch.txHash]
	if len(logs) == 1 {
		delete(l.logs.logs, *ch.txHash)
	} else {
		l.logs.logs[*ch.txHash] = logs[:len(logs)-1]
	}
	l.logs.logSize--
}

func (ch addLogChange) dirtied() *types.Address {
	return nil
}
