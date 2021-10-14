package contracts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/iancoleman/orderedmap"
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

func TestServiceManager_Manage(t *testing.T) {
	sm, mockStub, services, _, _ := servicePrepare(t)
	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)
	chainServiceRegisteringID := fmt.Sprintf("%s:%s", services[4].ChainID, services[4].ServiceID)
	chainServiceUpdatingID := fmt.Sprintf("%s:%s", services[5].ChainID, services[5].ServiceID)
	chainServiceLogoutingID := fmt.Sprintf("%s:%s", services[6].ChainID, services[6].ServiceID)
	logger := log.NewWithModule("contracts")

	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceID), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceRegisteringID), gomock.Any()).SetArg(1, *services[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceUpdatingID), gomock.Any()).SetArg(1, *services[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceLogoutingID), gomock.Any()).SetArg(1, *services[6]).Return(true).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("addrNoPermission").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Error("crossinvoke isavailable error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte(FALSE))).AnyTimes()

	// test without permission
	res := sm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chainServiceID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// test changestatus error
	res = sm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chainServiceID, nil)
	assert.False(t, res.Ok, string(res.Result))

	// test register, register error
	res = sm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chainServiceRegisteringID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// test register, ok
	res = sm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chainServiceRegisteringID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// test update
	updateInfo := &UpdateServiceInfo{
		ServiceName: UpdateInfo{
			OldInfo: services[0].Name,
			NewInfo: services[1].Name,
			IsEdit:  true,
		},
		Intro: UpdateInfo{
			OldInfo: services[0].Intro,
			NewInfo: services[1].Intro,
			IsEdit:  true,
		},
		Details: UpdateInfo{
			OldInfo: services[0].Details,
			NewInfo: services[1].Details,
			IsEdit:  true,
		},
		Permission: UpdateMapInfo{
			OldInfo: services[0].Permission,
			NewInfo: services[0].Permission,
			IsEdit:  false,
		},
	}
	updateInfoData, err := json.Marshal(updateInfo)
	assert.Nil(t, err)
	res = sm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chainServiceUpdatingID, updateInfoData)
	assert.True(t, res.Ok, string(res.Result))

	// test logout, isavailable error
	res = sm.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceUnavailable), chainServiceLogoutingID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// test logout, ok
	res = sm.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceUnavailable), chainServiceLogoutingID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestServiceManager_RegisterService(t *testing.T) {
	sm, mockStub, services, _, rolesData := servicePrepare(t)

	mockStub.EXPECT().GetObject(service_mgr.ServiceOccupyNameKey(services[0].Name), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceOccupyNameKey(services[1].Name), gomock.Any()).SetArg(1, services[1].ServiceID).Return(true).AnyTimes()

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()

	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	checkAppchainErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(1)
	checkInfoErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[2]).Return(true).Times(4)
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
	// name
	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, "", string(services[0].Type), services[0].Intro, services[0].Ordered, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[1].Name, string(services[0].Type), services[0].Intro, services[0].Ordered, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// type
	res = sm.RegisterService(services[0].ChainID, services[0].ServiceID, services[1].Name, "123", services[0].Intro, services[0].Ordered, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// permission
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

	mockStub.EXPECT().GetObject(service_mgr.ServiceOccupyNameKey(services[0].Name), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceOccupyNameKey("newName"), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceOccupyNameKey(services[1].Name), gomock.Any()).SetArg(1, services[1].ServiceID).Return(true).AnyTimes()

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	checkPermissionErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	checkAppchainErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	checkInfoErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(3)
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
	res := sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 2. check permision error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 3. check appchain error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	// 4. check info error
	// name
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), "", services[0].Intro, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[1].Name, services[0].Intro, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// permission
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, "00", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	// 5. update error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), services[0].Name, services[0].Intro, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 6. submit error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), "newName", services[0].Intro, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 7. change status error
	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), "newName", services[0].Intro, "", services[0].Details, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.UpdateService(fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID), "newName", services[0].Intro, "", services[0].Details, reason)
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

