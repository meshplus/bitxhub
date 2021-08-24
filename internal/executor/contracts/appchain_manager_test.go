package contracts

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/meshplus/bitxhub-model/pb"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/types"

	"github.com/stretchr/testify/assert"
)

const (
	appchainID  = "appchain1"
	appchainID2 = "appchain2"
)

func TestAppchainManager_Query(t *testing.T) {
	am, mockStub, chains, chainsData, _ := prepare(t)

	mockStub.EXPECT().GetObject(AppchainKey(appchainID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey(appchainID2), gomock.Any()).Return(false).AnyTimes()
	appchainsReq1 := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(true, chainsData)
	appchainReq2 := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(false, nil)
	counterAppchainReq := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(true, chainsData).Times(2)
	gomock.InOrder(appchainsReq1, appchainReq2, counterAppchainReq)

	res := am.GetAppchain(appchainID)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, chainsData[0], res.Result)

	res = am.GetAppchain(appchainID2)
	assert.Equal(t, false, res.Ok)

	res = am.Appchains()
	assert.Equal(t, true, res.Ok)

	var appchains []*appchainMgr.Appchain
	err := json.Unmarshal(res.Result, &appchains)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(appchains))
	assert.Equal(t, chains[0], appchains[0])
	assert.Equal(t, chains[1], appchains[1])
	assert.Equal(t, chains[2], appchains[2])

	res = am.Appchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, []byte(nil), res.Result)

	res = am.CountAppchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "4", string(res.Result))

	res = am.CountAvailableAppchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "1", string(res.Result))

	res = am.IsAvailable(appchainID)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "true", string(res.Result))

	res = am.IsAvailable(appchainID2)
	assert.Equal(t, false, res.Ok)
}

