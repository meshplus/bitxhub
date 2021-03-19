package genesis

import (
	"encoding/json"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
)

var (
	roleAddr = types.NewAddress(bytesutil.LeftPadBytes([]byte{13}, 20))
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, lg ledger.Ledger) error {
	body, err := json.Marshal(genesis.Admins)
	if err != nil {
		return err
	}

	lg.SetState(roleAddr, []byte("admin-roles"), body)

	for _, admin := range genesis.Admins {
		lg.SetBalance(types.NewAddressByStr(admin.Address), 100000000)
	}

	for k, v := range genesis.Strategy {
		lg.SetState(constant.GovernanceContractAddr.Address(), []byte(k), []byte(v))
	}

	accounts, journal := lg.FlushDirtyDataAndComputeJournal()
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

	lg.PersistBlockData(blockData)

	return nil
}
