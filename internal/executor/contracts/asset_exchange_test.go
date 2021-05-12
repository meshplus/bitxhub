package contracts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetExchange_Init(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	ae := &AssetExchange{mockStub}

	res := ae.Init(appchainMethod, appchainMethod2, []byte{1})
	assert.False(t, res.Ok)

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    0,
	}
	info, err := aei.Marshal()
	assert.Nil(t, err)

	mockStub.EXPECT().Has(AssetExchangeKey(aei.Id)).Return(true).MaxTimes(1)
	res = ae.Init(appchainMethod, appchainMethod2, info)
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exhcange id already exists", string(res.Result))

	mockStub.EXPECT().Has(AssetExchangeKey(aei.Id)).Return(false).AnyTimes()
	res = ae.Init(appchainMethod, appchainMethod2, info)
	assert.False(t, res.Ok)
	assert.Equal(t, "illegal asset exchange info", string(res.Result))

	aei.AssetOnDst = 100
	info, err = aei.Marshal()
	assert.Nil(t, err)

	aer := AssetExchangeRecord{
		Chain0: appchainMethod,
		Chain1: appchainMethod2,
		Status: AssetExchangeInit,
		Info:   aei,
	}
	mockStub.EXPECT().SetObject(AssetExchangeKey(aei.Id), aer)
	res = ae.Init(appchainMethod, appchainMethod2, info)
	assert.True(t, res.Ok)
}

func TestAssetExchange_Redeem(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	ae := &AssetExchange{mockStub}

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    100,
	}

	aer := AssetExchangeRecord{
		Chain0: appchainMethod,
		Chain1: appchainMethod2,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).Return(false).MaxTimes(1)
	res := ae.Redeem(appchainMethod, appchainMethod2, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange record does not exist", string(res.Result))

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).MaxTimes(1)
	res = ae.Redeem(appchainMethod, appchainMethod2, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange status for this id is not 'Init'", string(res.Result))

	aer.Status = AssetExchangeInit
	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).AnyTimes()
	res = ae.Redeem(appchainMethod2, appchainMethod2, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("invalid participator of asset exchange id %s", aei.Id), string(res.Result))

	expAer := AssetExchangeRecord{
		Chain0: appchainMethod,
		Chain1: appchainMethod2,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}
	mockStub.EXPECT().SetObject(AssetExchangeKey(aei.Id), expAer)
	res = ae.Redeem(appchainMethod, appchainMethod2, []byte(aei.Id))
	assert.True(t, res.Ok)
}

func TestAssetExchange_Refund(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	ae := &AssetExchange{mockStub}

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    100,
	}

	aer := AssetExchangeRecord{
		Chain0: appchainMethod,
		Chain1: appchainMethod2,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).Return(false).MaxTimes(1)
	res := ae.Refund(appchainMethod, appchainMethod2, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange record does not exist", string(res.Result))

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).MaxTimes(1)
	res = ae.Refund(appchainMethod, appchainMethod2, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange status for this id is not 'Init'", string(res.Result))

	aer.Status = AssetExchangeInit
	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).AnyTimes()
	res = ae.Refund(appchainMethod2, appchainMethod2, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("invalid participator of asset exchange id %s", aei.Id), string(res.Result))

	expAer := AssetExchangeRecord{
		Chain0: appchainMethod,
		Chain1: appchainMethod2,
		Status: AssetExchangeRefund,
		Info:   aei,
	}
	mockStub.EXPECT().SetObject(AssetExchangeKey(aei.Id), expAer)
	res = ae.Refund(appchainMethod, appchainMethod2, []byte(aei.Id))
	assert.True(t, res.Ok)
}

func TestAssetExchange_GetStatus(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    100,
	}

	aer := AssetExchangeRecord{
		Chain0: appchainMethod,
		Chain1: appchainMethod2,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}

	ae := &AssetExchange{mockStub}

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).Return(false).MaxTimes(1)
	res := ae.GetStatus(aei.Id)
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange record does not exist", string(res.Result))

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).MaxTimes(1)
	res = ae.GetStatus(aei.Id)
	assert.True(t, res.Ok)
	assert.Equal(t, "1", string(res.Result))
}

func TestInterRelayBroker_InvokeInterRelayContract(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&boltvm.Response{
		Ok:     true,
		Result: nil,
	}).AnyTimes()

	realArgs := [2][]byte{[]byte("123"), []byte("123454")}
	args, err := json.Marshal(realArgs)
	require.Nil(t, err)

	interRelayBroker := InterRelayBroker{mockStub}
	res := interRelayBroker.InvokeInterRelayContract(constant.DIDRegistryContractAddr.String(), "", args)
	require.False(t, res.Ok)

	res = interRelayBroker.InvokeInterRelayContract(constant.MethodRegistryContractAddr.String(), "Synchronize", args)
	require.True(t, res.Ok)

	res = interRelayBroker.GetInCouterMap()
	require.True(t, res.Ok)

	res = interRelayBroker.GetOutCouterMap()
	require.True(t, res.Ok)

	res = interRelayBroker.GetOutMessageMap()
	require.True(t, res.Ok)

	ibtp := &pb.IBTP{
		From: appchainMethod,
		To:   appchainMethod2,
		Type: pb.IBTP_INTERCHAIN,
	}

	ibtps := &pb.IBTPs{Ibtps: []*pb.IBTP{ibtp}}
	data, err := ibtps.Marshal()
	require.Nil(t, err)

	res = interRelayBroker.RecordIBTPs(data)
	require.True(t, res.Ok)

	res = interRelayBroker.GetOutMessage("123", 1)
	require.True(t, res.Ok)
}