func TestServiceManager_PauseChainService(t *testing.T) {
	sm, mockStub, services, _, _ := servicePrepare(t)
	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)
	chainServiceunavailableID := fmt.Sprintf("%s:%s", services[2].ChainID, services[2].ServiceID)

	serviceMap1 := orderedmap.New()
	serviceMap1.Set(chainServiceunavailableID, struct{}{})
	serviceMap2 := orderedmap.New()
	serviceMap2.Set(chainServiceID, struct{}{})
	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[0].ChainID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[0].ChainID), gomock.Any()).SetArg(1, *serviceMap1).Return(true).Times(1)
	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[0].ChainID), gomock.Any()).SetArg(1, *serviceMap2).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceunavailableID), gomock.Any()).SetArg(1, *services[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceID), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Error("LockLowPriorityProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	mockStub.EXPECT().CurrentCaller().Return("").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()

	// check permission error
	res := sm.PauseChainService(services[0].ChainID)
	assert.Equal(t, false, res.Ok, string(res.Result))

	// no services list, ok
	res = sm.PauseChainService(services[0].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// no service can be paused, ok
	res = sm.PauseChainService(services[0].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))

	// LockLowPriorityProposal error
	res = sm.PauseChainService(services[0].ChainID)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.PauseChainService(services[0].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_UnPauseChainService(t *testing.T) {
	sm, mockStub, services, _, _ := servicePrepare(t)
	chainServiceID := fmt.Sprintf("%s:%s", services[3].ChainID, services[3].ServiceID)
	chainServiceunavailableID := fmt.Sprintf("%s:%s", services[2].ChainID, services[2].ServiceID)

	serviceMap1 := orderedmap.New()
	serviceMap1.Set(chainServiceunavailableID, struct{}{})
	serviceMap2 := orderedmap.New()
	serviceMap2.Set(chainServiceID, struct{}{})
	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[3].ChainID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[3].ChainID), gomock.Any()).SetArg(1, *serviceMap1).Return(true).Times(1)
	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[3].ChainID), gomock.Any()).SetArg(1, *serviceMap2).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceunavailableID), gomock.Any()).SetArg(1, *services[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceID), gomock.Any()).SetArg(1, *services[3]).Return(true).AnyTimes()

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	mockStub.EXPECT().CurrentCaller().Return("").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Error("lock error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// check permission error
	res := sm.UnPauseChainService(services[3].ChainID)
	assert.Equal(t, false, res.Ok, string(res.Result))

	// no services list, ok
	res = sm.UnPauseChainService(services[3].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// no service can be paused, ok
	res = sm.UnPauseChainService(services[3].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))

	// LockLowPriorityProposal error
	res = sm.UnPauseChainService(services[3].ChainID)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.UnPauseChainService(services[3].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_EvaluateService(t *testing.T) {
	sm, mockStub, services, _, _ := servicePrepare(t)
	chainServiceID := fmt.Sprintf("%s:%s", services[3].ChainID, services[3].ServiceID)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Caller().Return(ownerAddr).Times(1)
	mockStub.EXPECT().Caller().Return(ownerAddr1).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()

	// 1. illegal score
	res := sm.EvaluateService(chainServiceID, "good service", 6)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 2. get service error
	res = sm.EvaluateService(chainServiceID, "bad service", 1)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 3. has evaluated
	res = sm.EvaluateService(chainServiceID, "bad service", 1)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.EvaluateService(chainServiceID, "bad service", 1)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_RecordInvokeService(t *testing.T) {
	sm, mockStub, services, _, _ := servicePrepare(t)
	fullServiceID := fmt.Sprintf("1356:%s:%s", services[0].ChainID, services[0].ServiceID)
	fromFullServiceID := fmt.Sprintf("1356:%s:%s", services[1].ChainID, services[1].ServiceID)
	//chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)
	logger := log.NewWithModule("contracts")

	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.InterchainContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	// 1. permission error
	res := sm.RecordInvokeService(fullServiceID, fromFullServiceID, true)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 2. get service error
	res = sm.RecordInvokeService(fullServiceID, fromFullServiceID, true)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = sm.RecordInvokeService(fullServiceID, fromFullServiceID, true)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestServiceManager_Query(t *testing.T) {
	sm, mockStub, services, servicesData, _ := servicePrepare(t)
	chainServiceID := fmt.Sprintf("%s:%s", services[0].ChainID, services[0].ServiceID)

	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(service_mgr.ServiceKey(chainServiceID), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	// get error
	res := sm.GetServiceInfo(chainServiceID)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// ok
	res = sm.GetServiceInfo(chainServiceID)
	assert.Equal(t, true, res.Ok, string(res.Result))

	mockStub.EXPECT().Query(service_mgr.ServicePrefix).Return(true, servicesData).AnyTimes()
	// ok
	res = sm.GetAllServices()
	assert.Equal(t, true, res.Ok, string(res.Result))
	theServices := []*service_mgr.Service{}
	err := json.Unmarshal(res.Result, &theServices)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(theServices))

	res = sm.GetPermissionServices(chainServiceID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, &theServices)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(theServices))

	serviceMap := orderedmap.New()
	serviceMap.Set(chainServiceID, struct{}{})

	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[0].ChainID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(service_mgr.AppchainServicesKey(services[0].ChainID), gomock.Any()).SetArg(1, *serviceMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(serviceKey(chainServiceID), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	// 0
	res = sm.GetServicesByAppchainID(services[0].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, &theServices)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(theServices))
	// 1
	res = sm.GetServicesByAppchainID(services[0].ChainID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, &theServices)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(theServices))

	mockStub.EXPECT().GetObject(service_mgr.ServicesTypeKey(string(services[0].Type)), gomock.Any()).SetArg(1, *serviceMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(serviceKey(chainServiceID), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	res = sm.GetServicesByType(string(services[0].Type))
	assert.Equal(t, true, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, &theServices)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(theServices))

	mockStub.EXPECT().GetObject(serviceKey(chainServiceID), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(serviceKey(chainServiceID), gomock.Any()).SetArg(1, *services[1]).Return(true).Times(1)
	res = sm.IsAvailable(chainServiceID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))
	res = sm.IsAvailable(chainServiceID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	assert.Equal(t, "false", string(res.Result))
}

func TestServiceManager_CheckServiceIDFormat(t *testing.T) {
	sm, mockStub, services, _, _ := servicePrepare(t)

	err := sm.checkServiceIDFormat("11")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "the ID does not contain three parts")

	err = sm.checkServiceIDFormat("::")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "BitxhubID is empty")

	err = sm.checkServiceIDFormat("1:1:")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "AppchainID or ServiceID is empty")

	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "GetBitXHubID").Return(boltvm.Error("error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "GetBitXHubID").Return(boltvm.Success([]byte("1356"))).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *services[0]).Return(true).Times(1)

	err = sm.checkServiceIDFormat("1:1:1")
	assert.NotNil(t, err)
	err = sm.checkServiceIDFormat("1356:1:1")
	assert.NotNil(t, err)
	err = sm.checkServiceIDFormat("1356:1:1")
	assert.Nil(t, err)
}

func servicePrepare(t *testing.T) (*ServiceManager, *mock_stub.MockStub, []*service_mgr.Service, [][]byte, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	sm := &ServiceManager{
		Stub: mockStub,
	}

	var services []*service_mgr.Service
	var servicesData [][]byte
	statusType := []governance.GovernanceStatus{
		governance.GovernanceAvailable,
		governance.GovernanceFrozen,
		governance.GovernanceUnavailable,
		governance.GovernancePause,
		governance.GovernanceRegisting,
		governance.GovernanceUpdating,
		governance.GovernanceLogouting,
	}

	for i := 0; i < 7; i++ {
		service := &service_mgr.Service{
			ChainID:           appchainAdminAddr,
			ServiceID:         fmt.Sprintf("%s%d", serviceID[:len(serviceID)-2], i),
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
			InvokeRecords:     make(map[string]*governance.InvokeRecord),
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
