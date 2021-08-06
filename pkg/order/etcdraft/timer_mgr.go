package etcdraft

import (
	cmap "github.com/orcaman/concurrent-map"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type Timer struct {
	logger        logrus.FieldLogger
	timeout       time.Duration      // default timeout of this timer
	isActive      cmap.ConcurrentMap // track all the timers with this timerName if it is active now
	timeoutEventC chan bool
}

// NewTimer news a timer with default timeout.
func NewTimer(d time.Duration, logger logrus.FieldLogger) *Timer {
	return &Timer{
		timeout:       d,
		isActive:      cmap.New(),
		timeoutEventC: make(chan bool),
		logger:        logger,
	}
}

func (timer *Timer) IsTimerActive() bool {
	return !timer.isActive.IsEmpty()
}

// TODO (YH): add restartTimer???
// StartTimer starts the batch timer and reset the TimerActive to true.
func (timer *Timer) StartTimer() {
	// stop old timer
	timer.StopTimer()
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

// StopTimer stops the batch timer and reset the TimerActive to false.
func (timer *Timer) StopTimer() {
	if timer.isActive.IsEmpty() {
		return
	}
	timer.logger.Debugf("Leader stop batch timer")
	timer.isActive = cmap.New()
}

func (timer *Timer) TimeoutEvent() chan bool {
	return timer.timeoutEventC
}
