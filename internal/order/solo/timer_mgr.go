package solo

import (
	"strconv"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/sirupsen/logrus"
)

const (
	Batch     = "Batch"
	NoTxBatch = "NoTxBatch"
)

type batchTimeoutEvent string

// singleTimer manages timer with the same timer name, which, we allow different timer with the same timer name, such as:
// we allow several request timers at the same time, each timer started after received a new request batch
type singleTimer struct {
	timerName string             // the unique timer name
	timeout   time.Duration      // default timeout of this timer
	isActive  cmap.ConcurrentMap // track all the timers with this timerName if it is active now
}

func (tt *singleTimer) clear() {
	tt.isActive.Clear()
}

func (tt *singleTimer) isTimerActive() bool {
	return tt.isActive.Count() > 0
}

// timerManager manages consensus used timers.
type timerManager struct {
	timersM   map[string]*singleTimer
	eventChan chan<- consensusEvent
	logger    logrus.FieldLogger
}

// NewTimerManager news a timer with default timeout.
func NewTimerManager(ch chan consensusEvent, logger logrus.FieldLogger) *timerManager {
	return &timerManager{
		timersM:   make(map[string]*singleTimer),
		logger:    logger,
		eventChan: ch,
	}
}

func (tm *timerManager) newTimer(name string, d time.Duration) {
	if d == 0 {
		switch name {
		case Batch:
			d = 500 * time.Millisecond
		case NoTxBatch:
			d = 2 * time.Second
		}
	}
	tm.timersM[name] = &singleTimer{
		timerName: name,
		isActive:  cmap.New(),
		timeout:   d,
	}
}

// Stop stops all timers managed by timerManager
func (tm *timerManager) Stop() {
	for timerName := range tm.timersM {
		tm.stopTimer(timerName)
	}
}

// stopTimer stops all timers with the same timerName.
func (tm *timerManager) stopTimer(timerName string) {
	if !tm.containsTimer(timerName) {
		return
	}

	tm.timersM[timerName].clear()
}

// containsTimer returns true if there exists a timer named timerName
func (tm *timerManager) containsTimer(timerName string) bool {
	if t, ok := tm.timersM[timerName]; ok {
		return t.isTimerActive()
	}
	return false
}

// startTimer starts the timer with the given name and default timeout, then sets the event which will be triggered
// after this timeout
func (tm *timerManager) startTimer(name string) {
	tm.stopTimer(name)

	event := batchTimeoutEvent(name)

	timestamp := time.Now().UnixNano()
	key := strconv.FormatInt(timestamp, 10)
	tm.timersM[name].isActive.Set(key, true)
	time.AfterFunc(tm.timersM[name].timeout, func() {
		if tm.timersM[name].isActive.Has(key) {
			tm.eventChan <- event
		}
	})
}

func (tm *timerManager) isTimerActive(name string) bool {
	if t, ok := tm.timersM[name]; ok {
		return t.isTimerActive()
	}
	return false
}
