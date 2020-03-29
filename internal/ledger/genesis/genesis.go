package genesis

import (
	"encoding/json"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
)

var (
	roleAddr = types.Bytes2Address(bytesutil.LeftPadBytes([]byte{13}, 20))
)

// Initialize initialize block
func Initialize(config *repo.Config, ledger ledger.Ledger) error {
	for _, addr := range config.Addresses {
		ledger.SetBalance(types.String2Address(addr), 100000000)
	}

	body, err := json.Marshal(config.Genesis.Addresses)
	if err != nil {
		return err
	}

	ledger.SetState(roleAddr, []byte("admin-roles"), body)

	hash, err := ledger.Commit()
	if err != nil {
		return err
	}

	block := &pb.Block{
		BlockHeader: &pb.BlockHeader{
			Number:    1,
			StateRoot: hash,
		},
	}

	block.BlockHash = block.Hash()

	return ledger.PersistExecutionResult(block, nil)
}
