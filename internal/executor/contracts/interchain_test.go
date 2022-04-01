package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	service_mgr "github.com/meshplus/bitxhub-core/service-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	srcChainID    = "appchain1"
	dstChainID    = "appchain2"
	srcServiceID  = "service1"
	dstServiceID  = "service2"
	relayAdminDID = "did:bitxhub:relay:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	docAddr       = "/ipfs/QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
	docHash       = "QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
)

var fakeSig = []byte("fake signature")

func TestInterchainManager_Register(t *testing.T) {
	srcChainService, dstChainService := mockChainService()
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	//mockStub.EXPECT().Caller().Return(addr).AnyTimes()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(BitXHubID).Return(true, []byte("bxh")).AnyTimes()
	o1 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(false, nil)

	interchain := pb.Interchain{
		ID:                   srcChainService.getFullServiceId(),
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[dstChainService.getFullServiceId()] = 1
	interchain.ReceiptCounter[dstChainService.getFullServiceId()] = 1
	interchain.SourceReceiptCounter[dstChainService.getFullServiceId()] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)
	o2 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(true, data0).Times(3)

	interchain = pb.Interchain{
		ID:                   srcChainService.getFullServiceId(),
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	data1, err := interchain.Marshal()
	assert.Nil(t, err)
	o3 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(true, data1).Times(2)
	gomock.InOrder(o1, o2, o3)

	im := &InterchainManager{mockStub}

	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()

	res := im.Register(srcChainService.getChainServiceId())
	assert.Equal(t, true, res.Ok)

	ic := &pb.Interchain{}
	err = ic.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, srcChainService.getFullServiceId(), ic.ID)
	assert.Equal(t, 0, len(ic.InterchainCounter))
	assert.Equal(t, 0, len(ic.ReceiptCounter))
	assert.Equal(t, 0, len(ic.SourceReceiptCounter))

	res = im.Register(srcChainService.getChainServiceId())
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, string(data0), string(res.Result))

	res = im.Register(srcChainService.getChainServiceId())
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, data1, res.Result)
}

func TestInterchainManager_GetInterchain(t *testing.T) {
	srcChainService, dstChainService := mockChainService()
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	o1 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(false, nil)

	interchain := pb.Interchain{
		ID:                   srcChainService.getFullServiceId(),
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[dstChainService.getFullServiceId()] = 1
	interchain.ReceiptCounter[dstChainService.getFullServiceId()] = 1
	interchain.SourceReceiptCounter[dstChainService.getFullServiceId()] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)
	o2 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(true, data0)
	gomock.InOrder(o1, o2)

	im := &InterchainManager{mockStub}

	res := im.GetInterchain(srcChainService.getFullServiceId())
	assert.False(t, res.Ok)

	res = im.GetInterchain(srcChainService.getFullServiceId())
	assert.True(t, res.Ok)
	assert.Equal(t, data0, res.Result)
}

