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
	appchainID          = "appchain1"
	appchainID2         = "appchain2"
	appchainNotExistID  = "appchainNotExist"
	appchainAvailableID = "appchainAvailable"
	appchainFrozenID    = "appchainFrozen"
)

func TestAppchainManager_Query(t *testing.T) {
	am, mockStub, chains, chainsData, _, _, _ := prepare(t)

	mockStub.EXPECT().GetObject(AppchainKey(appchainID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey(appchainID2), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
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
	assert.Equal(t, 7, len(appchains))
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
	assert.Equal(t, "7", string(res.Result))

	res = am.CountAvailableAppchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "2", string(res.Result))

	res = am.IsAvailable(appchainID)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "true", string(res.Result))

	res = am.IsAvailable(appchainID2)
	assert.Equal(t, false, res.Ok)

	res = am.GetBitXHubChainIDs()
	assert.Equal(t, true, res.Ok)
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
	// register relay chain
	res = am.RegisterAppchain(appchainID, chains[0].TrustRoot, string(constant.InterBrokerContractAddr), chains[0].Desc, ruleAddr, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_PauseAppchain(t *testing.T) {
	am, mockStub, chains, chainsData, _, _, _ := prepare(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RuleManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Error("PauseChainService error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey(appchainNotExistID), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey(appchainAvailableID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(appchainAvailableID)).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(AppchainKey(appchainAvailableID)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	// check permision error
	res := am.PauseAppchain(appchainID)
	assert.False(t, res.Ok, string(res.Result))
	// governancePre error
	res = am.PauseAppchain(appchainNotExistID)
	assert.False(t, res.Ok, string(res.Result))
	// changeStatus error
	res = am.PauseAppchain(appchainAvailableID)
	assert.False(t, res.Ok, string(res.Result))
	// PauseChainService error
	res = am.PauseAppchain(appchainAvailableID)
	assert.False(t, res.Ok, string(res.Result))

	res = am.PauseAppchain(appchainAvailableID)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_UnPauseAppchain(t *testing.T) {
	am, mockStub, chains, chainsData, _, _, _ := prepare(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RuleManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", gomock.Any()).Return(boltvm.Error("PauseChainService error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey(appchainNotExistID), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(AppchainKey(appchainFrozenID), gomock.Any()).SetArg(1, *chains[3]).Return(true).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(appchainFrozenID)).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(AppchainKey(appchainFrozenID)).Return(true, chainsData[3]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	// check permision error
	res := am.UnPauseAppchain(appchainID, string(governance.GovernanceAvailable))
	assert.False(t, res.Ok, string(res.Result))
	// governancePre error
	res = am.UnPauseAppchain(appchainNotExistID, string(governance.GovernanceAvailable))
	assert.False(t, res.Ok, string(res.Result))
	// changeStatus error
	res = am.UnPauseAppchain(appchainFrozenID, string(governance.GovernanceAvailable))
	assert.False(t, res.Ok, string(res.Result))
	// UnPauseChainService error
	res = am.UnPauseAppchain(appchainFrozenID, string(governance.GovernanceAvailable))
	assert.False(t, res.Ok, string(res.Result))

	res = am.UnPauseAppchain(appchainFrozenID, string(governance.GovernanceAvailable))
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_Manage(t *testing.T) {
	am, mockStub, chains, chainsData, _, _, rulesData := prepare(t)

	mockStub.EXPECT().CurrentCaller().Return("addrNoPermission").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(AppchainKey(chains[0].ID)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[1].ID)).Return(true, chainsData[1]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[2].ID)).Return(true, chainsData[2]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[3].ID)).Return(true, chainsData[3]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[4].ID)).Return(true, chainsData[4]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[5].ID)).Return(true, chainsData[5]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[6].ID)).Return(true, chainsData[6]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "UnPauseRule", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

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
	// test register approve
	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[2].ID, rulesData[1])
	assert.True(t, res.Ok, string(res.Result))
	// test register reject
	res = am.Manage(string(governance.EventRegister), string(REJECTED), string(governance.GovernanceUnavailable), chains[2].ID, rulesData[1])
	assert.True(t, res.Ok, string(res.Result))

	// test freeze
	res = am.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), chains[4].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// test activate
	res = am.Manage(string(governance.EventActivate), string(APPROVED), string(governance.GovernanceFrozen), chains[5].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// test logout
	res = am.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceFrozen), chains[6].ID, nil)
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
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Error("get master rule error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Success(rulesData[0])).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Success(rulesData[1])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Error("pause chain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "PauseRule", gomock.Any()).Return(boltvm.Error("pause rule error")).Times(1)
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

	// test ActivateAppchain failed, rule updating
	res = am.ActivateAppchain(appchainID2, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = am.ActivateAppchain(appchainID2, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test ActivateAppchain success
	res = am.ActivateAppchain(appchainID2, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))

	// test LogoutAppchain, pause service error
	res = am.LogoutAppchain(appchainID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test LogoutAppchain, pause rule error
	res = am.LogoutAppchain(appchainID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test LogoutAppchain ok
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

func TestManageChain_baseGovernanceError(t *testing.T) {
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

	// 1. PermissionSelf
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Error("getAppchainAdmin error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	//getAppchainAdmin error
	err := am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, appchainAdminAddr, nil)
	assert.NotNil(t, err)
	// normal
	err = am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, appchainAdminAddr, nil)
	assert.Nil(t, err)
	err = am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, noAdminAddr, nil)
	assert.NotNil(t, err)

	// 2. PermissionAdmin
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Error("invoke error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	// crossinvoke error
	err = am.checkPermission([]string{string(PermissionAdmin)}, chains[0].ID, adminAddr, nil)
	assert.NotNil(t, err)
	// normal
	err = am.checkPermission([]string{string(PermissionAdmin)}, chains[0].ID, adminAddr, nil)
	assert.Nil(t, err)
	err = am.checkPermission([]string{string(PermissionAdmin)}, chains[0].ID, noAdminAddr, nil)
	assert.NotNil(t, err)

	// 3. PermissionSpecific
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	// unmarshal error
	err = am.checkPermission([]string{string(PermissionSpecific)}, "", constant.GovernanceContractAddr.Address().String(), []byte("str"))
	assert.NotNil(t, err)
	// normal
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
	chainStatus := []string{string(governance.GovernanceAvailable), string(governance.GovernanceUpdating), string(governance.GovernanceRegisting), string(governance.GovernanceFrozen), string(governance.GovernanceFreezing), string(governance.GovernanceActivating), string(governance.GovernanceLogouting)}

	for i := 0; i < 7; i++ {
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
