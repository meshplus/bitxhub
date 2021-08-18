package etcdraft

import (
	cmap "github.com/orcaman/concurrent-map"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type BatchTimer struct {
	logger        logrus.FieldLogger
	timeout       time.Duration      // default timeout of this timer
	isActive      cmap.ConcurrentMap // track all the timers with this timerName if it is active now
	timeoutEventC chan bool
}

// NewTimer news a timer with default timeout.
func NewTimer(d time.Duration, logger logrus.FieldLogger) *BatchTimer {
	return &BatchTimer{
		timeout:       d,
		isActive:      cmap.New(),
		timeoutEventC: make(chan bool),
		logger:        logger,
	}
}

func (timer *BatchTimer) IsBatchTimerActive() bool {
	return !timer.isActive.IsEmpty()
}

// StartBatchTimer starts the batch timer and reset the batchTimerActive to true.
// TODO (YH): add restartTimer???
func (timer *BatchTimer) StartBatchTimer() {
	// stop old timer
	timer.StopBatchTimer()
	timer.logger.Debug("Leader start batch timer")
	timestamp := time.Now().UnixNano()
	key := strconv.FormatInt(timestamp, 10)
	timer.isActive.Set(key, true)

	time.AfterFunc(timer.timeout, func() {
		if timer.isActive.Has(key) {
			timer.timeoutEventC <- true
		}
	})
}

// StopBatchTimer stops the batch timer and reset the batchTimerActive to false.
func (timer *BatchTimer) StopBatchTimer() {
	if timer.isActive.IsEmpty() {
		return
	}
	timer.logger.Debugf("Leader stop batch timer")
	timer.isActive = cmap.New()
}

func (timer *BatchTimer) BatchTimeoutEvent() chan bool {
	return timer.timeoutEventC
}