func TestAppchainManager_Register(t *testing.T) {
	am, mockStub, chains, _, _ := prepare(t)

	logger := log.NewWithModule("contracts")

	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "RegisterRole", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "RegisterRole", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	registerErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			chain := ret.(*appchainMgr.Appchain)
			chain.ID = chains[2].ID
			chain.Status = chains[2].Status
			return true
		}).Return(true).Times(1)
	registerReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	gomock.InOrder(registerErrReq, registerReq)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	// RegisterRole error
	res := am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, chains[0].Version, FALSE, reason)
	assert.False(t, res.Ok, string(res.Result))
	// governancePre error
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, chains[0].Version, FALSE, reason)
	assert.False(t, res.Ok, string(res.Result))
	// submit proposal error
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, chains[0].Version, FALSE, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, chains[0].Version, FALSE, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_Manage(t *testing.T) {
	am, mockStub, chains, chainsData, _ := prepare(t)

	mockStub.EXPECT().CurrentCaller().Return("addrNoPermission").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(AppchainKey(chains[1].ID)).Return(true, chainsData[1]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[2].ID)).Return(true, chainsData[2]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	defaultRuleErr1 := mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "DefaultRule", gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	defaultRuleOk2 := mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "DefaultRule", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).Times(1)
	defaultRuleErr2 := mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "DefaultRule", gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	defaultRuleOk3 := mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "DefaultRule", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	gomock.InOrder(defaultRuleErr1, defaultRuleOk2, defaultRuleErr2, defaultRuleOk3)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test without permission
	res := am.Manage(string(governance.EventUpdate), string(APPOVED), string(governance.GovernanceAvailable), chains[0].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// test changestatus error
	res = am.Manage(string(governance.EventUpdate), string(APPOVED), string(governance.GovernanceAvailable), chains[0].ID, nil)
	assert.False(t, res.Ok, string(res.Result))

	// test update
	res = am.Manage(string(governance.EventUpdate), string(APPOVED), string(governance.GovernanceAvailable), chains[1].ID, chainsData[1])
	assert.True(t, res.Ok, string(res.Result))

	// test register with default rule, bind default rule error1
	res = am.Manage(string(governance.EventRegister), string(APPOVED), string(governance.GovernanceUnavailable), chains[2].ID, []byte(strconv.FormatBool(true)))
	assert.False(t, res.Ok, string(res.Result))
	// test register with default rule, bind default rule error2
	res = am.Manage(string(governance.EventRegister), string(APPOVED), string(governance.GovernanceUnavailable), chains[2].ID, []byte(strconv.FormatBool(true)))
	assert.False(t, res.Ok, string(res.Result))
	// test register with default rule, interchain register error
	res = am.Manage(string(governance.EventRegister), string(APPOVED), string(governance.GovernanceUnavailable), chains[2].ID, []byte(strconv.FormatBool(true)))
	assert.False(t, res.Ok, string(res.Result))
	// test register with default rule, interchain register ok
	res = am.Manage(string(governance.EventRegister), string(APPOVED), string(governance.GovernanceUnavailable), chains[2].ID, []byte(strconv.FormatBool(true)))
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_IsAvailable(t *testing.T) {
	am, mockStub, chains, _, _ := prepare(t)
	mockStub.EXPECT().GetObject(AppchainKey(chains[0].ID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey(chains[1].ID), gomock.Any()).SetArg(1, *chains[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey("errId"), gomock.Any()).Return(false).AnyTimes()

	res := am.IsAvailable(chains[0].ID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	res = am.IsAvailable(chains[1].ID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	assert.Equal(t, "false", string(res.Result))
	res = am.IsAvailable("errId")
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestManageChain(t *testing.T) {
	am, mockStub, chains, chainsData, rolesData := prepare(t)
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(appchainID)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(appchainID2)).Return(true, chainsData[3]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(appchainAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()

	availableChain := chains[0]
	availableChain.Status = governance.GovernanceAvailable
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *availableChain).Return(true).Times(2)
	frozenChain := chains[1]
	frozenChain.Status = governance.GovernanceFrozen
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *frozenChain).Return(true).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(appchainID, chains[0].Desc, chains[0].Version)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test FreezeAppchain
	res = am.FreezeAppchain(appchainID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain(appchainID2, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test LogoutAppchain
	res = am.LogoutAppchain(appchainID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestManageChain_WithoutPermission(t *testing.T) {
	am, mockStub, chains, chainsData, rolesData := prepare(t)
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, chainsData[0]).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(appchainID, chains[0].Desc, chains[0].Version)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test FreezeAppchain
	res = am.FreezeAppchain("addr", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = am.FreezeAppchain("addr", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain("addr", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestManageChain_Error(t *testing.T) {
	am, mockStub, chains, _, _ := prepare(t)
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	availableChain := chains[0]
	availableChain.Status = governance.GovernanceAvailable
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *availableChain).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// governance pre error
	res := am.FreezeAppchain(caller, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// submit proposal error
	res = am.FreezeAppchain(caller, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// change status error
	res = am.FreezeAppchain(caller, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestAppchainManager_checkPermission(t *testing.T) {
	am, mockStub, chains, _, rolesData := prepare(t)

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	err := am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, appchainAdminAddr, nil)
	assert.Nil(t, err)
	err = am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, noAdminAddr, nil)
	assert.NotNil(t, err)

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	err = am.checkPermission([]string{string(PermissionAdmin)}, chains[0].ID, adminAddr, nil)
	assert.Nil(t, err)
	err = am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, noAdminAddr, nil)
	assert.NotNil(t, err)

	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	err = am.checkPermission([]string{string(PermissionSpecific)}, "", constant.GovernanceContractAddr.Address().String(), addrsData)
	assert.Nil(t, err)
	err = am.checkPermission([]string{string(PermissionSpecific)}, "", noAdminAddr, addrsData)
	assert.NotNil(t, err)

	err = am.checkPermission([]string{""}, "", "", nil)
	assert.NotNil(t, err)
}

func prepare(t *testing.T) (*AppchainManager, *mock_stub.MockStub, []*appchainMgr.Appchain, [][]byte, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	am := &AppchainManager{
		Stub: mockStub,
	}

	var chains []*appchainMgr.Appchain
	var chainsData [][]byte
	chainStatus := []string{string(governance.GovernanceAvailable), string(governance.GovernanceUpdating), string(governance.GovernanceRegisting), string(governance.GovernanceFrozen)}

	for i := 0; i < 4; i++ {
		addr := appchainID + types.NewAddress([]byte{byte(i)}).String()

		chain := &appchainMgr.Appchain{
			Status:    governance.GovernanceStatus(chainStatus[i]),
			ID:        addr,
			TrustRoot: nil,
			Broker:    "",
			Desc:      "",
			Version:   "",
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
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

	return am, mockStub, chains, chainsData, rolesData
}
