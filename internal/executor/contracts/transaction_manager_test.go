package contracts

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

func TestTransactionManager_Begin(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id := types.NewHash([]byte{0}).String()
	mockStub.EXPECT().GetCurrentHeight().Return(uint64(100)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	im := &TransactionManager{mockStub}

	var timeoutList []string
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, timeoutList).Return(true)
	res := im.Begin(id, 10)
	assert.True(t, res.Ok)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, timeoutList).Return(false)
	res = im.Begin(id, 10)
	assert.True(t, res.Ok)
}

func TestTransactionManager_Report(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := "id0"
	id1 := "id1"
	txInfoKey := fmt.Sprintf("%s-%s", PREFIX, id0)
	recBegin := pb.TransactionRecord{
		Height: 100,
		Status: pb.TransactionStatus_BEGIN,
	}
	recSuccess := pb.TransactionRecord{
		Height: 100,
		Status: pb.TransactionStatus_SUCCESS,
	}
	recFailure := pb.TransactionRecord{
		Height: 100,
		Status: pb.TransactionStatus_FAILURE,
	}

	im := &TransactionManager{mockStub}

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, recSuccess).Return(true)
	res := im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction with Id %s is finished", id0), string(res.Result))

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, recBegin).Return(true)
	mockStub.EXPECT().SetObject(txInfoKey, recSuccess)
	mockStub.EXPECT().GetObject(TimeoutKey(100), gomock.Any()).SetArg(1, []string{id0, id1}).Return(true)
	mockStub.EXPECT().SetObject(TimeoutKey(100), []string{id1})
	res = im.Report(id0, 0)
	assert.True(t, res.Ok)

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, recBegin).Return(true)
	mockStub.EXPECT().SetObject(txInfoKey, recFailure)
	mockStub.EXPECT().GetObject(TimeoutKey(100), gomock.Any()).SetArg(1, []string{id1}).Return(true)
	mockStub.EXPECT().SetObject(TimeoutKey(100), []string{id1})
	res = im.Report(id0, 1)
	assert.True(t, res.Ok)

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Get(id0).Return(false, nil)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("cannot get global id of child tx id %s", id0), string(res.Result))

	globalId := "globalId"
	globalInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, globalId)
	mockStub.EXPECT().Get(id0).Return(true, []byte(globalId)).AnyTimes()
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).Return(false)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction global id %s does not exist", globalId), string(res.Result))

	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_SUCCESS,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction with global Id %s is finished", globalId), string(res.Result))

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("%s is not in transaction %s, %v", id0, globalId, txInfo), string(res.Result))

	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_SUCCESS
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("%s has already reported result", id0), string(res.Result))

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_BEGIN
	expTxInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	expTxInfo.ChildTxInfo[id0] = pb.TransactionStatus_SUCCESS
	expTxInfo.ChildTxInfo[id1] = pb.TransactionStatus_BEGIN
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	mockStub.EXPECT().SetObject(globalInfoKey, expTxInfo).MaxTimes(1)
	res = im.Report(id0, 0)
	assert.True(t, res.Ok)

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	expTxInfo.GlobalState = pb.TransactionStatus_SUCCESS
	expTxInfo.ChildTxInfo[id0] = pb.TransactionStatus_SUCCESS
	expTxInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	mockStub.EXPECT().SetObject(globalInfoKey, expTxInfo).MaxTimes(1)
	res = im.Report(id0, 0)
	assert.True(t, res.Ok)

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	expTxInfo.GlobalState = pb.TransactionStatus_FAILURE
	expTxInfo.ChildTxInfo[id0] = pb.TransactionStatus_FAILURE
	expTxInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	mockStub.EXPECT().SetObject(globalInfoKey, expTxInfo).MaxTimes(1)
	res = im.Report(id0, 1)
	assert.True(t, res.Ok)
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

	im := &TransactionManager{mockStub}

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, recSuccess).Return(true).MaxTimes(1)
	res := im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "1", string(res.Result))

	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "0", string(res.Result))

	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Get(id).Return(false, nil).MaxTimes(1)
	res = im.GetStatus(id)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("cannot get global id of child tx id %s", id), string(res.Result))

	globalId := "globalId"
	globalIdInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, globalId)
	mockStub.EXPECT().Get(id).Return(true, []byte(globalId)).AnyTimes()
	mockStub.EXPECT().GetObject(globalIdInfoKey, gomock.Any()).Return(false).MaxTimes(1)
	res = im.GetStatus(id)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction info for global id %s does not exist", globalId), string(res.Result))

	mockStub.EXPECT().GetObject(globalIdInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "0", string(res.Result))
}

func TestTransactionManager_BeginMultiTXs(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := "id0"
	id1 := "id1"
	globalId := "globalId"
	txInfoKey := fmt.Sprintf("%s-%s", PREFIX, globalId)
	globalInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, globalId)

	im := &TransactionManager{mockStub}

	mockStub.EXPECT().Has(txInfoKey).Return(true).MaxTimes(1)
	res := im.BeginMultiTXs(globalId, id0, id1)
	assert.False(t, res.Ok)
	assert.Equal(t, "Transaction id already exists", string(res.Result))

	mockStub.EXPECT().Has(txInfoKey).Return(false).AnyTimes()
	mockStub.EXPECT().Set(id0, []byte(globalId))
	mockStub.EXPECT().Set(id1, []byte(globalId))
	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_BEGIN
	mockStub.EXPECT().SetObject(globalInfoKey, txInfo)
	res = im.BeginMultiTXs(globalId, id0, id1)
	assert.True(t, res.Ok)
}
