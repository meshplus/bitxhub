package finance

import (
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-ledger/pkg/loggers"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

var (
	ErrTxsOutOfRange = errors.New("current txs is out of range")

	ErrGasOutOfRange = errors.New("parent gas price is out of range")
)

type Gas struct {
	repo   *repo.Repo
	logger logrus.FieldLogger
}

func NewGas(repo *repo.Repo) *Gas {
	logger := loggers.Logger(loggers.Finance)
	return &Gas{repo: repo, logger: logger}
}

// CalNextGasPrice returns the current block gas price, based on the formula:
//
// G_c = G_p * (1 + (0.5 * (MaxBatchSize - TxsCount) / MaxBatchSize))
//
// if G_c <= minGasPrice, G_c = minGasPrice
//
// if G_c >= maxGasPrice, G_c = maxGasPrice
func (gas *Gas) CalNextGasPrice(parentGasPrice uint64, txs int) (uint64, error) {
	max := gas.repo.Config.Genesis.MaxGasPrice
	min := gas.repo.Config.Genesis.MinGasPrice
	if uint64(parentGasPrice) < min || uint64(parentGasPrice) > max {
		gas.logger.Errorf("gas price is out of range, parent gas price is %d, min is %d, max is %d", parentGasPrice, min, max)
		return 0, ErrGasOutOfRange
	}
	total := int(gas.repo.EpochInfo.ConsensusParams.BlockMaxTxNum)
	if txs > total {
		return 0, ErrTxsOutOfRange
	}
	percentage := 2 * float64(txs-total/2) / float64(total) * gas.repo.Config.Genesis.GasChangeRate
	currentPrice := uint64(float64(parentGasPrice) * (1 + percentage))
	if currentPrice > max {
		gas.logger.Warningf("gas price is touching ceiling, current price is %d, max is %d", currentPrice, max)
		currentPrice = max
	}
	if currentPrice < min {
		gas.logger.Warningf("gas price is touching floor, current price is %d, min is %d", currentPrice, min)
		currentPrice = min
	}
	return currentPrice, nil
}
