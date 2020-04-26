package ledger

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/storage"
)

type journalEntry interface {
	revert(batch storage.Batch)
}

type (
	accountChange struct {
		address     types.Address
		prevAccount *innerAccount
	}

	stateChange struct {
		address    types.Address
		prevStates map[string][]byte
	}

	codeChange struct {
		address  types.Address
		prevCode []byte
	}
)

type BlockJournal struct {
	journals    []journalEntry
	changedHash types.Hash
}

func (journal accountChange) revert(batch storage.Batch) {
	if journal.prevAccount != nil {
		data, err := journal.prevAccount.Marshal()
		if err != nil {
			panic(err)
		}
		batch.Put(compositeKey(accountKey, journal.address.Hex()), data)
	} else {
		batch.Delete(compositeKey(accountKey, journal.address.Hex()))
	}
}

func (journal stateChange) revert(batch storage.Batch) {
	for key, val := range journal.prevStates {
		if val != nil {
			batch.Put(compositeKey(journal.address.Hex(), key), val)
		} else {
			batch.Delete(compositeKey(journal.address.Hex(), key))
		}
	}
}

func (journal codeChange) revert(batch storage.Batch) {
	if journal.prevCode != nil {
		batch.Put(compositeKey(codeKey, journal.address.Hex()), journal.prevCode)
	} else {
		batch.Delete(compositeKey(codeKey, journal.address.Hex()))
	}
}

func getHeightFromJournal(ldb storage.Storage) (uint64, error) {
	height := uint64(0)
	begin, end := bytesPrefix([]byte(journalKey))
	it := ldb.Iterator(begin, end)

	for it.Next() {
		h := uint64(0)
		_, e := fmt.Sscanf(string(it.Key()), journalKey+"%d", &h)
		if e != nil {
			return 0, e
		}

		if h > height {
			height = h
		}
	}

	return height, nil
}
