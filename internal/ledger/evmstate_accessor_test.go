package ledger

import (
	"testing"

	"github.com/axiomesh/axiom-kit/types"
)

func TestCreateBloom(t *testing.T) {
	addr := types.NewAddressByStr("0x0000000000000000000000000000000000001001")

	receipt := &types.Receipt{
		EvmLogs: []*types.EvmLog{
			&types.EvmLog{
				Address: addr,
				Topics: []*types.Hash{
					types.NewHashByStr("0xe6bfc3cff2e28bc2ab583f413a459f93526e55a1a46c944572150de96997c84e"),
					types.NewHashByStr("0x0000000000000000000000000000000000000000000000000000000000000001"),
				},
			},
		},
	}

	receipt.Bloom = CreateBloom(EvmReceipts{receipt})

	receipts := EvmReceipts{
		receipt,
	}

	bloom := CreateBloom(receipts)

	t.Logf("%v", bloom.Test(addr.Bytes()))
}
