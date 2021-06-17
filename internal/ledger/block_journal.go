package ledger

import (
	"encoding/json"
	"strconv"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/eth-kit/ledger"
)

var (
	minHeightStr = "minHeight"
	maxHeightStr = "maxHeight"
)

type BlockJournal struct {
	Journals    []*blockJournalEntry
	ChangedHash *types.Hash
}

type blockJournalEntry struct {
	Address        *types.Address
	PrevAccount    *ledger.InnerAccount
	AccountChanged bool
	PrevStates     map[string][]byte
	PrevCode       []byte
	CodeChanged    bool
}

func revertJournal(journal *blockJournalEntry, batch storage.Batch) {
	if journal.AccountChanged {
		if journal.PrevAccount != nil {
			data, err := journal.PrevAccount.Marshal()
			if err != nil {
				panic(err)
			}
			batch.Put(compositeKey(accountKey, journal.Address), data)
		} else {
			batch.Delete(compositeKey(accountKey, journal.Address))
		}
	}

	for key, val := range journal.PrevStates {
		if val != nil {
			batch.Put(composeStateKey(journal.Address, []byte(key)), val)
		} else {
			batch.Delete(composeStateKey(journal.Address, []byte(key)))
		}
	}

	if journal.CodeChanged {
		if journal.PrevCode != nil {
			batch.Put(compositeKey(codeKey, journal.Address), journal.PrevCode)
		} else {
			batch.Delete(compositeKey(codeKey, journal.Address))
		}
	}
}

func getJournalRange(ldb storage.Storage) (uint64, uint64) {
	minHeight := uint64(0)
	maxHeight := uint64(0)

	data := ldb.Get(compositeKey(journalKey, minHeightStr))
	if data != nil {
		minHeight = unmarshalHeight(data)
	}

	data = ldb.Get(compositeKey(journalKey, maxHeightStr))
	if data != nil {
		maxHeight = unmarshalHeight(data)
	}

	return minHeight, maxHeight
}

func getBlockJournal(height uint64, ldb storage.Storage) *BlockJournal {
	data := ldb.Get(compositeKey(journalKey, height))
	if data == nil {
		return nil
	}

	journal := &BlockJournal{}
	if err := json.Unmarshal(data, journal); err != nil {
		panic(err)
	}

	return journal
}

func marshalHeight(height uint64) []byte {
	return []byte(strconv.FormatUint(height, 10))
}

func unmarshalHeight(data []byte) uint64 {
	height, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		panic(err)
	}

	return height
}
