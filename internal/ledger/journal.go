package ledger

import (
	"encoding/hex"
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
		byteKey, err := hex.DecodeString(key)
		if err != nil {
			panic(err)
		}

		if val != nil {
			batch.Put(append(journal.Address.Bytes(), byteKey...), val)
		} else {
			batch.Delete(append(journal.Address.Bytes(), byteKey...))
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

func getBlockJournal(height uint64, ldb storage.Storage) *BlockJournal {
	data, err := ldb.Get(compositeKey(journalKey, height))
	if err != nil {
		panic(err)
	}

	journal := &BlockJournal{}
	if err := json.Unmarshal(data, journal); err != nil {
		panic(err)
	}

	return journal
}
