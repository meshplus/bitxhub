package contracts

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

func TestTransactionManager_BeginMultiTXs(t *testing.T) {
	id0 := "1356:chain0:service0-1356:chain1:service1-1"
	id1 := "1356:chain0:service0-1356:chain2:service2-1"
	globalId := "globalId"

	setup := func(t *testing.T) (*mock_stub.MockStub, *TransactionManager) {
		mockCtl := gomock.NewController(t)
		mockStub := mock_stub.NewMockStub(mockCtl)
		mockStub.EXPECT().GetCurrentHeight().Return(uint64(100)).AnyTimes()
		logger := log.NewWithModule("transaction_contract")
		mockStub.EXPECT().Logger().Return(logger).AnyTimes()
		im := &TransactionManager{Stub: mockStub}
		return mockStub, im
	}
	// check current caller
	t.Run("case1", func(t *testing.T) {
		mockStub, im := setup(t)

		mockStub.EXPECT().CurrentCaller().Return(constant.TransactionMgrContractAddr.Address().String()).MaxTimes(2)
		res := im.BeginMultiTXs(globalId, id0, 10, false, 2)
		assert.False(t, res.Ok)
		assert.Contains(t, string(res.Result), "current caller 0x000000000000000000000000000000000000000F is not allowed")
	})

	// add tx
	t.Run("case2", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, TransactionInfo{}).Return(false).MaxTimes(1)
		mockStub.EXPECT().Get(TimeoutKey(uint64(110))).Return(false, nil).MaxTimes(1)
		mockStub.EXPECT().Set(TimeoutKey(uint64(110)), []byte(globalId)).MaxTimes(1)

		mockStub.EXPECT().Set(id0, []byte(globalId)).MaxTimes(1)
		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().AddObject(GlobalTxInfoKey(globalId), txInfo).MaxTimes(1)

		res := im.BeginMultiTXs(globalId, id0, 10, false, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.CurStatus)
		assert.Equal(t, 1, len(statusChange.ChildIBTPIDs))
	})

	// add failed tx
	t.Run("case3", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, TransactionInfo{}).Return(false).MaxTimes(1)
		mockStub.EXPECT().Set(id0, []byte(globalId)).MaxTimes(1)
		mockStub.EXPECT().AddObject(GlobalTxInfoKey(globalId), txInfo).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id0, 10, true, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN_FAILURE, statusChange.CurStatus)
		assert.Equal(t, 1, len(statusChange.ChildIBTPIDs))
	})

	// add existing child tx
	t.Run("case4", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id0, 10, false, 2)
		assert.False(t, res.Ok)
		assert.Contains(t, string(res.Result), fmt.Sprintf("child tx %s of global tx %s exists", id0, globalId))
	})

	// add child tx when GlobalState is BEGIN_FAILURE
	t.Run("case5", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
		expTxInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE, id1: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalId), expTxInfo).MaxTimes(1)
		mockStub.EXPECT().Set(id1, []byte(globalId)).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id1, 10, false, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN_FAILURE, statusChange.CurStatus)
		assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
	})

	// add child tx when GlobalState is BEGIN
	t.Run("case6", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN},
			Height:       110,
			ChildTxCount: 2,
		}
		expTxInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN, id1: pb.TransactionStatus_BEGIN},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
		mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalId), expTxInfo).MaxTimes(1)
		mockStub.EXPECT().Set(id1, []byte(globalId)).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id1, 10, false, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.CurStatus)
		assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
	})

	// add failed child tx when GlobalState is BEGIN
	t.Run("case7", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN},
			Height:       110,
			ChildTxCount: 2,
		}
		expTxInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE, id1: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
		mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalId), expTxInfo)
		mockStub.EXPECT().Get(TimeoutKey(uint64(110))).Return(false, nil).MaxTimes(1)
		mockStub.EXPECT().Set(TimeoutKey(uint64(110)), []byte(globalId)).MaxTimes(1)
		mockStub.EXPECT().Set(id1, []byte(globalId)).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id1, 10, true, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN_FAILURE, statusChange.CurStatus)
		assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
	})

	// add failed child tx when GlobalState is BEGIN, last child tx is Success
	t.Run("case8", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_SUCCESS},
			Height:       110,
			ChildTxCount: 2,
		}
		expTxInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE, id1: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
		mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalId), expTxInfo)
		mockStub.EXPECT().Get(TimeoutKey(uint64(110))).Return(false, nil).MaxTimes(1)
		mockStub.EXPECT().Set(TimeoutKey(uint64(110)), []byte(globalId)).MaxTimes(1)
		mockStub.EXPECT().Set(id1, []byte(globalId)).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id1, 10, true, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN_FAILURE, statusChange.CurStatus)
		assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
		assert.Equal(t, 1, len(statusChange.NotifySrcIBTPIDs))
		assert.Equal(t, 1, len(statusChange.NotifyDstIBTPIDs))
		assert.Equal(t, id0, statusChange.NotifySrcIBTPIDs[0])
		assert.Equal(t, id0, statusChange.NotifyDstIBTPIDs[0])
	})

	// add failed child tx when GlobalState is BEGIN_FAILURE
	t.Run("case9", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		expTxInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE, id1: pb.TransactionStatus_BEGIN_FAILURE},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
		mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalId), expTxInfo)
		mockStub.EXPECT().Set(id1, []byte(globalId)).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id1, 10, true, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN_FAILURE, statusChange.CurStatus)
		assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
		assert.Equal(t, 0, len(statusChange.NotifySrcIBTPIDs))
		assert.Equal(t, 0, len(statusChange.NotifyDstIBTPIDs))
	})

	// add child tx when GlobalState is BEGIN_ROLLBACK
	t.Run("case10", func(t *testing.T) {
		mockStub, im := setup(t)
		mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).MaxTimes(1)

		txInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_ROLLBACK,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_ROLLBACK},
			Height:       110,
			ChildTxCount: 2,
		}
		expTxInfo := TransactionInfo{
			GlobalState:  pb.TransactionStatus_BEGIN_ROLLBACK,
			ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_ROLLBACK, id1: pb.TransactionStatus_BEGIN_ROLLBACK},
			Height:       110,
			ChildTxCount: 2,
		}
		mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalId), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
		mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalId), expTxInfo)
		mockStub.EXPECT().Set(id1, []byte(globalId)).MaxTimes(1)
		res := im.BeginMultiTXs(globalId, id1, 10, false, 2)
		assert.True(t, res.Ok)
		statusChange := pb.StatusChange{}
		err := statusChange.Unmarshal(res.Result)
		assert.Nil(t, err)
		assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
		assert.Equal(t, pb.TransactionStatus_BEGIN_ROLLBACK, statusChange.CurStatus)
		assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
		assert.Equal(t, 0, len(statusChange.NotifySrcIBTPIDs))
		assert.Equal(t, 0, len(statusChange.NotifyDstIBTPIDs))
	})
}

