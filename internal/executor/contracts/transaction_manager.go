package contracts

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/meshplus/bitxhub-model/constant"

	"github.com/looplab/fsm"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/pb"
)

const (
	PREFIX         = "tx"
	TIMEOUT_PREFIX = "timeout"
)

type TransactionManager struct {
	boltvm.Stub
	fsm *fsm.FSM
}

type TransactionInfo struct {
	GlobalState  pb.TransactionStatus
	Height       uint64
	ChildTxInfo  map[string]pb.TransactionStatus
	ChildTxCount uint64
}

type StatusChange struct {
	PrevStatus   pb.TransactionStatus
	CurStatus    pb.TransactionStatus
	OtherIBTPIDs []string
}

func (c *StatusChange) NotifyFlags() (bool, bool) {
	if c.CurStatus == c.PrevStatus {
		return false, false
	}

	switch c.CurStatus {
	case pb.TransactionStatus_BEGIN:
		return false, true
	case pb.TransactionStatus_BEGIN_FAILURE:
		return true, true
	case pb.TransactionStatus_BEGIN_ROLLBACK:
		return true, true
	case pb.TransactionStatus_SUCCESS:
		return true, false
	case pb.TransactionStatus_FAILURE:
		if c.PrevStatus == pb.TransactionStatus_BEGIN {
			return true, false
		}
		return false, false
	case pb.TransactionStatus_ROLLBACK:
		return false, false
	}

	return false, false
}

type TransactionEvent string

func (e TransactionEvent) String() string {
	return string(e)
}

const (
	TransactionEvent_BEGIN         TransactionEvent = "begin"
	TransactionEvent_BEGIN_FAILURE TransactionEvent = "begin_failure"
	TransactionEvent_TIMEOUT       TransactionEvent = "timeout"
	TransactionEvent_FAILURE       TransactionEvent = "failure"
	TransactionEvent_SUCCESS       TransactionEvent = "success"
	TransactionEvent_ROLLBACK      TransactionEvent = "rollback"
	TransactionState_INIT                           = "init"
)

var receipt2EventM = map[int32]TransactionEvent{
	int32(pb.IBTP_RECEIPT_FAILURE):  TransactionEvent_FAILURE,
	int32(pb.IBTP_RECEIPT_SUCCESS):  TransactionEvent_SUCCESS,
	int32(pb.IBTP_RECEIPT_ROLLBACK): TransactionEvent_ROLLBACK,
}

func (t *TransactionManager) BeginMultiTXs(globalID, ibtpID string, timeoutHeight uint64, isFailed bool, count uint64) *boltvm.Response {
	if bxhErr := t.checkCurrentCaller(); bxhErr != nil {
		return boltvm.Error(bxhErr.Code, string(bxhErr.Msg))
	}

	change := StatusChange{PrevStatus: -1}
	txInfo := TransactionInfo{}
	if ok := t.GetObject(GlobalTxInfoKey(globalID), &txInfo); !ok {
		txInfo = TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN,
			Height:       t.GetCurrentHeight() + timeoutHeight,
			ChildTxInfo:  map[string]pb.TransactionStatus{ibtpID: pb.TransactionStatus_BEGIN},
			ChildTxCount: count,
		}

		if timeoutHeight == 0 || timeoutHeight >= math.MaxUint64-t.GetCurrentHeight() {
			txInfo.Height = math.MaxUint64
		}
		if isFailed {
			txInfo.ChildTxInfo[ibtpID] = pb.TransactionStatus_BEGIN_FAILURE
			txInfo.GlobalState = pb.TransactionStatus_BEGIN_FAILURE
		} else {
			t.addToTimeoutList(txInfo.Height, globalID)
		}

		t.AddObject(GlobalTxInfoKey(globalID), txInfo)
	} else {
		if _, ok := txInfo.ChildTxInfo[ibtpID]; ok {
			return boltvm.Error(boltvm.TransactionExistentChildTxCode, fmt.Sprintf(string(boltvm.TransactionExistentChildTxMsg), ibtpID, globalID))
		}

		if txInfo.GlobalState != pb.TransactionStatus_BEGIN {
			txInfo.ChildTxInfo[ibtpID] = txInfo.GlobalState
		} else {
			if isFailed {
				for key := range txInfo.ChildTxInfo {
					change.OtherIBTPIDs = append(change.OtherIBTPIDs, key)
					txInfo.ChildTxInfo[key] = pb.TransactionStatus_BEGIN_FAILURE
				}
				txInfo.ChildTxInfo[ibtpID] = pb.TransactionStatus_BEGIN_FAILURE
				txInfo.GlobalState = pb.TransactionStatus_BEGIN_FAILURE
				t.removeFromTimeoutList(txInfo.Height, globalID)
			} else {
				txInfo.ChildTxInfo[ibtpID] = pb.TransactionStatus_BEGIN
			}
		}
		t.SetObject(GlobalTxInfoKey(globalID), txInfo)
	}

	t.Set(ibtpID, []byte(globalID))

	change.CurStatus = txInfo.ChildTxInfo[ibtpID]
	data, err := json.Marshal(change)
	if err != nil {
		return boltvm.Error(boltvm.TransactionInternalErrCode, fmt.Sprintf(string(boltvm.TransactionInternalErrMsg), err.Error()))
	}

	return boltvm.Success(data)
}

