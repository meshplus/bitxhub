package ledger

import (
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/storage"
)

type journal struct {
	Address        types.Address
	PrevAccount    *innerAccount
	AccountChanged bool
	PrevStates     map[string][]byte
	PrevCode       []byte
	CodeChanged    bool
}

type BlockJournal struct {
	Journals    []*journal
	ChangedHash types.Hash
}

func (journal *journal) revert(batch storage.Batch) {
	if journal.AccountChanged {
		if journal.PrevAccount != nil {
			data, err := journal.PrevAccount.Marshal()
			if err != nil {
				panic(err)
			}
			batch.Put(compositeKey(accountKey, journal.Address.Hex()), data)
		} else {
			batch.Delete(compositeKey(accountKey, journal.Address.Hex()))
		}
	}

	for key, val := range journal.PrevStates {
		if val != nil {
			batch.Put(compositeKey(journal.Address.Hex(), key), val)
		} else {
			batch.Delete(compositeKey(journal.Address.Hex(), key))
		}
	}

	if journal.CodeChanged {
		if journal.PrevCode != nil {
			batch.Put(compositeKey(codeKey, journal.Address.Hex()), journal.PrevCode)
		} else {
			batch.Delete(compositeKey(codeKey, journal.Address.Hex()))
		}
	}
}

func getLatestJournal(ldb storage.Storage) (uint64, *BlockJournal, error) {
	maxHeight := uint64(0)
	journal := &BlockJournal{}
	begin, end := bytesPrefix([]byte(journalKey))
	it := ldb.Iterator(begin, end)

	for it.Next() {
		height := uint64(0)
		_, err := fmt.Sscanf(string(it.Key()), journalKey+"%d", &height)
		if err != nil {
			return 0, nil, err
		}

		if height > maxHeight {
			maxHeight = height
			if err := json.Unmarshal(it.Value(), journal); err != nil {
				panic(err)
			}
		}
	}

	return maxHeight, journal, nil
}
