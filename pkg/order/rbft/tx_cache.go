package rbft

import (
	"time"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

const (
	DefaultTxCacheSize = 10000
	DefaultTxSetTick   = 100 * time.Millisecond
	DefaultTxSetSize   = 10
)

type TxCache struct {
	txSetC  chan []pb.Transaction
	recvTxC chan pb.Transaction
	TxRespC chan *TxWithResp
	close   chan bool

	txSet      []pb.Transaction
	logger     logrus.FieldLogger
	timerC     chan bool
	stopTimerC chan bool
	txSetTick  time.Duration
	txSetSize  uint64
}

type TxWithResp struct {
	Tx pb.Transaction
	Ch chan bool
}

func newTxCache(txSliceTimeout time.Duration, txSetSize uint64, logger logrus.FieldLogger) *TxCache {
	txCache := &TxCache{}
	txCache.recvTxC = make(chan pb.Transaction, DefaultTxCacheSize)
	txCache.txSetC = make(chan []pb.Transaction)
	txCache.TxRespC = make(chan *TxWithResp)
	txCache.close = make(chan bool)
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

func (tc *TxCache) listenEvent() {
	for {
		select {
		case <-tc.close:
			tc.logger.Info("Transaction cache stopped!")
			return

		case tx := <-tc.recvTxC:
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
	tc.txSetC <- dst
	tc.txSet = make([]pb.Transaction, 0)
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