func TestInterchainManager_HandleIBTP(t *testing.T) {
	srcChainService, dstChainService := mockChainService()

	unexistServiceID := "service3"
	unexistServiceChainServiceID := fmt.Sprintf("%s:%s", srcChainID, unexistServiceID)
	unexistServiceFullServiceID := fmt.Sprintf("bxh:%s", unexistServiceChainServiceID)

	unavailableServiceID := "service4"
	unavailableServiceChainServiceID := fmt.Sprintf("%s:%s", srcChainID, unavailableServiceID)
	unavailableServiceFullServiceID := fmt.Sprintf("bxh:%s", unavailableServiceChainServiceID)

	unavailableDstServiceChainServiceID := fmt.Sprintf("%s:%s", dstChainID, unavailableServiceID)
	unavailableDstServiceFullServiceID := fmt.Sprintf("bxh:%s", unavailableDstServiceChainServiceID)

	unPermissionServiceID := "service5"
	unPermissionServiceChainServiceID := fmt.Sprintf("%s:%s", dstChainID, unPermissionServiceID)
	unPermissionServiceServiceID := fmt.Sprintf("bxh:%s", unPermissionServiceChainServiceID)

	otherBitxhubID := "bxh2"
	otherBitxhubServiceID := fmt.Sprintf("%s:appchain:service", otherBitxhubID)
	otherBitxhubServiceID1 := fmt.Sprintf("%s:appchain1:service", otherBitxhubID)

	unavailableBitxhubID := "bxh3"
	unavailableBitxhubServiceID := fmt.Sprintf("%s:appchain:service", unavailableBitxhubID)

	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(1)).AnyTimes()
	mockStub.EXPECT().GetCurrentHeight().Return(uint64(100)).AnyTimes()
	mockStub.EXPECT().GetObject(MultiTxNotifyKey(100), gomock.Any()).SetArg(1, make(map[string][]string)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true).AnyTimes()

	interchain := pb.Interchain{
		ID:                   srcChainService.getFullServiceId(),
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[dstChainService.getFullServiceId()] = 1
	interchain.ReceiptCounter[dstChainService.getFullServiceId()] = 1
	interchain.SourceReceiptCounter[dstChainService.getFullServiceId()] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)

	interchain2 := pb.Interchain{
		ID:                   dstChainService.getFullServiceId(),
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	data2, err := interchain2.Marshal()
	assert.Nil(t, err)

	mockStub.EXPECT().Get(gomock.Any()).DoAndReturn(func(key string) (bool, []byte) {
		switch key {
		case serviceKey(srcChainService.getFullServiceId()):
			return true, data0
		case serviceKey(dstChainService.getFullServiceId()):
			return true, data2
		case BitXHubID:
			return true, []byte("bxh")
		default:
			return true, nil
		}
		return false, nil
	}).AnyTimes()

	unavailableService := &service_mgr.Service{
		ServiceID: unavailableServiceChainServiceID,
		ChainID:   srcChainID,
		Status:    governance.GovernanceUnavailable,
		Ordered:   true,
	}
	unavailableServiceData, err := json.Marshal(unavailableService)
	require.Nil(t, err)

	srcService := &service_mgr.Service{
		ServiceID: srcChainService.ServiceId,
		ChainID:   srcChainService.ChainId,
		Status:    governance.GovernanceAvailable,
	}
	srcServiceData, err := json.Marshal(srcService)
	require.Nil(t, err)

	dstService := &service_mgr.Service{
		ServiceID: dstChainService.ServiceId,
		ChainID:   dstChainService.ChainId,
		Status:    governance.GovernanceAvailable,
		Ordered:   true,
	}
	dstServiceData, err := json.Marshal(dstService)
	require.Nil(t, err)

	unPermissionService := &service_mgr.Service{
		ServiceID: dstChainService.ServiceId,
		ChainID:   dstChainService.ChainId,
		Status:    governance.GovernanceAvailable,
		Permission: map[string]struct{}{
			srcChainService.getFullServiceId(): {},
		},
	}
	unPermissionServiceData, err := json.Marshal(unPermissionService)
	require.Nil(t, err)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr string, method string, args ...*pb.Arg) *boltvm.Response {
			if method == "IsAvailableBitxhub" {
				switch string(args[0].GetValue()) {
				case otherBitxhubID:
					return boltvm.Success([]byte(TRUE))
				case unavailableBitxhubID:
					return boltvm.Success([]byte(FALSE))
				}
			}

			return boltvm.Error("", "")
		}).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", gomock.Any()).DoAndReturn(
		func(addr string, method string, args ...*pb.Arg) *boltvm.Response {
			switch string(args[0].GetValue()) {
			case unexistServiceChainServiceID:
				return boltvm.Error("", "")
			case unavailableServiceChainServiceID:
				return boltvm.Success(unavailableServiceData)
			case unavailableDstServiceChainServiceID:
				return boltvm.Success(unavailableServiceData)
			case srcChainService.getChainServiceId():
				return boltvm.Success(srcServiceData)
			case dstChainService.getChainServiceId():
				return boltvm.Success(dstServiceData)
			case unPermissionServiceChainServiceID:
				return boltvm.Success(unPermissionServiceData)
			}

			return boltvm.Error("", "")
		}).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).DoAndReturn(
		func(addr string, method string, args ...*pb.Arg) *boltvm.Response {
			switch string(args[0].GetValue()) {
			case unexistServiceChainServiceID:
				return boltvm.Error("", "")
			case unavailableServiceChainServiceID:
				return boltvm.Success([]byte(FALSE))
			case unavailableDstServiceChainServiceID:
				return boltvm.Success([]byte(FALSE))
			case srcChainService.getChainServiceId():
				return boltvm.Success([]byte(TRUE))
			case dstChainService.getChainServiceId():
				return boltvm.Success([]byte(TRUE))
			case unPermissionServiceChainServiceID:
				return boltvm.Success([]byte(TRUE))
			}

			return boltvm.Error("", "")
		}).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "RecordInvokeService", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxIndex().Return(uint64(1)).AnyTimes()
	mockStub.EXPECT().GetTxHash().Return(&types.Hash{}).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()

	im := &InterchainManager{mockStub}
	ibtp := &pb.IBTP{}

	res := im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))

	// no bxhid is illegal
	ibtp.From = srcChainService.getChainServiceId()
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), string(boltvm.InterchainInvalidIBTPParseSourceErrorCode))

	// 1. request
	// 1.1 source local
	// 1.1.1 source check failed
	ibtp.To = dstChainService.getFullServiceId()
	ibtp.From = unexistServiceFullServiceID
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), string(boltvm.InterchainSourceServiceNotAvailableCode))

	ibtp.From = unavailableServiceFullServiceID
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), string(boltvm.InterchainSourceServiceNotAvailableCode))

	// 1.1.2 destination check failed
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.TransactionMgrContractAddr.Address().String()), gomock.Eq("Begin"), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "Begin error")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.TransactionMgrContractAddr.Address().String()), gomock.Eq("Begin"), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(addr string, method string, args ...*pb.Arg) *boltvm.Response {
			change := StatusChange{
				PrevStatus: -1,
				CurStatus:  pb.TransactionStatus_BEGIN,
			}
			if b, _ := strconv.ParseBool(string(args[2].Value)); b {
				change.CurStatus = pb.TransactionStatus_BEGIN_FAILURE
			}
			data, _ := json.Marshal(change)
			return boltvm.Success(data)
		}).AnyTimes()

	ibtp.From = srcChainService.getFullServiceId()
	ibtp.To = unavailableDstServiceFullServiceID
	ibtp.Index = 1
	mockStub.EXPECT().PostInterchainEvent(map[string]uint64{srcChainService.ChainId: 1, dstChainID: 1}).MaxTimes(1)
	mockStub.EXPECT().PostInterchainEvent(map[string]uint64{dstChainID: 1}).MaxTimes(1)
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.To = unPermissionServiceServiceID
	mockStub.EXPECT().PostInterchainEvent(map[string]uint64{srcChainService.ChainId: 1, dstChainService.ChainId: 1}).MaxTimes(1)
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Index = 1
	ibtp.To = unavailableBitxhubServiceID
	mockStub.EXPECT().PostInterchainEvent(map[string]uint64{srcChainService.ChainId: 1, DEFAULT_UNION_PIER_ID: 1}).MaxTimes(1)
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Index = 3
	ibtp.To = dstChainService.getFullServiceId()
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexWrong))

	ibtp.Index = 1
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexExist))

	ibtp.Type = pb.IBTP_RECEIPT_FAILURE
	ibtp.To = dstChainService.getFullServiceId()
	ibtp.Index = 1
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexExist))

	mockStub.EXPECT().PostInterchainEvent(gomock.Any()).AnyTimes()
	// 1.1.3 check ok (targetErr nil)
	ibtp.Index = 2
	ibtp.Type = pb.IBTP_INTERCHAIN
	ibtp.From = srcChainService.getFullServiceId()
	ibtp.To = dstChainService.getFullServiceId()
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	// 1.1.4 check ok (targetErr not nil)
	ibtp.Index = 1
	ibtp.To = unavailableDstServiceFullServiceID
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok, string(res.Result))

	// 1.2 source not local
	ibtp.From = otherBitxhubServiceID
	ibtp.To = otherBitxhubServiceID1
	ibtp.Type = pb.IBTP_INTERCHAIN
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok, string(res.Result))
	assert.Equal(t, true, strings.Contains(string(res.Result), string(boltvm.InterchainInvalidIBTPNotInCurBXHCode)))

	ibtp.From = unavailableBitxhubServiceID
	ibtp.To = dstChainService.getFullServiceId()
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), string(boltvm.InterchainSourceBitXHubNotAvailableCode))

	ibtp.Index = 2
	ibtp.From = otherBitxhubServiceID
	ibtp.To = dstChainService.getFullServiceId()
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok, string(res.Result))
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexWrong), string(res.Result))

	// 2. response
	ibtp.From = srcChainService.getFullServiceId()
	ibtp.To = dstChainService.getFullServiceId()
	ibtp.Type = pb.IBTP_RECEIPT_FAILURE
	ibtp.Index = 1
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok, string(res.Result))
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexExist), string(res.Result))

	// 3. other
	ibtp.Type = -1
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok, string(res.Result))
	assert.Equal(t, true, strings.Contains(string(res.Result), string(boltvm.InterchainInvalidIBTPIllegalTypeCode)), string(res.Result))

	// =======================check ibtp end

	ibtp.Index = 2
	ibtp.Type = pb.IBTP_RECEIPT_FAILURE
	mockStub.EXPECT().CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), gomock.Eq("Report"), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "Report error")).Times(1)
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok, string(res.Result))

	change := StatusChange{
		PrevStatus: pb.TransactionStatus_BEGIN,
		CurStatus:  pb.TransactionStatus_SUCCESS,
	}
	data, _ := json.Marshal(change)
	mockStub.EXPECT().CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), gomock.Eq("Report"), gomock.Any(), gomock.Any()).Return(boltvm.Success(data)).Times(1)
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok, string(res.Result))

	// ======================== ibtp group not nil
	ibtp.Type = pb.IBTP_INTERCHAIN
	ibtp.Index = 2
	ibtp.From = srcChainService.getFullServiceId()
	ibtp.To = dstChainService.getFullServiceId()
	ibtp.Group = &pb.StringUint64Map{
		Keys: []string{"111"},
		Vals: []uint64{1},
	}
	mockStub.EXPECT().CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), gomock.Eq("BeginMultiTXs"), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "BeginMultiTXs error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), gomock.Eq("BeginMultiTXs"), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(data)).Times(1)
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok, string(res.Result))
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok, string(res.Result))

	// ========================= test notifySrcDst
	ibtp.Type = pb.IBTP_RECEIPT_FAILURE
	ibtp.Index = 2
	change1 := StatusChange{
		PrevStatus:   pb.TransactionStatus_BEGIN,
		CurStatus:    pb.TransactionStatus_SUCCESS,
		OtherIBTPIDs: []string{fmt.Sprintf("%s-%s-1", fromServiceId, toServiceId)},
	}
	data1, _ := json.Marshal(change1)
	mockStub.EXPECT().CrossInvoke(constant.TransactionMgrContractAddr.Address().String(), gomock.Eq("Report"), gomock.Any(), gomock.Any()).Return(boltvm.Success(data1)).Times(1)
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok, string(res.Result))
}

