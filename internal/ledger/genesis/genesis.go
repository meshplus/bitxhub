package genesis

import (
	"encoding/json"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
)

var (
	roleAddr = types.Bytes2Address(bytesutil.LeftPadBytes([]byte{13}, 20))
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, lg ledger.Ledger) error {
	for _, addr := range genesis.Addresses {
		if err := lg.SetBalance(types.String2Address(addr), 100000000); err != nil {
			return err
		}
	}

	body, err := json.Marshal(genesis.Addresses)
	if err != nil {
		return err
	}

	if err := lg.SetState(roleAddr, []byte("admin-roles"), body); err != nil {
		return err
	}

	accounts, journal, err := lg.FlushDirtyDataAndComputeJournal()
	if err != nil {
		return err
	}

	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number:    1,
			StateRoot: journal.ChangedHash,
		},
	}
	block.BlockHash = block.Hash()
	blockData := &ledger.BlockData{
		Block:          block,
		Receipts:       nil,
		Accounts:       accounts,
		Journal:        journal,
		InterchainMeta: &pb.InterchainMeta{},
	}

	if err := lg.PersistBlockData(blockData); err != nil {
		return err
	}

	return nil
}
