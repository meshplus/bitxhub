package mempool

import (
	"time"

	"github.com/meshplus/bitxhub-model/pb"
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
	txSetSize uint64
}

func newTxCache(txSliceTimeout time.Duration, txSetSize uint64, logger logrus.FieldLogger) *TxCache {
	txCache := &TxCache{}
	txCache.recvTxC = make(chan *pb.Transaction, DefaultTxCacheSize)
	txCache.close = make(chan bool)
	txCache.txSetC = make(chan *TxSlice)
	txCache.timerC = make(chan bool)
	txCache.stopTimerC = make(chan bool)
	txCache.txSet = make([]*pb.Transaction, 0)
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

func (tc *TxCache) listenEvent() {
	for {
		select {
		case <-tc.close:
			tc.logger.Info("Exit transaction cache")
			return

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
	if uint64(len(tc.txSet)) >= tc.txSetSize {
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