func TestTransactionManager_Begin(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id := "1356:chain0:service0-1356:chain1:service1-1"
	mockStub.EXPECT().GetCurrentHeight().Return(uint64(100)).AnyTimes()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Add(gomock.Any(), gomock.Any()).AnyTimes()
	im := &TransactionManager{Stub: mockStub}

	mockStub.EXPECT().CurrentCaller().Return(constant.TransactionMgrContractAddr.Address().String()).MaxTimes(2)
	res := im.Begin(id, 0, false)
	assert.False(t, res.Ok)

	mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).AnyTimes()
	res = im.Begin(id, 10, false)
	assert.True(t, res.Ok)
	statusChange := pb.StatusChange{}
	err := statusChange.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
	assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.CurStatus)
	assert.Equal(t, 0, len(statusChange.ChildIBTPIDs))

	res = im.Begin(id, 10, true)
	assert.True(t, res.Ok)
	statusChange = pb.StatusChange{}
	err = statusChange.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, pb.TransactionStatus(-1), statusChange.PrevStatus)
	assert.Equal(t, pb.TransactionStatus_BEGIN_FAILURE, statusChange.CurStatus)
	assert.Equal(t, 0, len(statusChange.ChildIBTPIDs))
}

func TestTransactionManager_Report(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	logger := log.NewWithModule("transaction_contract")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	id := "1356:chain0:service0-1356:chain1:service1-1"
	recBegin := pb.TransactionRecord{
		Height: 100,
		Status: pb.TransactionStatus_BEGIN,
	}
	recBeginData, err := recBegin.Marshal()
	assert.Nil(t, err)
	recSuccess := pb.TransactionRecord{
		Height: 100,
		Status: pb.TransactionStatus_SUCCESS,
	}
	recSuccessData, err := recSuccess.Marshal()
	assert.Nil(t, err)
	recFailure := pb.TransactionRecord{
		Height: 100,
		Status: pb.TransactionStatus_FAILURE,
	}
	recFailureData, err := recFailure.Marshal()
	assert.Nil(t, err)

	im := &TransactionManager{Stub: mockStub}

	mockStub.EXPECT().CurrentCaller().Return(constant.TransactionMgrContractAddr.Address().String()).MaxTimes(2)
	res := im.Report(id, 0)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), "current caller 0x000000000000000000000000000000000000000F is not allowed")

	mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).AnyTimes()

	mockStub.EXPECT().Get(TxInfoKey(id)).Return(true, recSuccessData).MaxTimes(1)
	res = im.Report(id, 0)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("transaction %s with state %v get unexpected receipt %v", id, recSuccess.Status, 0))

	mockStub.EXPECT().Get(TxInfoKey(id)).Return(true, recBeginData).MaxTimes(1)
	mockStub.EXPECT().Set(TxInfoKey(id), recSuccessData).MaxTimes(1)
	res = im.Report(id, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.True(t, res.Ok)
	statusChange := pb.StatusChange{}
	err = statusChange.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.PrevStatus)
	assert.Equal(t, pb.TransactionStatus_SUCCESS, statusChange.CurStatus)
	assert.Equal(t, 0, len(statusChange.ChildIBTPIDs))

	mockStub.EXPECT().Get(TxInfoKey(id)).Return(true, recBeginData).MaxTimes(1)
	mockStub.EXPECT().Set(TxInfoKey(id), recFailureData).MaxTimes(1)
	res = im.Report(id, int32(pb.IBTP_RECEIPT_FAILURE))
	assert.True(t, res.Ok)
	statusChange = pb.StatusChange{}
	err = statusChange.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.PrevStatus)
	assert.Equal(t, pb.TransactionStatus_FAILURE, statusChange.CurStatus)
	assert.Equal(t, 0, len(statusChange.ChildIBTPIDs))

	// multi IBTP report test
	id0 := "1356:chain0:service0"
	id1 := "1356:chain1:service1"
	id2 := "1356:chain2:service2"
	globalID := "globalID"

	mockStub.EXPECT().Get(TxInfoKey(id0)).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Get(TxInfoKey(id2)).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Get(id0).Return(false, nil)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("transaction id %s does not exist", id0))

	mockStub.EXPECT().Get(gomock.Not(GlobalTxInfoKey(globalID))).Return(true, []byte(globalID)).AnyTimes()
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).Return(false).MaxTimes(1)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("global tx %s of child tx %s does not exist", globalID, id0))

	txInfo := TransactionInfo{
		GlobalState:  pb.TransactionStatus_BEGIN,
		Height:       110,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_SUCCESS, id1: pb.TransactionStatus_BEGIN},
		ChildTxCount: 2,
	}
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.Report(id2, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("%s is not in transaction %s, %v", id2, globalID, txInfo))

	mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalID), txInfo).MaxTimes(1)
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.True(t, res.Ok)

	txInfo.GlobalState = pb.TransactionStatus_SUCCESS
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("global tx of child tx %s with state %v get unexpected receipt %v", id0, txInfo.GlobalState, int32(pb.IBTP_RECEIPT_SUCCESS)))

	txInfo = TransactionInfo{
		GlobalState:  pb.TransactionStatus_BEGIN,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN, id1: pb.TransactionStatus_SUCCESS},
		ChildTxCount: txInfo.ChildTxCount,
	}
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	expTxInfo := TransactionInfo{
		GlobalState:  pb.TransactionStatus_SUCCESS,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_SUCCESS, id1: pb.TransactionStatus_SUCCESS},
		ChildTxCount: txInfo.ChildTxCount,
	}
	mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalID), expTxInfo).MaxTimes(1)
	mockStub.EXPECT().Get(TimeoutKey(txInfo.Height)).Return(true, []byte(globalID)).MaxTimes(1)
	mockStub.EXPECT().Set(TimeoutKey(txInfo.Height), []byte{}).MaxTimes(1)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.True(t, res.Ok)
	statusChange = pb.StatusChange{}
	err = statusChange.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.PrevStatus)
	assert.Equal(t, pb.TransactionStatus_SUCCESS, statusChange.CurStatus)
	assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
	assert.Equal(t, 1, len(statusChange.NotifySrcIBTPIDs))

	txInfo = TransactionInfo{
		GlobalState:  pb.TransactionStatus_BEGIN,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN, id1: pb.TransactionStatus_BEGIN},
		ChildTxCount: txInfo.ChildTxCount,
	}
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	expTxInfo = TransactionInfo{
		GlobalState:  pb.TransactionStatus_BEGIN,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_SUCCESS, id1: pb.TransactionStatus_BEGIN},
		ChildTxCount: txInfo.ChildTxCount,
	}
	mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalID), expTxInfo).MaxTimes(1)
	mockStub.EXPECT().Get(TimeoutKey(txInfo.Height)).Return(true, []byte(globalID)).MaxTimes(1)
	mockStub.EXPECT().Set(TimeoutKey(txInfo.Height), []byte{}).MaxTimes(1)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.True(t, res.Ok)
	statusChange = pb.StatusChange{}
	err = statusChange.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.PrevStatus)
	assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.CurStatus)
	assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))

	txInfo = TransactionInfo{
		GlobalState:  pb.TransactionStatus_BEGIN,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN, id1: pb.TransactionStatus_BEGIN},
		ChildTxCount: txInfo.ChildTxCount,
	}
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	expTxInfo = TransactionInfo{
		GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_FAILURE, id1: pb.TransactionStatus_BEGIN_FAILURE},
		ChildTxCount: txInfo.ChildTxCount,
	}
	mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalID), expTxInfo).MaxTimes(1)
	mockStub.EXPECT().Get(TimeoutKey(txInfo.Height)).Return(true, []byte(globalID)).MaxTimes(1)
	mockStub.EXPECT().Set(TimeoutKey(txInfo.Height), []byte{}).MaxTimes(1)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_FAILURE))
	assert.True(t, res.Ok)
	statusChange = pb.StatusChange{}
	err = statusChange.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, pb.TransactionStatus_BEGIN, statusChange.PrevStatus)
	assert.Equal(t, pb.TransactionStatus_BEGIN_FAILURE, statusChange.CurStatus)
	assert.Equal(t, 2, len(statusChange.ChildIBTPIDs))
	assert.Equal(t, 1, len(statusChange.NotifySrcIBTPIDs))
	assert.Equal(t, 0, len(statusChange.NotifyDstIBTPIDs))

	txInfo = TransactionInfo{
		GlobalState:  pb.TransactionStatus_BEGIN_FAILURE,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE, id1: pb.TransactionStatus_FAILURE},
		ChildTxCount: txInfo.ChildTxCount,
	}

	expTxInfo = TransactionInfo{
		GlobalState:  pb.TransactionStatus_FAILURE,
		Height:       txInfo.Height,
		ChildTxInfo:  map[string]pb.TransactionStatus{id0: pb.TransactionStatus_BEGIN_FAILURE, id1: pb.TransactionStatus_FAILURE},
		ChildTxCount: txInfo.ChildTxCount,
	}
	mockStub.EXPECT().SetObject(GlobalTxInfoKey(globalID), expTxInfo).MaxTimes(1)
	mockStub.EXPECT().GetObject(GlobalTxInfoKey(globalID), gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	mockStub.EXPECT().Get(TimeoutKey(txInfo.Height)).Return(true, []byte(globalID)).MaxTimes(1)
	mockStub.EXPECT().Set(TimeoutKey(txInfo.Height), []byte{}).MaxTimes(1)
	res = im.Report(id0, int32(pb.IBTP_RECEIPT_SUCCESS))
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("child tx %s with state %v get unexpected receipt %v", id0, txInfo.GlobalState, int32(pb.IBTP_RECEIPT_SUCCESS)))
}

