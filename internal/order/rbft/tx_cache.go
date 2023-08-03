package rbft

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types"
)

const (
	DefaultTxCacheSize = 10000
	DefaultTxSetTick   = 100 * time.Millisecond
	DefaultTxSetSize   = 10
)

type TxCache struct {
	txSetC  chan []*types.Transaction
	recvTxC chan *types.Transaction
	TxRespC chan *TxWithResp
	close   chan bool

	txSet      []*types.Transaction
	logger     logrus.FieldLogger
	timerC     chan bool
	stopTimerC chan bool
	txSetTick  time.Duration
	txSetSize  uint64
}

type TxWithResp struct {
	Tx *types.Transaction
	Ch chan bool
}

func newTxCache(txSliceTimeout time.Duration, txSetSize uint64, logger logrus.FieldLogger) *TxCache {
	txCache := &TxCache{}
	txCache.recvTxC = make(chan *types.Transaction, DefaultTxCacheSize)
	txCache.txSetC = make(chan []*types.Transaction)
	txCache.TxRespC = make(chan *TxWithResp)
	txCache.close = make(chan bool)
	txCache.timerC = make(chan bool)
	txCache.stopTimerC = make(chan bool)
	txCache.txSet = make([]*types.Transaction, 0)
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

func (tc *TxCache) appendTx(tx *types.Transaction) {
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
	dst := make([]*types.Transaction, len(tc.txSet))
	copy(dst, tc.txSet)
	tc.txSetC <- dst
	tc.txSet = make([]*types.Transaction, 0)
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
