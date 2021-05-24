package executor

import (
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

type SerialExecutor struct {
	normalTxs         []*types.Hash
	interchainCounter map[string][]*pb.VerifiedIndex
	applyTxFunc       agency.ApplyTxFunc
	boltContracts     map[string]agency.Contract
	logger            logrus.FieldLogger
}

func NewSerialExecutor(f1 agency.ApplyTxFunc, f2 agency.RegisterContractFunc, logger logrus.FieldLogger) agency.TxsExecutor {
	return &SerialExecutor{
		applyTxFunc:   f1,
		boltContracts: f2(),
		logger:        logger,
	}
}

func init() {
	agency.RegisterExecutorConstructor("serial", NewSerialExecutor)
}

func (se *SerialExecutor) ApplyTransactions(txs []pb.Transaction, invalidTxs map[int]agency.InvalidReason) []*pb.Receipt {
	se.interchainCounter = make(map[string][]*pb.VerifiedIndex)
	se.normalTxs = make([]*types.Hash, 0)
	receipts := make([]*pb.Receipt, 0, len(txs))

	for i, tx := range txs {
		receipts = append(receipts, se.applyTxFunc(i, tx, invalidTxs[i], nil))
	}

	se.logger.Debugf("serial executor executed %d txs", len(txs))

	return receipts
}

func (se *SerialExecutor) GetBoltContracts() map[string]agency.Contract {
	return se.boltContracts
}

func (se *SerialExecutor) AddNormalTx(hash *types.Hash) {
	se.normalTxs = append(se.normalTxs, hash)
}

func (se *SerialExecutor) GetNormalTxs() []*types.Hash {
	return se.normalTxs
}

func (se *SerialExecutor) AddInterchainCounter(to string, index *pb.VerifiedIndex) {
	se.interchainCounter[to] = append(se.interchainCounter[to], index)
}

func (se *SerialExecutor) GetInterchainCounter() map[string][]*pb.VerifiedIndex {
	return se.interchainCounter
}

func (se *SerialExecutor) GetDescription() string {
	return "serial executor"
}