func (t *TransactionManager) Begin(txId string, timeoutHeight uint64, isFailed bool) *boltvm.Response {
	if bxhErr := t.checkCurrentCaller(); bxhErr != nil {
		return boltvm.Error(bxhErr.Code, string(bxhErr.Msg))
	}

	record := pb.TransactionRecord{
		Status: pb.TransactionStatus_BEGIN,
		Height: t.GetCurrentHeight() + timeoutHeight,
	}

	if timeoutHeight == 0 || timeoutHeight >= math.MaxUint64-t.GetCurrentHeight() {
		record.Height = math.MaxUint64
	}

	if isFailed {
		record.Status = pb.TransactionStatus_BEGIN_FAILURE
	} else {
		//t.addToTimeoutList(record.Height, txId)
	}

	t.AddObject(TxInfoKey(txId), record)

	change := StatusChange{
		PrevStatus: -1,
		CurStatus:  record.Status,
	}

	data, err := json.Marshal(change)
	if err != nil {
		return boltvm.Error(boltvm.TransactionInternalErrCode, fmt.Sprintf(string(boltvm.TransactionInternalErrMsg), err.Error()))
	}

	return boltvm.Success(data)
}

func (t *TransactionManager) Report(txId string, result int32) *boltvm.Response {
	if bxhErr := t.checkCurrentCaller(); bxhErr != nil {
		return boltvm.Error(bxhErr.Code, string(bxhErr.Msg))
	}

	change := StatusChange{}
	var record pb.TransactionRecord
	ok := t.GetObject(TxInfoKey(txId), &record)
	if ok {
		change.PrevStatus = record.Status
		if err := t.setFSM(&record.Status, receipt2EventM[result]); err != nil {
			return boltvm.Error(boltvm.TransactionStateErrCode, fmt.Sprintf(string(boltvm.TransactionStateErrMsg), fmt.Sprintf("transaction %s with state %v get unexpected receipt %v", txId, record.Status, result)))
		}
		change.CurStatus = record.Status

		t.SetObject(TxInfoKey(txId), record)
		//t.removeFromTimeoutList(record.Height, txId)
	} else {
		ok, val := t.Get(txId)
		if !ok {
			return boltvm.Error(boltvm.TransactionNonexistentTxCode, fmt.Sprintf(string(boltvm.TransactionNonexistentTxMsg), txId))
		}

		globalId := string(val)
		txInfo := TransactionInfo{}
		if !t.GetObject(GlobalTxInfoKey(globalId), &txInfo) {
			return boltvm.Error(boltvm.TransactionNonexistentGlobalTxCode, fmt.Sprintf(string(boltvm.TransactionNonexistentGlobalTxMsg), globalId, txId))
		}

		_, ok = txInfo.ChildTxInfo[txId]
		if !ok {
			return boltvm.Error(boltvm.TransactionInternalErrCode, fmt.Sprintf(string(boltvm.TransactionInternalErrMsg), fmt.Sprintf("%s is not in transaction %s, %v", txId, globalId, txInfo)))
		}

		change.PrevStatus = txInfo.GlobalState
		if err := t.changeMultiTxStatus(globalId, &txInfo, txId, result); err != nil {
			return boltvm.Error(boltvm.TransactionStateErrCode, fmt.Sprintf(string(boltvm.TransactionStateErrMsg), err.Error()))
		}
		change.CurStatus = txInfo.GlobalState

		for key := range txInfo.ChildTxInfo {
			if key != txId {
				change.OtherIBTPIDs = append(change.OtherIBTPIDs, key)
			}
		}

		t.SetObject(GlobalTxInfoKey(globalId), txInfo)
	}

	data, err := json.Marshal(change)
	if err != nil {
		return boltvm.Error(boltvm.TransactionInternalErrCode, fmt.Sprintf(string(boltvm.TransactionInternalErrMsg), err.Error()))
	}

	return boltvm.Success(data)
}

