package precheck

import (
	"context"
	"math/big"
	"time"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

const (
	basicGas = 21000
	to       = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
)

var mockDb = make(map[string]*big.Int)

func newMockPreCheckMgr() (*TxPreCheckMgr, *logrus.Entry, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.NewWithModule("precheck")
	getAccountBalance := func(address *types.Address) *big.Int {
		val, ok := mockDb[address.String()]
		if !ok {
			return big.NewInt(0)
		}
		return val
	}
	return NewTxPreCheckMgr(ctx, logger, getAccountBalance), logger, cancel
}

func cleanDb() {
	mockDb = make(map[string]*big.Int)
}

func setBalance(address string, balance *big.Int) {
	mockDb[address] = balance
}

func getBalance(address string) *big.Int {
	val, ok := mockDb[address]
	if !ok {
		return big.NewInt(0)
	}
	return val
}

func createLocalTxEvent(tx *types.Transaction) *order.UncheckedTxEvent {
	return &order.UncheckedTxEvent{
		EventType: order.LocalTxEvent,
		Event: &order.TxWithResp{
			Tx:     tx,
			RespCh: make(chan *order.TxResp),
		},
	}
}

func createRemoteTxEvent(txs []*types.Transaction) *order.UncheckedTxEvent {
	return &order.UncheckedTxEvent{
		EventType: order.RemoteTxEvent,
		Event:     txs,
	}
}

func generateBatchTx(s *types.Signer, size, illegalIndex int) ([]*types.Transaction, error) {
	toAddr := common.HexToAddress(to)
	txs := make([]*types.Transaction, size)
	for i := 0; i < size; i++ {
		if i != illegalIndex {
			tx, err := generateLegacyTx(s, &toAddr, uint64(i), nil, uint64(basicGas), 1, big.NewInt(0))
			if err != nil {
				return nil, err
			}
			txs[i] = tx
		}
	}
	// illegal tx
	tx, err := generateLegacyTx(s, nil, uint64(illegalIndex), nil, uint64(basicGas+1), 1, big.NewInt(0))
	if err != nil {
		return nil, err
	}
	txs[illegalIndex] = tx

	return txs, nil
}

func generateLegacyTx(s *types.Signer, to *common.Address, nonce uint64, data []byte, gasLimit, gasPrice uint64, value *big.Int) (*types.Transaction, error) {
	inner := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(int64(gasPrice)),
		Gas:      gasLimit,
		To:       to,
		Data:     data,
		Value:    value,
	}
	tx := &types.Transaction{
		Inner: inner,
		Time:  time.Now(),
	}

	if err := tx.SignByTxType(s.Sk); err != nil {
		return nil, err
	}
	return tx, nil
}

func generateDynamicFeeTx(s *types.Signer, to *common.Address, data []byte,
	gasLimit uint64, value, gasFeeCap, gasTipCap *big.Int) (*types.Transaction, error) {
	inner := &types.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     0,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        to,
		Data:      data,
		Value:     value,
	}
	tx := &types.Transaction{
		Inner: inner,
		Time:  time.Now(),
	}

	if err := tx.SignByTxType(s.Sk); err != nil {
		return nil, err
	}
	return tx, nil
}