func TestInterchainManager_DeleteInterchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	im := &InterchainManager{mockStub}

	mockStub.EXPECT().Delete(gomock.Any())
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()
	res := im.DeleteInterchain("1356:123:123")
	assert.True(t, res.Ok, string(res.Result))
}

func TestInterchainManager_GetIBTPByID(t *testing.T) {
	srcChainService, dstChainService := mockChainService()
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	from := types.NewAddress([]byte{0}).String()

	mockStub.EXPECT().Caller().Return(from).AnyTimes()
	im := &InterchainManager{mockStub}

	res := im.GetIBTPByID("a", true)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), string(boltvm.InterchainWrongIBTPIDCode))

	unexistId := getIBTPID(srcChainService.getFullServiceId(), dstChainService.getChainServiceId(), 10)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-tx-%s", unexistId), gomock.Any()).Return(false)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-receipt-tx-%s", unexistId), gomock.Any()).Return(false)

	res = im.GetIBTPByID(unexistId, true)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), string(boltvm.InterchainNonexistentIBTPCode))

	res = im.GetIBTPByID(unexistId, false)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), string(boltvm.InterchainNonexistentIBTPCode))

	validID := getIBTPID(srcChainService.getFullServiceId(), dstChainService.getChainServiceId(), 1)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-tx-%s", validID), gomock.Any()).Return(true)
	res = im.GetIBTPByID(validID, true)
	assert.True(t, res.Ok)
}