func (t *TransactionManager) GetStatus(txId string) *boltvm.Response {
	var record pb.TransactionRecord
	ok := t.GetObject(TxInfoKey(txId), &record)
	if ok {
		status := record.Status
		return boltvm.Success([]byte(strconv.Itoa(int(status))))
	}

	txInfo := TransactionInfo{}
	ok = t.GetObject(GlobalTxInfoKey(txId), &txInfo)
	if ok {
		return boltvm.Success([]byte(strconv.Itoa(int(txInfo.GlobalState))))
	}

	ok, val := t.Get(txId)
	if !ok {
		return boltvm.Error(boltvm.TransactionNonexistentGlobalIdCode, fmt.Sprintf(string(boltvm.TransactionNonexistentGlobalIdMsg), txId))
	}

	globalId := string(val)
	txInfo = TransactionInfo{}
	if !t.GetObject(GlobalTxInfoKey(globalId), &txInfo) {
		return boltvm.Error(boltvm.TransactionNonexistentGlobalTxCode, fmt.Sprintf(string(boltvm.TransactionNonexistentGlobalTxMsg), globalId, txId))
	}

	return boltvm.Success([]byte(strconv.Itoa(int(txInfo.GlobalState))))
}

func (t *TransactionManager) setFSM(state *pb.TransactionStatus, event TransactionEvent) error {
	callbackFunc := func(event *fsm.Event) {
		*state = pb.TransactionStatus(pb.TransactionStatus_value[event.FSM.Current()])
	}

	t.fsm = fsm.NewFSM(
		state.String(),
		fsm.Events{
			{Name: TransactionEvent_BEGIN.String(), Src: []string{TransactionState_INIT}, Dst: pb.TransactionStatus_BEGIN.String()},
			{Name: TransactionEvent_BEGIN_FAILURE.String(), Src: []string{TransactionState_INIT, pb.TransactionStatus_BEGIN.String()}, Dst: pb.TransactionStatus_BEGIN_FAILURE.String()},
			{Name: TransactionEvent_TIMEOUT.String(), Src: []string{pb.TransactionStatus_BEGIN.String()}, Dst: pb.TransactionStatus_BEGIN_ROLLBACK.String()},
			{Name: TransactionEvent_SUCCESS.String(), Src: []string{pb.TransactionStatus_BEGIN.String()}, Dst: pb.TransactionStatus_SUCCESS.String()},
			{Name: TransactionEvent_FAILURE.String(), Src: []string{pb.TransactionStatus_BEGIN.String(), pb.TransactionStatus_BEGIN_FAILURE.String()}, Dst: pb.TransactionStatus_FAILURE.String()},
			{Name: TransactionEvent_ROLLBACK.String(), Src: []string{pb.TransactionStatus_BEGIN_ROLLBACK.String()}, Dst: pb.TransactionStatus_ROLLBACK.String()},
		},
		fsm.Callbacks{
			TransactionEvent_BEGIN.String():         callbackFunc,
			TransactionEvent_BEGIN_FAILURE.String(): callbackFunc,
			TransactionEvent_TIMEOUT.String():       callbackFunc,
			TransactionEvent_SUCCESS.String():       callbackFunc,
			TransactionEvent_FAILURE.String():       callbackFunc,
			TransactionEvent_ROLLBACK.String():      callbackFunc,
		},
	)

	return t.fsm.Event(event.String())
}

