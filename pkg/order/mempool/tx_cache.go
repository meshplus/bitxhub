package mempool

import (
	"time"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/loggers"

	"github.com/sirupsen/logrus"
)

type TxCache struct {
	recvTxC chan *pb.Transaction
	txSetC  chan *TxSlice
	txSet   []*pb.Transaction
	logger  logrus.FieldLogger

	timerC     chan bool
	stopTimerC chan bool
	close      chan bool
	txSetTick  time.Duration
}

func newTxCache(txSliceTimeout time.Duration) *TxCache {
	txCache := &TxCache{}
	txCache.recvTxC = make(chan *pb.Transaction, DefaultTxCacheSize)
	txCache.close = make(chan bool)
	txCache.txSetC = make(chan *TxSlice)
	txCache.timerC = make(chan bool)
	txCache.stopTimerC = make(chan bool)
	txCache.txSet = make([]*pb.Transaction, 0)
	txCache.logger = loggers.Logger(loggers.Order)
	if txSliceTimeout == 0 {
		txCache.txSetTick = DefaultTxSetTick
	} else {
		txCache.txSetTick = txSliceTimeout
	}
	return txCache
}

func (tc *TxCache) listenEvent() {
	for {
		select {
		case <-tc.close:
			tc.logger.Info("Exit transaction cache")

		case tx := <-tc.recvTxC:
			tc.appendTx(tx)

		case <-tc.timerC:
			tc.stopTxSetTimer()
			tc.postTxSet()
		}
	}
}

func (tc *TxCache) appendTx(tx *pb.Transaction) {
	if tx == nil {
		tc.logger.Errorf("Transaction is nil")
		return
	}
	if len(tc.txSet) == 0 {
		tc.startTxSetTimer()
	}
	tc.txSet = append(tc.txSet, tx)
	if len(tc.txSet) >= DefaultTxSetSize {
		tc.stopTxSetTimer()
		tc.postTxSet()
	}
}

func (tc *TxCache) postTxSet() {
	dst := make([]*pb.Transaction, len(tc.txSet))
	copy(dst, tc.txSet)
	txSet := &TxSlice{
		TxList: dst,
	}
	tc.txSetC <- txSet
	tc.txSet = make([]*pb.Transaction, 0)
}

func (tc *TxCache) IsFull() bool {
	return len(tc.recvTxC) == DefaultTxCacheSize
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
