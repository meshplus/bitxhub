package genesis

import (
	"encoding/json"
	"math/big"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	ledger2 "github.com/meshplus/eth-kit/ledger"
)

var (
	roleAddr = types.NewAddress(bytesutil.LeftPadBytes([]byte{13}, 20))
)

// Initialize initialize block
func Initialize(genesis *repo.Genesis, lg *ledger.Ledger) error {
	body, err := json.Marshal(genesis.Admins)
	if err != nil {
		return err
	}

	admin, err := json.Marshal(genesis.Dider)
	if err != nil {
		return err
	}

	lg.SetState(roleAddr, []byte("admin-roles"), body)

	lg.SetState(constant.MethodRegistryContractAddr.Address(), []byte("admin-method"), admin)
	lg.SetState(constant.DIDRegistryContractAddr.Address(), []byte("admin-did"), admin)

	balance, _ := new(big.Int).SetString("100000000000000000000000000000000000", 10)
	for _, admin := range genesis.Admins {
		lg.SetBalance(types.NewAddressByStr(admin.Address), balance)
	}

	for k, v := range genesis.Strategy {
		lg.SetState(constant.GovernanceContractAddr.Address(), []byte(k), []byte(v))
	}

	accounts, journal := lg.FlushDirtyDataAndComputeJournal()

	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number: 1,
			//StateRoot: journal.ChangedHash,
			Bloom: &types.Bloom{},
		},
		Transactions: &pb.Transactions{},
	}
	//block.BlockHash = block.Hash()
	blockData := &ledger2.BlockData{
		Block:          block,
		Receipts:       nil,
		Accounts:       accounts,
		Journal:        journal,
		InterchainMeta: &pb.InterchainMeta{},
	}

	lg.PersistBlockData(blockData)

	return nil
}
