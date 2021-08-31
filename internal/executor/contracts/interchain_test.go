package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
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
	o2 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(true, data0)

	interchain = pb.Interchain{
		ID:                   srcChainService.getFullServiceId(),
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	data1, err := interchain.Marshal()
	assert.Nil(t, err)
	o3 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(true, data1)
	gomock.InOrder(o1, o2, o3)

	im := &InterchainManager{mockStub}

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
	assert.Equal(t, data0, res.Result)

	res = im.Register(srcChainService.getChainServiceId())
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, data1, res.Result)
}

func TestInterchainManager_Interchain(t *testing.T) {
	srcChainService, _ := mockChainService()
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addr := types.NewAddress([]byte{0}).String()
	mockStub.EXPECT().Caller().Return(addr).AnyTimes()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	o1 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(false, nil)

	interchain := pb.Interchain{
		ID:                   addr,
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[addr] = 1
	interchain.ReceiptCounter[addr] = 1
	interchain.SourceReceiptCounter[addr] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)
	o2 := mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(true, data0)
	gomock.InOrder(o1, o2)

	im := &InterchainManager{mockStub}

	res := im.Interchain(srcChainService.getFullServiceId())
	assert.False(t, res.Ok)

	res = im.Interchain(srcChainService.getFullServiceId())
	assert.True(t, res.Ok)
	assert.Equal(t, data0, res.Result)
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

// TODO: ZR (This is caused by a section of code commented in checkIBTP）
func TestInterchainManager_HandleIBTP(t *testing.T) {
	srcChainService, dstChainService := mockChainService()
	unavailableChainID := "appchain3"
	unexistChainID := "appchain4"
	unavailableChainServiceID := fmt.Sprintf("bxh:%s:service", unavailableChainID)
	unexistChainServiceID := fmt.Sprintf("bxh:%s:service", unexistChainID)

	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	//fromPrivKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	//assert.Nil(t, err)
	//fromPubKey := fromPrivKey.PublicKey()
	//from, err := fromPubKey.Address()
	//assert.Nil(t, err)
	//
	//toPrivKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	//assert.Nil(t, err)
	//toPubKey := toPrivKey.PublicKey()
	//to, err := toPubKey.Address()
	//assert.Nil(t, err)

	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(service_mgr.ServiceKey(unexistChainServiceID)).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.PREFIX+unexistChainID).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Get(service_mgr.ServiceKey(unavailableChainServiceID)).Return(false, nil).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(1)).AnyTimes()
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
	mockStub.EXPECT().Get(service_mgr.ServiceKey(srcChainService.getFullServiceId())).Return(true, data0).AnyTimes()

	interchain2 := pb.Interchain{
		ID:                   dstChainService.getFullServiceId(),
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	data2, err := interchain2.Marshal()
	assert.Nil(t, err)
	mockStub.EXPECT().Get(service_mgr.ServiceKey(dstChainService.getFullServiceId())).Return(true, data2).AnyTimes()

	appchain := &appchainMgr.Appchain{
		ID:      srcChainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "Relay1",
		Version: 0,
	}
	appchainData, err := json.Marshal(appchain)
	require.Nil(t, err)

	dstAppchain := &appchainMgr.Appchain{
		ID:      dstChainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "Relay2",
		Version: 0,
	}
	dstAppchainData, err := json.Marshal(dstAppchain)
	assert.Nil(t, err)

	unavailableChain := &appchainMgr.Appchain{
		ID:      unavailableChainID,
		Status:  governance.GovernanceFrozen,
		Desc:    "Relay1",
		Version: 0,
	}
	unavailableChainData, err := json.Marshal(unavailableChain)
	require.Nil(t, err)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), gomock.Eq("GetAppchain"), pb.String(srcChainID)).Return(boltvm.Success(appchainData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), gomock.Eq("GetAppchain"), pb.String(dstChainID)).Return(boltvm.Success(dstAppchainData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), gomock.Eq("GetAppchain"), pb.String(unexistChainID)).Return(boltvm.Error("")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), gomock.Eq("GetAppchain"), pb.String(unavailableChainID)).Return(boltvm.Success(unavailableChainData)).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxIndex().Return(uint64(1)).AnyTimes()
	mockStub.EXPECT().PostInterchainEvent(gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxHash().Return(&types.Hash{}).AnyTimes()
	mockStub.EXPECT().Get(BitXHubID).Return(true, []byte("bxh")).AnyTimes()

	im := &InterchainManager{mockStub}

	ibtp := &pb.IBTP{}

	res := im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))

	ibtp.From = srcChainService.getChainServiceId()
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidTargetService))

	ibtp.From = unexistChainServiceID
	ibtp.To = dstChainService.getChainServiceId()
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), CurAppchainNotAvailable))

	ibtp.From = unavailableChainServiceID
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), CurAppchainNotAvailable))

	//mockStub.EXPECT().Caller().Return(to.String()).MaxTimes(1)
	//ibtp.From = srcChainService.getChainServiceId()
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))
	//
	//mockStub.EXPECT().Caller().Return(from.String()).MaxTimes(8)
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", gomock.Any()).Return(boltvm.Error(""))
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), TargetServiceNotAvailable))
	//
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", gomock.Any()).Return(boltvm.Success(nil))
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), TargetServiceNotAvailable))
	//
	//service := Service{
	//	ChainID:   dstChainService.ChainId,
	//	ServiceID: dstChainService.ServiceId,
	//	Ordered:   true,
	//	Status:    0,
	//}
	//
	//dstServiceData, err := json.Marshal(service)
	//assert.Nil(t, err)
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", gomock.Any()).Return(boltvm.Success(dstServiceData))
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), "not permitted to visit"))
	//
	//service.Permission = map[string]struct{}{srcChainService.getFullServiceId(): {}}
	//dstServiceData, err = json.Marshal(service)
	//assert.Nil(t, err)
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", gomock.Any()).Return(boltvm.Success(dstServiceData))
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexExist))
	//
	//ibtp.Index = 3
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", gomock.Any()).Return(boltvm.Success(dstServiceData))
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexWrong))
	//
	//ibtp.Type = pb.IBTP_RECEIPT_SUCCESS
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", gomock.Any()).Return(boltvm.Error(""))
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), ServiceNotAvailable))
	//
	//service = Service{
	//	ChainID:   srcChainService.ChainId,
	//	ServiceID: srcChainService.ServiceId,
	//	Ordered:   true,
	//	Status:    0,
	//}
	//srcServiceData, err := json.Marshal(service)
	//assert.Nil(t, err)
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", pb.String(srcChainService.getChainServiceId())).Return(boltvm.Success(srcServiceData)).AnyTimes()
	//mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "GetServiceInfo", pb.String(dstChainService.getChainServiceId())).Return(boltvm.Success(dstServiceData)).AnyTimes()
	//ibtp.Index = 1
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexExist))
	//
	//ibtp.Index = 3
	//res = im.HandleIBTP(ibtp)
	//assert.False(t, res.Ok)
	//assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexWrong))
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
	assert.Equal(t, "wrong ibtp id", string(res.Result))

	unexistId := getIBTPID(srcChainService.getFullServiceId(), dstChainService.getChainServiceId(), 10)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-tx-%s", unexistId), gomock.Any()).Return(false)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-receipt-tx-%s", unexistId), gomock.Any()).Return(false)

	res = im.GetIBTPByID(unexistId, true)
	assert.False(t, res.Ok)
	assert.Equal(t, "this ibtp id does not exist", string(res.Result))

	res = im.GetIBTPByID(unexistId, false)
	assert.False(t, res.Ok)
	assert.Equal(t, "this ibtp id does not exist", string(res.Result))

	validID := getIBTPID(srcChainService.getFullServiceId(), dstChainService.getChainServiceId(), 1)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-tx-%s", validID), gomock.Any()).Return(true)
	res = im.GetIBTPByID(validID, true)
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