func TestTransactionManager_GetStatus(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id := "id"
	txInfoKey := fmt.Sprintf("%s-%s", PREFIX, id)
	globalInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, id)

	recSuccess := pb.TransactionRecord{
		Height: 100,
		Status: pb.TransactionStatus_SUCCESS,
	}
	recSuccessData, err := recSuccess.Marshal()
	assert.Nil(t, err)

	im := &TransactionManager{Stub: mockStub}

	mockStub.EXPECT().Get(txInfoKey).Return(true, recSuccessData).MaxTimes(1)
	res := im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "3", string(res.Result))

	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	mockStub.EXPECT().Get(txInfoKey).Return(false, nil).AnyTimes()
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "0", string(res.Result))

	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Get(id).Return(false, nil).MaxTimes(1)
	res = im.GetStatus(id)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("cannot get global id of child tx id %s", id))

	globalId := "globalId"
	globalIdInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, globalId)
	mockStub.EXPECT().Get(id).Return(true, []byte(globalId)).AnyTimes()
	mockStub.EXPECT().GetObject(globalIdInfoKey, gomock.Any()).Return(false).MaxTimes(1)
	res = im.GetStatus(id)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), fmt.Sprintf("global tx %s of child tx %s does not exist", globalId, id))

	mockStub.EXPECT().GetObject(globalIdInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "0", string(res.Result))
}
