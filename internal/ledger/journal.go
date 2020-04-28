package ledger

import (
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/pkg/storage"
)

var (
	minHeightStr = "minHeight"
	maxHeightStr = "maxHeight"
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
