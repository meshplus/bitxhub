package finance

import (
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/loggers"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/eth-kit/ledger/mock_ledger"
	"github.com/stretchr/testify/assert"
)

const repoRoot = "testdata"

func TestGetGasPrice(t *testing.T) {
	// Gas should be less than 5000
	GasPriceBySize(t, 100, 5000, nil)
	// Gas should be larger than 5000
	GasPriceBySize(t, 400, 5000, nil)
	// Gas should be equals to 5000
	GasPriceBySize(t, 250, 5000, nil)
	// Gas touch the ceiling
	GasPriceBySize(t, 400, 10000, nil)
	// Gas touch the floor
	GasPriceBySize(t, 100, 1000, nil)
	// Txs too much error
	GasPriceBySize(t, 700, 5000, ErrTxsOutOfRange)
	// parent gas out of range error
	GasPriceBySize(t, 100, 11000, ErrGasOutOfRange)
	// parent gas out of range error
	GasPriceBySize(t, 100, 900, ErrGasOutOfRange)
}

func TestMockTxsGasPriceChange(t *testing.T) {
	parentGasPrice := uint64(5000)
	roundSize := 300
	count := 0
	down := 0
	up := 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < roundSize; i++ {
		txs := rand.Intn(500) + 1
		if txs > 250 {
			count++
			up += txs - 250
		} else {
			down += 250 - txs
		}
		parentGasPrice = GasPriceBySize(t, txs, int64(parentGasPrice), nil)
	}
	t.Logf("larger than 250: %d, up: %d, down: %d\n", count, up, down)
}

func GasPriceBySize(t *testing.T, size int, parentGasPrice int64, expectErr error) uint64 {
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	chainLedger.EXPECT().Close().AnyTimes()
	stateLedger.EXPECT().Close().AnyTimes()
	defer mockLedger.Close()
	chainMeta := &types.ChainMeta{
		Height: 1,
	}
	// mock block for ledger
	chainLedger.EXPECT().GetChainMeta().Return(chainMeta).AnyTimes()
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
	chainLedger.EXPECT().GetBlock(uint64(1)).Return(block, nil).AnyTimes()
	config := generateMockConfig(t)
	initializeLog(t, config)

	gasPrice := NewGas(config, mockLedger)
	gas, err := gasPrice.GetGasPrice()
	if expectErr != nil {
		assert.EqualError(t, err, expectErr.Error())
		return 0
	}
	assert.Nil(t, err)
	return checkResult(t, block, config, parentGasPrice, gas)
}

func generateMockConfig(t *testing.T) *repo.Repo {
	repo, err := repo.Load(repoRoot, "", "../../config/bitxhub.toml", "../../config/network.toml")
	assert.Nil(t, err)
	return repo
}

func initializeLog(t *testing.T, repo *repo.Repo) {
	err := log.Initialize(
		log.WithReportCaller(repo.Config.Log.ReportCaller),
		log.WithPersist(true),
		log.WithFilePath(filepath.Join(repoRoot, repo.Config.Log.Dir)),
		log.WithFileName(repo.Config.Log.Filename),
		log.WithMaxAge(90*24*time.Hour),
		log.WithRotationTime(24*time.Hour),
	)
	assert.Nil(t, err)

	loggers.Initialize(repo.Config)
}

func checkResult(t *testing.T, block *types.Block, config *repo.Repo, parentGasPrice int64, gas uint64) uint64 {
	percentage := (float64(len(block.Transactions)) - float64(config.Config.Txpool.BatchSize)/2) / float64(config.Config.Txpool.BatchSize)
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