func TestInterchainManager_HandleIBTPData(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	im := &InterchainManager{mockStub}

	srcChainService, dstChainService := mockChainService()
	ibtp := &pb.IBTP{
		From:  srcChainService.getChainServiceId(),
		To:    dstChainService.getChainServiceId(),
		Index: 0,
		Type:  pb.IBTP_INTERCHAIN,
	}
	input, err := ibtp.Marshal()
	assert.Nil(t, err)

	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).AnyTimes()

	res := im.HandleIBTPData(input)
	assert.False(t, res.Ok, string(res.Result))
}

func TestInterchainManager_GetAllServiceIDs(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	im := &InterchainManager{mockStub}

	mockStub.EXPECT().Query(gomock.Any()).Return(false, nil).Times(1)
	res := im.GetAllServiceIDs()
	assert.True(t, res.Ok)

	interchain := &pb.Interchain{}
	interchainData, err := interchain.Marshal()
	assert.Nil(t, err)
	data := [][]byte{}
	data = append(data, interchainData)
	mockStub.EXPECT().Query(gomock.Any()).Return(true, data).Times(1)

	res = im.GetAllServiceIDs()
	assert.True(t, res.Ok)

	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("bxh")).Times(1)
	res = im.GetBitXHubID()
	assert.False(t, res.Ok)
	res = im.GetBitXHubID()
	assert.True(t, res.Ok)
}

