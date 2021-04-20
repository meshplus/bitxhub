package ledger

import (
	"math/big"

	"github.com/meshplus/bitxhub-kit/types"
)

type stateChange interface {
	// revert undoes the state changes by this entry
	revert(*Ledger)

	// dirted returns the address modified by this state entry
	dirtied() *types.Address
}

type stateChanger struct {
	changes []stateChange
	dirties map[types.Address]int // dirty address and the number of changes
}

func newChanger() *stateChanger {
	return &stateChanger{
		dirties: make(map[types.Address]int),
	}
}

func (s *stateChanger) append(change stateChange) {
	s.changes = append(s.changes, change)
	if addr := change.dirtied(); addr != nil {
		s.dirties[*addr]++
	}
}

func (s *stateChanger) revert(ledger Ledger, snapshot int) {
	for i := len(s.changes) - 1; i >= snapshot; i-- {
		s.changes[i].revert(&ledger)

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
		prev *Account
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
		key, prevalue *types.Hash
	}

	codeChange struct {
		account            *types.Address
		prevcode, prevhash []byte
	}

	refundChange struct {
		prev uint64
	}

	addLogChange struct {
		txHash types.Hash
	}

	addPreimageChange struct {
		hash types.Hash
	}

	touchChange struct {
		account *types.Address
	}
)

func (ch *createObjectChange) revert(l *ChainLedger) {
	delete(l.accounts, ch.account.String())
	l.accountCache.rmAccount(ch.account)
}

func (ch *createObjectChange) dirtied() *types.Address {
	return ch.account
}

func (ch *resetObjectChange) revert(l *ChainLedger) {
	l.setAccount(ch.prev)
}

func (ch *resetObjectChange) dirtied() *types.Address {
	return nil
}

func (ch *suicideChange) revert(l *ChainLedger) {

}
