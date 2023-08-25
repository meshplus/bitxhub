package txcache

import (
	"time"

	"github.com/axiomesh/axiom/internal/order"
	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/types"
)

const (
	DefaultTxCacheSize = 10000
	DefaultTxSetTick   = 100 * time.Millisecond
	DefaultTxSetSize   = 10
)

type TxCache struct {
	TxSetC  chan []*types.Transaction
	RecvTxC chan *types.Transaction
	TxRespC chan *order.TxWithResp
	CloseC  chan bool

	txSet      []*types.Transaction
	logger     logrus.FieldLogger
	timerC     chan bool
	stopTimerC chan bool
	txSetTick  time.Duration
	txSetSize  uint64
}

func NewTxCache(txSliceTimeout time.Duration, txSetSize uint64, logger logrus.FieldLogger) *TxCache {
	txCache := &TxCache{}
	txCache.RecvTxC = make(chan *types.Transaction, DefaultTxCacheSize)
	txCache.TxSetC = make(chan []*types.Transaction)
	txCache.TxRespC = make(chan *order.TxWithResp)
	txCache.CloseC = make(chan bool)
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

func (tc *TxCache) ListenEvent() {
	for {
		select {
		case <-tc.CloseC:
			tc.logger.Info("Transaction cache stopped!")
			return

		case tx := <-tc.RecvTxC:
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
	tc.TxSetC <- dst
	tc.txSet = make([]*types.Transaction, 0)
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
