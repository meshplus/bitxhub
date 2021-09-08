package contracts

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	rule_mgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

const (
	appchainID  = "appchain1"
	appchainID2 = "appchain2"
)

func TestAppchainManager_Query(t *testing.T) {
	am, mockStub, chains, chainsData, _, _, _ := prepare(t)

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
	appchains1 := []*appchainMgr.Appchain{}
	err = json.Unmarshal(res.Result, &appchains1)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(appchains1))

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
	am, mockStub, chains, _, roles, rolesData, rulesData := prepare(t)

	logger := log.NewWithModule("contracts")

	mockStub.EXPECT().Caller().Return(roles[0].ID).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Error("GetAppchainAdmin error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "RegisterRole", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("RegisterRole error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetRuleByAddr", gomock.Any(), gomock.Any()).Return(boltvm.Error("get rule by addr error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetRuleByAddr", gomock.Any(), gomock.Any()).Return(boltvm.Success(rulesData[1])).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetRuleByAddr", gomock.Any(), gomock.Any()).Return(boltvm.Success(rulesData[0])).AnyTimes()
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
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "BindFirstMasterRule", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// register role error
	res := am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, reason)
	assert.False(t, res.Ok, string(res.Result))
	// check permision error
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, reason)
	assert.False(t, res.Ok, string(res.Result))
	// governancePre error
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, reason)
	assert.False(t, res.Ok, string(res.Result))
	// check rule error
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, reason)
	assert.False(t, res.Ok, string(res.Result))
	// submit proposal error
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_Manage(t *testing.T) {
	am, mockStub, chains, chainsData, _, _, rulesData := prepare(t)

	mockStub.EXPECT().CurrentCaller().Return("addrNoPermission").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(AppchainKey(chains[1].ID)).Return(true, chainsData[1]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[2].ID)).Return(true, chainsData[2]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test without permission
	res := am.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chains[0].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// test changestatus error
	res = am.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chains[0].ID, nil)
	assert.False(t, res.Ok, string(res.Result))

	// test register, BindFirstMasterRule error
	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[2].ID, rulesData[1])
	assert.False(t, res.Ok, string(res.Result))
	// test register, interchain register error
	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[2].ID, rulesData[1])
	assert.False(t, res.Ok, string(res.Result))

	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[2].ID, rulesData[1])
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_IsAvailable(t *testing.T) {
	am, mockStub, chains, _, _, _, _ := prepare(t)
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
	am, mockStub, chains, chainsData, _, rolesData, rulesData := prepare(t)
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
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Success(rulesData[1])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "PauseRule", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	availableChain := chains[0]
	availableChain.Status = governance.GovernanceAvailable
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *availableChain).Return(true).Times(3)
	frozenChain := chains[1]
	frozenChain.Status = governance.GovernanceFrozen
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *frozenChain).Return(true).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(appchainID, chains[0].Desc)
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
	am, mockStub, chains, chainsData, _, rolesData, rulesData := prepare(t)
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Success(rulesData[1])).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(appchainID, chains[0].Desc)
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
	am, mockStub, chains, _, _, _, _ := prepare(t)
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
	am, mockStub, chains, _, _, rolesData, _ := prepare(t)

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

func prepare(t *testing.T) (*AppchainManager, *mock_stub.MockStub, []*appchainMgr.Appchain, [][]byte, []*Role, [][]byte, [][]byte) {
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
			Version:   0,
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
	}

	// prepare role
	var rolesData [][]byte
	var roles []*Role
	role1 := &Role{
		ID: appchainAdminAddr,
	}
	data, err := json.Marshal(role1)
	assert.Nil(t, err)
	rolesData = append(rolesData, data)
	roles = append(roles, role1)
	role2 := &Role{
		ID: noAdminAddr,
	}
	data, err = json.Marshal(role2)
	assert.Nil(t, err)
	rolesData = append(rolesData, data)
	roles = append(roles, role2)

	// prepare rule
	var rulesData [][]byte
	rule1 := &rule_mgr.Rule{
		Address: ruleAddr,
		Status:  governance.GovernanceBindable,
	}
	data, err = json.Marshal(rule1)
	assert.Nil(t, err)
	rulesData = append(rulesData, data)
	rule2 := &rule_mgr.Rule{
		Address: ruleAddr,
		Status:  governance.GovernanceAvailable,
	}
	data, err = json.Marshal(rule2)
	assert.Nil(t, err)
	rulesData = append(rulesData, data)

	return am, mockStub, chains, chainsData, roles, rolesData, rulesData
}
