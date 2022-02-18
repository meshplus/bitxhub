package contracts

import (
	"testing"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"

	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
)

const (
	illegalFromServiceId = "fromServiceId"
	illegalToServiceId   = "toServiceId"
	fromServiceId        = "1356:appchain1:service1"
	toServiceId          = "1356:appchain2:service2"
	brokerFuncs          = "CallFunc, Callback, Rollback"
	args                 = "args"
)

func TestInterBroker_GetMeta(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	ib := &InterBroker{
		Stub: mockStub,
	}

	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).AnyTimes()

	res := ib.GetInMeta()
	assert.True(t, res.Ok, string(res.Result))

	res = ib.GetOutMeta()
	assert.True(t, res.Ok, string(res.Result))

	res = ib.GetCallbackMeta()
	assert.True(t, res.Ok, string(res.Result))
}

func TestInterBroker_EmitInterchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	ib := &InterBroker{
		Stub: mockStub,
	}

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "HandleIBTPData", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// splitFuncs error
	res := ib.EmitInterchain(fromServiceId, toServiceId, "", "", "", "")
	assert.False(t, res.Ok, string(res.Result))

	res = ib.EmitInterchain(fromServiceId, toServiceId, brokerFuncs, args, "", "")
	assert.True(t, res.Ok, string(res.Result))
}

func TestInterBroker_InvokeReceipt(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	ib := &InterBroker{
		Stub: mockStub,
	}

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(CallbackCounterKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true).SetArg(1, &InterchainInvoke{
		CallFunc: CallFunc{
			CallF: "",
			Args:  nil,
		},
		Callback: CallFunc{},
		Rollback: CallFunc{},
	}).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true).SetArg(1, &InterchainInvoke{
		CallFunc: CallFunc{
			CallF: "1",
			Args:  []string{"1"},
		},
		Callback: CallFunc{
			CallF: "1",
			Args:  []string{"1"},
		},
		Rollback: CallFunc{
			CallF: "1",
			Args:  []string{"1"},
		},
	}).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "HandleIBTPData", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvokeEVM(gomock.Any(), gomock.Any()).AnyTimes()

	illegalIbtp := &pb.IBTP{
		From: illegalFromServiceId,
		To:   illegalToServiceId,
	}
	illegalIbtpData, err := illegalIbtp.Marshal()
	assert.Nil(t, err)

	responseIbtp := &pb.IBTP{
		From: fromServiceId,
		To:   toServiceId,
		Type: pb.IBTP_RECEIPT_SUCCESS,
	}
	responseIbtpData, err := responseIbtp.Marshal()
	assert.Nil(t, err)

	rollbackResponseIbtp := &pb.IBTP{
		From: fromServiceId,
		To:   toServiceId,
		Type: pb.IBTP_RECEIPT_ROLLBACK,
	}
	rollbackResponseIbtpData, err := rollbackResponseIbtp.Marshal()
	assert.Nil(t, err)

	// Split to error
	res := ib.InvokeReceipt(illegalIbtpData)
	assert.False(t, res.Ok, string(res.Result))

	// nonexistent interchain
	res = ib.InvokeReceipt(responseIbtpData)
	assert.False(t, res.Ok, string(res.Result))

	// response ok
	res = ib.InvokeReceipt(responseIbtpData)
	assert.True(t, res.Ok, string(res.Result))

	res = ib.InvokeReceipt(rollbackResponseIbtpData)
	assert.True(t, res.Ok, string(res.Result))
}

func TestInterBroker_InvokeInterchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	ib := &InterBroker{
		Stub: mockStub,
	}

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(InCounterKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().CrossInvokeEVM(gomock.Any(), gomock.Any()).Return(boltvm.Error("", "CrossInvokeEVM error")).Times(1)
	mockStub.EXPECT().CrossInvokeEVM(gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	illegalIbtp := &pb.IBTP{
		From: illegalFromServiceId,
		To:   illegalToServiceId,
	}
	illegalIbtpData, err := illegalIbtp.Marshal()
	assert.Nil(t, err)

	content := &pb.Content{
		Func: "1",
		Args: [][]byte{
			[]byte("1"),
		},
	}
	payload, err := content.Marshal()
	assert.Nil(t, err)
	interchainIbtp := &pb.IBTP{
		From:    fromServiceId,
		To:      toServiceId,
		Type:    pb.IBTP_INTERCHAIN,
		Payload: payload,
	}
	interchainIbtpData, err := interchainIbtp.Marshal()
	assert.Nil(t, err)

	// Split to error
	res := ib.InvokeInterchain(illegalIbtpData)
	assert.False(t, res.Ok, string(res.Result))

	res = ib.InvokeInterchain(interchainIbtpData)
	assert.True(t, res.Ok, string(res.Result))

	res = ib.InvokeInterchain(interchainIbtpData)
	assert.True(t, res.Ok, string(res.Result))
}

func TestInterBroker_GetMessage(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	ib := &InterBroker{
		Stub: mockStub,
	}

	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "GetIBTPById", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true).AnyTimes()

	res := ib.GetInMessage(fromServiceId, toServiceId, 1)
	assert.True(t, res.Ok, string(res.Result))

	// nonexistent out msg
	res = ib.GetOutMessage(fromServiceId, toServiceId, 1)
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = ib.GetOutMessage(fromServiceId, toServiceId, 1)
	assert.True(t, res.Ok, string(res.Result))
}
