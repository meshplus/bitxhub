package finance

import (
	"errors"

	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/internal/loggers"
	"github.com/axiomesh/axiom/internal/repo"
	"github.com/sirupsen/logrus"
)

var (
	ErrTxsOutOfRange = errors.New("current txs is out of range")

	ErrGasOutOfRange = errors.New("parent gas price is out of range")
)

type Gas struct {
	repo   *repo.Repo
	ledger *ledger.Ledger
	logger logrus.FieldLogger
}

func NewGas(repo *repo.Repo, ledger *ledger.Ledger) *Gas {
	logger := loggers.Logger(loggers.Finance)
	return &Gas{repo: repo, ledger: ledger, logger: logger}
}

// GetGasPrice returns the current block gas price, based on the formula:
//
// G_c = G_p * (1 + (0.5 * (MaxBatchSize - TxsCount) / MaxBatchSize))
//
// if G_c <= minGasPrice, G_c = minGasPrice
//
// if G_c >= maxGasPrice, G_c = maxGasPrice
func (gas *Gas) GetGasPrice() (uint64, error) {
	latest := gas.ledger.ChainLedger.GetChainMeta().Height
	block, err := gas.ledger.ChainLedger.GetBlock(latest)
	if err != nil {
		return 0, errors.New("fail to get block")
	}
	gas.logger.Debugf("get %dth gas info: %v", latest, block.BlockHeader.GasPrice)
	max := gas.repo.Config.Genesis.MaxGasPrice
	min := gas.repo.Config.Genesis.MinGasPrice
	parentGasPrice := block.BlockHeader.GasPrice
	if uint64(parentGasPrice) < min || uint64(parentGasPrice) > max {
		gas.logger.Errorf("gas price is out of range, parent gas price is %d, min is %d, max is %d", parentGasPrice, min, max)
		return 0, ErrGasOutOfRange
	}
	total := gas.repo.Config.Txpool.BatchSize
	currentTxs := len(block.Transactions)
	if currentTxs > total {
		return 0, ErrTxsOutOfRange
	}
	percentage := float64(currentTxs-total/2) / float64(total) * gas.repo.Config.Genesis.GasChangeRate
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