//func TestInterchainManager_HandleUnionIBTP(t *testing.T) {
//	mockCtl := gomock.NewController(t)
//	mockStub := mock_stub.NewMockStub(mockCtl)
//
//	from := types.NewAddress([]byte{0}).String()
//	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
//	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
//	mockStub.EXPECT().Has(gomock.Any()).Return(true).AnyTimes()
//
//	interchain := pb.Interchain{
//		ID:                   appchainMethod,
//		InterchainCounter:    make(map[string]uint64),
//		ReceiptCounter:       make(map[string]uint64),
//		SourceReceiptCounter: make(map[string]uint64),
//	}
//	interchain.InterchainCounter[appchainMethod2] = 1
//	interchain.ReceiptCounter[appchainMethod2] = 1
//	interchain.SourceReceiptCounter[appchainMethod2] = 1
//	data0, err := interchain.Marshal()
//	assert.Nil(t, err)
//
//	relayChain := &appchainMgr.Appchain{
//		Status:        governance.GovernanceAvailable,
//		ID:            appchainMethod,
//		Name:          "appchain" + appchainMethod,
//		Validators:    "",
//		ConsensusType: "",
//		ChainType:     "fabric",
//		Desc:          "",
//		Version:       "",
//		PublicKey:     "pubkey",
//	}
//
//	keys := make([]crypto.PrivateKey, 0, 4)
//	var bv BxhValidators
//	addrs := make([]string, 0, 4)
//	for i := 0; i < 4; i++ {
//		keyPair, err := asym.GenerateKeyPair(crypto.Secp256k1)
//		require.Nil(t, err)
//		keys = append(keys, keyPair)
//		address, err := keyPair.PublicKey().Address()
//		require.Nil(t, err)
//		addrs = append(addrs, address.String())
//	}
//
//	bv.Addresses = addrs
//	addrsData, err := json.Marshal(bv)
//	require.Nil(t, err)
//	relayChain.Validators = string(addrsData)
//
//	data, err := json.Marshal(relayChain)
//	assert.Nil(t, err)
//
//	mockStub.EXPECT().Get(appchainMgr.PREFIX+appchainMethod).Return(true, data0).AnyTimes()
//	mockStub.EXPECT().Get(appchainMgr.PREFIX+appchainMethod+"-"+appchainMethod).Return(true, data0).AnyTimes()
//	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(data)).AnyTimes()
//	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
//	mockStub.EXPECT().GetTxIndex().Return(uint64(1)).AnyTimes()
//	mockStub.EXPECT().PostInterchainEvent(gomock.Any()).AnyTimes()
//	mockStub.EXPECT().GetTxHash().Return(&types.Hash{}).AnyTimes()
//
//	im := &InterchainManager{mockStub}
//
//	ibtp := &pb.IBTP{
//		From:    appchainMethod + "-" + appchainMethod,
//		To:      appchainMethod2,
//		Index:   0,
//		Type:    pb.IBTP_INTERCHAIN,
//		Proof:   nil,
//		Payload: nil,
//		Version: "",
//		Extra:   nil,
//	}
//
//	mockStub.EXPECT().Caller().Return(from).AnyTimes()
//
//	res := im.handleUnionIBTP(ibtp)
//	assert.False(t, res.Ok)
//	assert.Equal(t, "wrong index, required 2, but 0", string(res.Result))
//
//	ibtp.Index = 2
//
//	ibtpHash := ibtp.Hash()
//	hash := sha256.Sum256([]byte(ibtpHash.String()))
//	sign := &pb.SignResponse{Sign: make(map[string][]byte)}
//	for _, key := range keys {
//		signData, err := key.Sign(hash[:])
//		require.Nil(t, err)
//
//		address, err := key.PublicKey().Address()
//		require.Nil(t, err)
//		ok, err := asym.Verify(crypto.Secp256k1, signData[:], hash[:], *address)
//		require.Nil(t, err)
//		require.True(t, ok)
//		sign.Sign[address.String()] = signData
//	}
//	signData, err := sign.Marshal()
//	require.Nil(t, err)
//	ibtp.Proof = signData
//
//	res = im.handleUnionIBTP(ibtp)
//	assert.True(t, res.Ok)
//}

func mockChainService() (*ChainService, *ChainService) {
	srcChainService := &ChainService{
		BxhId:     "bxh",
		ChainId:   srcChainID,
		ServiceId: srcServiceID,
		IsLocal:   true,
	}

	dstChainService := &ChainService{
		BxhId:     "bxh",
		ChainId:   dstChainID,
		ServiceId: dstServiceID,
		IsLocal:   true,
	}

	return srcChainService, dstChainService
}
