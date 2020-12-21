package etcdraft

import (
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type batchTimer struct {
	logger logrus.FieldLogger
	seqNo            uint64
	timeout       time.Duration      // default timeout of this timer
	isActive      cmap.ConcurrentMap // track all the timers with this timerName if it is active now
	timeoutEventC chan bool
}

// newTimer news a timer with default timeout.
func newTimer(d time.Duration, logger logrus.FieldLogger) *batchTimer {
	return &batchTimer{
		timeout:       d,
		isActive:      cmap.New(),
		timeoutEventC: make(chan bool),
		logger: logger,
	}
}

func (timer *batchTimer) isBatchTimerActive() bool {
	return !timer.isActive.IsEmpty()
}

// TODO (YH): add restartTimer???
// startBatchTimer starts the batch timer and reset the batchTimerActive to true.
func (timer *batchTimer) startBatchTimer() {
	// stop old timer
	timer.stopBatchTimer()
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

// stopBatchTimer stops the batch timer and reset the batchTimerActive to false.
func (timer *batchTimer) stopBatchTimer() {
	if timer.isActive.IsEmpty() {
		return
	}
	timer.logger.Debugf("Leader stop batch timer")
	timer.isActive = cmap.New()
}

func msgToConsensusPbMsg(data []byte, tyr raftproto.RaftMessage_Type, replicaID uint64) *pb.Message {
	rm := &raftproto.RaftMessage{
		Type:   tyr,
		FromId: replicaID,
		Data:   data,
	}
	cmData, err := rm.Marshal()
	if err != nil {
		return nil
	}
	msg := &pb.Message{
		Type: pb.Message_CONSENSUS,
		Data: cmData,
	}
	return msg
}