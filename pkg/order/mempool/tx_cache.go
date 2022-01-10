package mempool

import (
	"context"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

type TxWithResp struct {
	Tx pb.Transaction
	Ch chan bool
}

type TxCache struct {
	TxSetC  chan *pb.Transactions
	RecvTxC chan pb.Transaction
	TxRespC chan *TxWithResp

	txSet      []pb.Transaction
	timerC     chan bool
	stopTimerC chan bool
	txSetTick  time.Duration
	txSetSize  uint64
	logger     logrus.FieldLogger
}

func NewTxCache(txSliceTimeout time.Duration, txSetSize uint64, logger logrus.FieldLogger) *TxCache {
	txCache := &TxCache{}
	txCache.RecvTxC = make(chan pb.Transaction, DefaultTxCacheSize)
	txCache.TxSetC = make(chan *pb.Transactions)
	txCache.TxRespC = make(chan *TxWithResp)
	txCache.timerC = make(chan bool)
	txCache.stopTimerC = make(chan bool)
	txCache.txSet = make([]pb.Transaction, 0)
	txCache.logger = logger
	if txSliceTimeout == 0 {
		txCache.txSetTick = DefaultTxSetTick
	} else {
		txCache.txSetTick = txSliceTimeout
	}
	if txSetSize == 0 {
		txCache.txSetSize = DefaultTxSetSize
	} else {
		txCache.txSetSize = txSetSize
	}
	return txCache
}

func (tc *TxCache) ListenEvent(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			tc.logger.Infof("Exit transaction cache ListenEvent")
			return

		case tx := <-tc.RecvTxC:
			tc.appendTx(tx)

		case <-tc.timerC:
			tc.stopTxSetTimer()
			tc.postTxSet()
		}
	}
}

func (tc *TxCache) appendTx(tx pb.Transaction) {
	if tx == nil {
		tc.logger.Errorf("Transaction is nil")
		return
	}
	if len(tc.txSet) == 0 {
		tc.startTxSetTimer()
	}
	tc.txSet = append(tc.txSet, tx)
	if uint64(len(tc.txSet)) >= tc.txSetSize {
		tc.stopTxSetTimer()
		tc.postTxSet()
	}
}

func (tc *TxCache) postTxSet() {
	dst := make([]pb.Transaction, len(tc.txSet))
	copy(dst, tc.txSet)
	txSet := &pb.Transactions{
		Transactions: dst,
	}
	tc.TxSetC <- txSet
	tc.txSet = make([]pb.Transaction, 0)
}

func (tc *TxCache) IsFull() bool {
	return len(tc.RecvTxC) == DefaultTxCacheSize
}

func (tc *TxCache) startTxSetTimer() {
	go func() {
		timer := time.NewTimer(tc.txSetTick)
		select {
		case <-timer.C:
			tc.timerC <- true
		case <-tc.stopTimerC:
			return
		}
	}()
}

func (tc *TxCache) stopTxSetTimer() {
	close(tc.stopTimerC)
	tc.stopTimerC = make(chan bool)
}