func (t *TransactionManager) addToTimeoutList(height uint64, txId string) {
	var timeoutList string
	var builder strings.Builder
	ok, val := t.Get(TimeoutKey(height))
	if !ok {
		timeoutList = txId
	} else {
		timeoutList = string(val)
		builder.WriteString(timeoutList)
		builder.WriteString(",")
		builder.WriteString(txId)
		timeoutList = builder.String()
	}
	t.Set(TimeoutKey(height), []byte(timeoutList))
}

func (t *TransactionManager) removeFromTimeoutList(height uint64, txId string) {
	ok, timeoutList := t.Get(TimeoutKey(height))
	if ok {
		list := strings.Split(string(timeoutList), ",")
		for index, value := range list {
			if value == txId {
				list = append(list[:index], list[index+1:]...)
			}
		}
		t.Set(TimeoutKey(height), []byte(strings.Join(list, ",")))
	}
}

func (t *TransactionManager) checkCurrentCaller() *boltvm.BxhError {
	if t.CurrentCaller() != constant.InterchainContractAddr.Address().String() {
		return boltvm.BError(boltvm.TransactionNoPermissionCode, fmt.Sprintf(string(boltvm.TransactionNoPermissionMsg), t.CurrentCaller()))
	}

	return nil
}

func (t *TransactionManager) changeMultiTxStatus(globalID string, txInfo *TransactionInfo, txId string, result int32) error {
	if txInfo.GlobalState == pb.TransactionStatus_BEGIN && result == int32(pb.IBTP_RECEIPT_FAILURE) {
		for childTx := range txInfo.ChildTxInfo {
			txInfo.ChildTxInfo[childTx] = pb.TransactionStatus_BEGIN_FAILURE
		}
		txInfo.ChildTxInfo[txId] = pb.TransactionStatus_FAILURE
		txInfo.GlobalState = pb.TransactionStatus_BEGIN_FAILURE

		t.removeFromTimeoutList(txInfo.Height, globalID)

		return nil
	} else {
		status := txInfo.ChildTxInfo[txId]
		if err := t.setFSM(&status, receipt2EventM[result]); err != nil {
			return fmt.Errorf("child tx %s with state %v get unexpected receipt %v", txId, status, result)
		}

		txInfo.ChildTxInfo[txId] = status

		if isMultiTxFinished(status, txInfo) {
			if err := t.setFSM(&txInfo.GlobalState, receipt2EventM[result]); err != nil {
				return fmt.Errorf("global tx of child tx %s with state %v get unexpected receipt %v", txId, status, result)
			}

			t.removeFromTimeoutList(txInfo.Height, globalID)

			return nil
		}
	}

	return nil
}

func isMultiTxFinished(childStatus pb.TransactionStatus, txInfo *TransactionInfo) bool {
	count := uint64(0)
	for _, res := range txInfo.ChildTxInfo {
		if res != childStatus {
			return false
		}
		count++
	}

	return count == txInfo.ChildTxCount
}

func TxInfoKey(id string) string {
	return fmt.Sprintf("%s-%s", PREFIX, id)
}

func GlobalTxInfoKey(id string) string {
	return fmt.Sprintf("global-%s-%s", PREFIX, id)
}

func TimeoutKey(height uint64) string {
	return fmt.Sprintf("%s-%d", TIMEOUT_PREFIX, height)
}
