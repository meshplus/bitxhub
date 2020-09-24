package mempool

import (
	"encoding/hex"
	"errors"
	"fmt"
	cmap "github.com/orcaman/concurrent-map"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	raftproto "github.com/meshplus/bitxhub/pkg/order/etcdraft/proto"
)

func (mpi *mempoolImpl) getBatchSeqNo() uint64 {
	return atomic.LoadUint64(&mpi.batchSeqNo)
}

func (mpi *mempoolImpl) increaseBatchSeqNo() {
	atomic.AddUint64(&mpi.batchSeqNo, 1)
}

// getTxByTxPointer returns the tx stored in allTxs by given TxPointer.
func (mpi *mempoolImpl) getTxByTxPointer(txPointer orderedIndexKey) *pb.Transaction {
	if txnMap, ok := mpi.txStore.allTxs[txPointer.account]; ok {
		return txnMap.items[txPointer.nonce].tx
	}
	return nil
}

func (mpi *mempoolImpl) msgToConsensusPbMsg(data []byte, tyr raftproto.RaftMessage_Type) *pb.Message {
	rm := &raftproto.RaftMessage{
		Type:   tyr,
		FromId: mpi.localID,
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

func newSubscribe() *subscribeEvent {
	return &subscribeEvent{
		txForwardC:           make(chan *TxSlice),
		localMissingTxnEvent: make(chan *LocalMissingTxnEvent),
		fetchTxnRequestC:     make(chan *FetchTxnRequest),
		updateLeaderC:        make(chan uint64),
		fetchTxnResponseC:    make(chan *FetchTxnResponse),
		commitTxnC:           make(chan *raftproto.Ready),
		getBlockC:            make(chan *constructBatchEvent),
	}
}

// TODO (YH): restore commitNonce and pendingNonce from db.
func newNonceCache() *nonceCache {
	return &nonceCache{
		commitNonces:  make(map[string]uint64),
		pendingNonces: make(map[string]uint64),
	}
}

func hex2Hash(hash string) (types.Hash, error) {
	var (
		hubHash   types.Hash
		hashBytes []byte
		err       error
	)
	if hashBytes, err = hex.DecodeString(hash); err != nil {
		return types.Hash{}, err
	}
	if len(hashBytes) != types.HashLength {
		return types.Hash{}, errors.New("invalid tx hash")
	}
	copy(hubHash[:], hashBytes)
	return hubHash, nil
}

func (mpi *mempoolImpl) poolIsFull() bool {
	return atomic.LoadInt32(&mpi.txStore.poolSize) >= DefaultPoolSize
}

func (mpi *mempoolImpl) isLeader() bool {
	return mpi.leader == mpi.localID
}

func (mpi *mempoolImpl) isBatchTimerActive() bool {
	return !mpi.batchTimerMgr.isActive.IsEmpty()
}

// startBatchTimer starts the batch timer and reset the batchTimerActive to true.
func (mpi *mempoolImpl) startBatchTimer(reason string) {
	// stop old timer
	mpi.stopBatchTimer(StopReason3)
	mpi.logger.Debugf("Start batch timer, reason: %s",reason)
	timestamp := time.Now().UnixNano()
	key := strconv.FormatInt(timestamp, 10)
	mpi.batchTimerMgr.isActive.Set(key, true)

	time.AfterFunc(mpi.batchTimerMgr.timeout, func() {
		if mpi.batchTimerMgr.isActive.Has(key) {
			mpi.batchTimerMgr.timeoutEventC <- true
		}
	})
}

// stopBatchTimer stops the batch timer and reset the batchTimerActive to false.
func (mpi *mempoolImpl) stopBatchTimer(reason string) {
	if mpi.batchTimerMgr.isActive.IsEmpty() {
		return
	}
	mpi.logger.Debugf("Stop batch timer, reason: %s", reason)
	mpi.batchTimerMgr.isActive = cmap.New()
}

// newTimer news a timer with default timeout.
func newTimer(d time.Duration) *timerManager {
	return &timerManager{
		timeout:       d,
		isActive:      cmap.New(),
		timeoutEventC: make(chan bool),
	}
}

func getAccount(tx *pb.Transaction) (string, error) {
	payload := &pb.InvokePayload{}
	if err := payload.Unmarshal(tx.Data.Payload); err != nil {
		return "",fmt.Errorf("unmarshal invoke payload: %s", err.Error())
	}
	if payload.Method == IBTPMethod1 || payload.Method == IBTPMethod2 {
		ibtp := &pb.IBTP{}
		if err := ibtp.Unmarshal(payload.Args[0].Value); err != nil {
			return "", fmt.Errorf("unmarshal ibtp from tx :%w", err)
		}
		account := fmt.Sprintf("%s-%s",ibtp.From,ibtp.To)
		return account, nil
	}
	return tx.From.Hex(), nil
}