package finance

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

func TestGetGasPrice(t *testing.T) {
	// Gas should be less than 5000
	GasPriceBySize(t, 100, 5000000000000, nil)
	// Gas should be larger than 5000
	GasPriceBySize(t, 400, 5000000000000, nil)
	// Gas should be equals to 5000
	GasPriceBySize(t, 250, 5000000000000, nil)
	// Gas touch the ceiling
	GasPriceBySize(t, 400, 10000000000000, nil)
	// Gas touch the floor
	GasPriceBySize(t, 100, 1000000000000, nil)
	// Txs too much error
	GasPriceBySize(t, 700, 5000000000000, ErrTxsOutOfRange)
	// parent gas out of range error
	GasPriceBySize(t, 100, 11000000000000, ErrGasOutOfRange)
	// parent gas out of range error
	GasPriceBySize(t, 100, 900000000000, ErrGasOutOfRange)
}

func GasPriceBySize(t *testing.T, size int, parentGasPrice int64, expectErr error) uint64 {
	// mock block for ledger
	block := &types.Block{
		BlockHeader:  &types.BlockHeader{GasPrice: parentGasPrice},
		Transactions: []*types.Transaction{},
	}
	prepareTxs := func(size int) []*types.Transaction {
		txs := []*types.Transaction{}
		for i := 0; i < size; i++ {
			txs = append(txs, &types.Transaction{})
		}
		return txs
	}
	block.Transactions = prepareTxs(size)
	config := generateMockConfig(t)
	gasPrice := NewGas(config)
	gas, err := gasPrice.CalNextGasPrice(uint64(parentGasPrice), size)
	if expectErr != nil {
		assert.EqualError(t, err, expectErr.Error())
		return 0
	}
	assert.Nil(t, err)
	return checkResult(t, block, config, parentGasPrice, gas)
}

func generateMockConfig(t *testing.T) *repo.Repo {
	repo, err := repo.Default(t.TempDir())
	assert.Nil(t, err)
	return repo
}

func checkResult(t *testing.T, block *types.Block, config *repo.Repo, parentGasPrice int64, gas uint64) uint64 {
	percentage := 2 * (float64(len(block.Transactions)) - float64(config.EpochInfo.ConsensusParams.BlockMaxTxNum)/2) / float64(config.EpochInfo.ConsensusParams.BlockMaxTxNum)
	actualGas := uint64(float64(parentGasPrice) * (1 + percentage*config.Config.Genesis.GasChangeRate))
	if actualGas > config.Config.Genesis.MaxGasPrice {
		actualGas = config.Config.Genesis.MaxGasPrice
	}
	if actualGas < config.Config.Genesis.MinGasPrice {
		actualGas = config.Config.Genesis.MinGasPrice
	}
	assert.Equal(t, uint64(actualGas), gas, "Gas price is not correct")
	return gas
}
