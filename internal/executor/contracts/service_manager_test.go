package contracts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	service_mgr "github.com/meshplus/bitxhub-core/service-mgr"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/stretchr/testify/assert"
)

const (
	serviceID = "0xB2dD6977169c5067d3729E3deB9a82c3e7502BFb"
)

func TestServiceManager_RegisterService(t *testing.T) {
	sm, mockStub, services, _, rolesData := servicePrepare(t)

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()

	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	checkAppchainErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)
	checkInfoErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)
	submitErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)

	okReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	gomock.InOrder(governancePreErrReq, checkAppchainErrReq, checkInfoErrReq, submitErrReq, okReq)

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(FALSE))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// 1. check permission error
	res := sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[0].Name, string(services[0].Type), services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 2. governancePre error
	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[0].Name, string(services[0].Type), services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 3. check appchain error
	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[0].Name, string(services[0].Type), services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 4. check info error
	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[0].Name, string(services[0].Type), services[0].Intro, services[0].Ordered, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 5. submit error
	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[0].Name, string(services[0].Type), services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[0].Name, string(services[0].Type), services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_UpdateService(t *testing.T) {
	sm, mockStub, services, _, rolesData := servicePrepare(t)

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	checkPermissionErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	checkAppchainErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	checkInfoErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	updateErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	updateErrReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	submitErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	changeStatusErrReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	changeStatusErrReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)
	okReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(2)
	gomock.InOrder(governancePreErrReq, checkPermissionErrReq, checkAppchainErrReq, checkInfoErrReq, updateErrReq, updateErrReq2, submitErrReq, changeStatusErrReq1, changeStatusErrReq2, okReq)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(FALSE))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// 1. governancePre error
	res := sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 2. check permision error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 3. check appchain error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 4. check info error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, services[0].Ordered, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 5. update error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, services[0].Ordered, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 6. submit error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, false, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 7. change status error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, false, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, false, "", services[0].Details, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_LogoutService(t *testing.T) {
	sm, mockStub, services, _, rolesData := servicePrepare(t)
	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	checkPermissionErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	checkAppchainErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(2)
	submitErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	changeStatusErrReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	changeStatusErrReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)
	okReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(2)
	gomock.InOrder(governancePreErrReq, checkPermissionErrReq, checkAppchainErrReq, submitErrReq, changeStatusErrReq1, changeStatusErrReq2, okReq)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Error("is available error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(FALSE))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(TRUE))).AnyTimes()

	// 1. governancePre error
	res := sm.LogoutService(chainServiceID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 3. check permision error
	res = sm.LogoutService(chainServiceID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 4. check appchain error (is available error)
	res = sm.LogoutService(chainServiceID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 5. check appchain error (not available)
	res = sm.LogoutService(chainServiceID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 6. submit error
	res = sm.LogoutService(chainServiceID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 7. change status error
	res = sm.LogoutService(chainServiceID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.LogoutService(chainServiceID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_FreezeService(t *testing.T) {
	sm, mockStub, services, _, _ := servicePrepare(t)

	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(TRUE))).AnyTimes()

	res := sm.FreezeService(chainServiceID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_ActivateService(t *testing.T) {
	sm, mockStub, services, _, rolesData := servicePrepare(t)

	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[1]).Return(true).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(TRUE))).AnyTimes()

	res := sm.ActivateService(chainServiceID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

//func TestServiceManager_PauseChainService(t *testing.T) {
//	sm, mockStub, services, _, _ := servicePrepare(t)
//
//	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)
//
//	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
//	checkPermissionErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
//	changeStatusErrReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
//	changeStatusErrReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)
//	okReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(2)
//	gomock.InOrder(governancePreErrReq, checkPermissionErrReq, changeStatusErrReq1, changeStatusErrReq2, okReq)
//
//	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
//	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
//	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
//	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
//
//	mockStub.EXPECT().CurrentCaller().Return("").Times(1)
//	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
//
//	// check permission error
//	//governance pre error
//	res := sm.PauseChainService(chainServiceID)
//	assert.Equal(t, false, res.Ok, string(res.Result))
//	// check permission error
//	res = sm.PauseChainService(chainServiceID)
//	assert.Equal(t, false, res.Ok, string(res.Result))
//	// change status error
//	res = sm.PauseChainService(chainServiceID)
//	assert.Equal(t, false, res.Ok, string(res.Result))
//
//	res = sm.PauseChainService(chainServiceID)
//	assert.Equal(t, true, res.Ok, string(res.Result))
//}
//
//func TestServiceManager_UnPauseChainService(t *testing.T) {
//	sm, mockStub, services, _, _ := servicePrepare(t)
//
//	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)
//
//	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
//	checkPermissionErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
//	changeStatusErrReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
//	changeStatusErrReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)
//	okReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
//	gomock.InOrder(governancePreErrReq, checkPermissionErrReq, changeStatusErrReq1, changeStatusErrReq2, okReq)
//
//	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
//	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
//	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
//	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
//
//	mockStub.EXPECT().CurrentCaller().Return("").Times(1)
//	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
//	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Error("lock error")).Times(1)
//	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
//
//	// governance pre error
//	res := sm.UnPauseChainService(chainServiceID)
//	assert.Equal(t, false, res.Ok, string(res.Result))
//	// check permission error
//	res = sm.UnPauseChainService(chainServiceID)
//	assert.Equal(t, false, res.Ok, string(res.Result))
//	// change status error
//	res = sm.UnPauseChainService(chainServiceID)
//	assert.Equal(t, false, res.Ok, string(res.Result))
//	// unlock  proposal error
//	res = sm.UnPauseChainService(chainServiceID)
//	assert.Equal(t, false, res.Ok, string(res.Result))
//
//	res = sm.UnPauseChainService(chainServiceID)
//	assert.Equal(t, true, res.Ok, string(res.Result))
//}

//
//func TestserviceManager_ConfirmTransfer(t *testing.T) {
//	sm, mockStub, services, _ := servicePrepare(t)
//
//	mockStub.EXPECT().CurrentCaller().Return(noAsminAddr).Times(1)
//	mockStub.EXPECT().CurrentCaller().Return(ownerAddr).AnyTimes()
//	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
//	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
//	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
//
//	// 1. check permission error
//	res := sm.ConfirmTransfer(services[0].serviceID)
//	assert.Equal(t, false, res.Ok)
//	// 2. get service error
//	res = sm.ConfirmTransfer(services[0].serviceID)
//	assert.Equal(t, true, res.Ok)
//
//	res = sm.ConfirmTransfer(services[0].serviceID)
//	assert.Equal(t, true, res.Ok)
//}
//
//func TestserviceManager_Evaluateservice(t *testing.T) {
//	sm, mockStub, services, _ := servicePrepare(t)
//
//	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
//	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
//	mockStub.EXPECT().Caller().Return(ownerAddr).Times(1)
//	mockStub.EXPECT().Caller().Return(ownerAddr1).Times(1)
//	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
//
//	// 1. illegal score
//	res := sm.Evaluateservice(services[0].serviceID, services[0].Desc, 6)
//	assert.Equal(t, false, res.Ok)
//	// 2. get service error
//	res = sm.Evaluateservice(services[0].serviceID, services[0].Desc, 6)
//	assert.Equal(t, false, res.Ok)
//	// 3. has evaluated
//	res = sm.Evaluateservice(services[0].serviceID, services[0].Desc, 6)
//	assert.Equal(t, false, res.Ok)
//
//	res = sm.Evaluateservice(services[0].serviceID, services[0].Desc, 6)
//	assert.Equal(t, true, res.Ok)
//}
//
//func TestserviceManager_Query(t *testing.T) {
//	sm, mockStub, services, servicesData := servicePrepare(t)
//
//	mockStub.EXPECT().GetObject(ServiceMgr. :=ServiceKey(services[0].serviceID), gomock.Any()).Return(false).Times(1)
//	mockStub.EXPECT().GetObject(ServiceKey(services[0].serviceID), gomock.Any()).SetArg(1, 1).Return(true).Times(1)
//	mockStub.EXPECT().GetObject(ServiceKey(services[0].serviceID), gomock.Any()).SetArg(1, services[0]).Return(true).Times(1)
//	// 1. get error
//	res := sm.Getservice(services[0].serviceID)
//	assert.Equal(t, false, res.Ok)
//	// 2. marshal error
//	res = sm.Getservice(services[0].serviceID)
//	assert.Equal(t, false, res.Ok)
//	// 3. ok
//	res = sm.Getservice(services[0].serviceID)
//	assert.Equal(t, true, res.Ok)
//
//	mockStub.EXPECT().Query(servicePREFIX).Return(true, servicesData).AnyTimes()
//
//	res = sm.GetAllservices()
//	assert.Equal(t, true, res.Ok)
//	theservices := []*service{}
//	err := json.Unmarshal(res.Result, &theservices)
//	assert.Nil(t, err)
//	assert.Equal(t, 3, len(theservices))
//
//	res = sm.GetPermissionservices()
//	assert.Equal(t, true, res.Ok)
//	err = json.Unmarshal(res.Result, &theservices)
//	assert.Nil(t, err)
//	assert.Equal(t, 3, len(theservices))
//
//	mockStub.EXPECT().GetObject(OwnerKey(ownerAddr), gomock.Any()).SetArg(1, map[string]struct{}{
//		services[0].serviceID: struct{}{},
//	}).Return(true).Times(1)
//	mockStub.EXPECT().GetObject(serviceKey(services[0].serviceID), gomock.Any()).SetArg(1, services[0]).Return(true).Times(1)
//	res = sm.GetservicesByOwner(ownerAddr)
//	assert.Equal(t, true, res.Ok)
//	err = json.Unmarshal(res.Result, &theservices)
//	assert.Nil(t, err)
//	assert.Equal(t, 1, len(theservices))
//
//	mockStub.EXPECT().GetObject(serviceKey(services[0].serviceID), gomock.Any()).SetArg(1, services[0]).Return(true).Times(1)
//	mockStub.EXPECT().GetObject(serviceKey(services[0].serviceID), gomock.Any()).SetArg(1, services[1]).Return(true).Times(1)
//	res = sm.IsAvailable(services[0].serviceID)
//	assert.Equal(t, true, res.Ok)
//	assert.Equal(t, "true", string(res.Result))
//	res = sm.IsAvailable(services[0].serviceID)
//	assert.Equal(t, true, res.Ok)
//	assert.Equal(t, "false", string(res.Result))
//}

func servicePrepare(t *testing.T) (*ServiceManager, *mock_stub.MockStub, []*service_mgr.Service, [][]byte, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	sm := &ServiceManager{
		Stub: mockStub,
	}

	var services []*service_mgr.Service
	var servicesData [][]byte
	statusType := []governance.GovernanceStatus{governance.GovernanceAvailable, governance.GovernanceFrozen, governance.GovernanceUnavailable}

	for i := 0; i < 3; i++ {
		service := &service_mgr.Service{
			ChainID:           appchainAdminAddr,
			ServiceID:         serviceID,
			Name:              "name",
			Type:              service_mgr.ServiceCallContract,
			Intro:             "intro",
			Ordered:           true,
			Permission:        nil,
			Details:           "details",
			Score:             1,
			Status:            statusType[i],
			InvokeCount:       0,
			InvokeSuccessRate: 0,
			InvokeRecords:     nil,
			EvaluationRecords: map[string]*governance.EvaluationRecord{
				ownerAddr: &governance.EvaluationRecord{
					Addr:       ownerAddr,
					Score:      1,
					Desc:       "",
					CreateTime: 0,
				},
			},
		}

		data, err := json.Marshal(service)
		assert.Nil(t, err)

		services = append(services, service)
		servicesData = append(servicesData, data)
	}

	// prepare role
	var rolesData [][]byte
	role1 := &Role{
		ID: appchainAdminAddr,
	}
	data, err := json.Marshal(role1)
	assert.Nil(t, err)
	rolesData = append(rolesData, data)
	role2 := &Role{
		ID: noAdminAddr,
	}
	data, err = json.Marshal(role2)
	assert.Nil(t, err)
	rolesData = append(rolesData, data)

	return sm, mockStub, services, servicesData, rolesData
}
