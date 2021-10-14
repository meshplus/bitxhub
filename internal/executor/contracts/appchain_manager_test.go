package contracts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
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
	am, mockStub, chains, chainsData := prepareChain(t)

	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[0].ID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(appchainID2), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyNameKey(chains[0].ChainName), gomock.Any()).SetArg(1, chains[0].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	appchainsReq1 := mockStub.EXPECT().Query(appchainMgr.Prefix).Return(true, chainsData)
	appchainReq2 := mockStub.EXPECT().Query(appchainMgr.Prefix).Return(false, nil)
	counterAppchainReq := mockStub.EXPECT().Query(appchainMgr.Prefix).Return(true, chainsData).Times(2)
	gomock.InOrder(appchainsReq1, appchainReq2, counterAppchainReq)

	res := am.GetAppchain(chains[0].ID)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, chainsData[0], res.Result)

	res = am.GetAppchain(appchainID2)
	assert.Equal(t, false, res.Ok)

	res = am.Appchains()
	assert.Equal(t, true, res.Ok)

	var appchains []*appchainMgr.Appchain
	err := json.Unmarshal(res.Result, &appchains)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(appchains))
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
	assert.Equal(t, "6", string(res.Result))

	res = am.CountAvailableAppchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "2", string(res.Result))

	res = am.IsAvailable(chains[0].ID)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "true", string(res.Result))

	res = am.IsAvailable(appchainID2)
	assert.Equal(t, false, res.Ok)

	res = am.GetBitXHubChainIDs()
	assert.Equal(t, true, res.Ok)

	res = am.GetAppchainByName(chains[0].ChainName)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestAppchainManager_Register(t *testing.T) {
	am, mockStub, chains, _ := prepareChain(t)
	roles, _ := prepareRole(t)

	mockStub.EXPECT().GetTxHash().Return(types.NewHash(make([]byte, 0))).AnyTimes()
	mockStub.EXPECT().Caller().Return(roles[0].ID).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyNameKey(chains[1].ChainName), gomock.Any()).SetArg(1, chains[1].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(roles[1].ID), gomock.Any()).SetArg(1, chains[1].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	account := mockAccount(t)
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleByAddr",
		gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleByAddr",
		gomock.Any()).Return(boltvm.Success([]byte(AppchainAdmin))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleByAddr",
		gomock.Any()).Return(boltvm.Success([]byte(NoRole))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	// check broker error
	res := am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, nil, chains[0].Desc, ruleAddr, ruleUrl, roles[0].ID, reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, []byte("1"), chains[0].Desc, ruleAddr, ruleUrl, roles[0].ID, reason)
	assert.False(t, res.Ok, string(res.Result))

	// check name error
	res = am.RegisterAppchain("", chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, roles[0].ID, reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.RegisterAppchain(chains[1].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, roles[0].ID, reason)
	assert.False(t, res.Ok, string(res.Result))

	// check admin error
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, fmt.Sprintf("adminaddr,%s", roles[0].ID), reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, fmt.Sprintf("%s,%s", roles[0].ID, roles[1].ID), reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, fmt.Sprintf("%s,%s", roles[0].ID, roles[1].ID), reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, fmt.Sprintf("%s,%s", roles[0].ID, roles[1].ID), reason)
	assert.False(t, res.Ok, string(res.Result))

	// check rule error
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, "", fmt.Sprintf("%s,%s", roles[0].ID, roles[1].ID), reason)
	assert.False(t, res.Ok, string(res.Result))

	// submit proposal error
	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, roles[0].ID, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = am.RegisterAppchain(chains[0].ChainName, chains[0].ChainType, chains[0].TrustRoot, chains[0].Broker, chains[0].Desc, ruleAddr, ruleUrl, roles[0].ID, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_ManageRegister(t *testing.T) {
	am, mockStub, chains, chainsData := prepareChain(t)
	roles, _ := prepareRole(t)
	rules, _ := prepareRule(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("addrNoPermission").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[0].ID)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.ChainNumPrefix, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "UpdateAppchainAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Error("UpdateAppchainAdmin error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "UpdateAppchainAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "RegisterRuleFirst", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("RegisterRuleFirst error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "RegisterRuleFirst", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test without permission
	res := am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceAvailable), chains[0].ID, nil)
	assert.False(t, res.Ok, string(res.Result))

	// test register
	registerInfo := &RegisterAppchainInfo{
		ChainInfo:  chains[0],
		MasterRule: rules[0],
		AdminAddrs: roles[0].ID,
	}
	registerInfoData, err := json.Marshal(registerInfo)
	assert.Nil(t, err)
	// UpdateAppchainAdmin error
	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[0].ID, registerInfoData)
	assert.False(t, res.Ok, string(res.Result))
	// RegisterRuleFirst error
	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[0].ID, registerInfoData)
	assert.False(t, res.Ok, string(res.Result))
	// interchain register error
	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[0].ID, registerInfoData)
	assert.False(t, res.Ok, string(res.Result))
	// approve: ok
	res = am.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), chains[0].ID, registerInfoData)
	assert.True(t, res.Ok, string(res.Result))
	// reject: ok
	res = am.Manage(string(governance.EventRegister), string(REJECTED), string(governance.GovernanceUnavailable), chains[0].ID, registerInfoData)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_UpdateAppchain(t *testing.T) {
	am, mockStub, chains, chainsData := prepareChain(t)
	roles, _ := prepareRole(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Caller().Return(roles[0].ID).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(roles[1].ID).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(roles[0].ID).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[0].ID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[0].ID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyNameKey(chains[0].ChainName), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyNameKey(chains[1].ChainName), gomock.Any()).SetArg(1, chains[1].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(roles[0].ID), gomock.Any()).SetArg(1, chains[0].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(roles[1].ID), gomock.Any()).SetArg(1, chains[1].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainAdminKey(chains[0].ID), gomock.Any()).SetArg(1, []string{roles[0].ID}).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[0].ID)).Return(true, chainsData[0]).AnyTimes()

	// promission error
	res := am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, am.Caller(), reason)
	assert.False(t, res.Ok, string(res.Result))
	// governancePre error
	res = am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, am.Caller(), reason)
	assert.False(t, res.Ok, string(res.Result))
	// check name error
	res = am.UpdateAppchain(chains[0].ID, "", chains[0].Desc, chains[0].TrustRoot, am.Caller(), reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.UpdateAppchain(chains[0].ID, chains[1].ChainName, chains[0].Desc, chains[0].TrustRoot, am.Caller(), reason)
	assert.False(t, res.Ok, string(res.Result))
	// check admin error
	res = am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, "123", reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, fmt.Sprintf("123,%s", am.Caller()), reason)
	assert.False(t, res.Ok, string(res.Result))
	res = am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, fmt.Sprintf("%s,%s", roles[1].ID, am.Caller()), reason)
	assert.False(t, res.Ok, string(res.Result))
	// no proposal update
	res = am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, am.Caller(), reason)
	assert.True(t, res.Ok, string(res.Result))
	// submit proposal error
	res = am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, []byte("111"), am.Caller(), reason)
	assert.False(t, res.Ok, string(res.Result))
	// ok
	res = am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, am.Caller(), reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_ManageUpdate(t *testing.T) {
	am, mockStub, chains, chainsData := prepareChain(t)
	roles, _ := prepareRole(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[0].ID)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[1].ID)).Return(true, chainsData[1]).Times(1)
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[1].ID)).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[1].ID)).Return(true, chainsData[1]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.ChainNumPrefix, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[1].ID), gomock.Any()).SetArg(1, *chains[1]).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "UpdateAppchainAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Error("UpdateAppchainAdmin error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "UpdateAppchainAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test changestatus error
	res := am.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chains[0].ID, nil)
	assert.False(t, res.Ok, string(res.Result))

	// test update
	updateInfo := &UpdateAppchainInfo{
		Name: UpdateInfo{
			OldInfo: chains[0].ChainName,
			NewInfo: chains[1].ChainName,
			IsEdit:  true,
		},
		Desc: UpdateInfo{
			OldInfo: chains[0].Desc,
			NewInfo: chains[1].Desc,
			IsEdit:  true,
		},
		TrustRoot: UpdateInfo{
			OldInfo: chains[0].TrustRoot,
			NewInfo: chains[1].TrustRoot,
			IsEdit:  true,
		},
		AdminAddrs: UpdateInfo{
			OldInfo: roles[0].ID,
			NewInfo: roles[1].ID,
			IsEdit:  true,
		},
	}
	updateInfoData, err := json.Marshal(updateInfo)
	assert.Nil(t, err)
	// update error
	res = am.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chains[1].ID, updateInfoData)
	assert.False(t, res.Ok, string(res.Result))
	// UpdateAppchainAdmin error
	res = am.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chains[1].ID, updateInfoData)
	assert.False(t, res.Ok, string(res.Result))
	// approve ok
	res = am.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), chains[1].ID, updateInfoData)
	assert.True(t, res.Ok, string(res.Result))
	// reject ok
	res = am.Manage(string(governance.EventUpdate), string(REJECTED), string(governance.GovernanceAvailable), chains[1].ID, updateInfoData)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_ManageFreezeActivateLogout(t *testing.T) {
	am, mockStub, chains, chainsData := prepareChain(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[3].ID)).Return(true, chainsData[3]).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[4].ID)).Return(true, chainsData[4]).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[5].ID)).Return(true, chainsData[5]).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[3].ID), gomock.Any()).SetArg(1, *chains[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[4].ID), gomock.Any()).SetArg(1, *chains[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[5].ID), gomock.Any()).SetArg(1, *chains[5]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Error("PauseChainService error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", gomock.Any()).Return(boltvm.Error("UnPauseChainService error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "UnPauseRule", gomock.Any()).Return(boltvm.Error("UnPauseRule error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "UnPauseRule", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test freeze
	res := am.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), chains[3].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = am.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), chains[3].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// test activate
	res = am.Manage(string(governance.EventActivate), string(APPROVED), string(governance.GovernanceFrozen), chains[4].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = am.Manage(string(governance.EventActivate), string(APPROVED), string(governance.GovernanceFrozen), chains[4].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// test logout
	res = am.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceFrozen), chains[5].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = am.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceFrozen), chains[5].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_IsAvailable(t *testing.T) {
	am, mockStub, chains, _ := prepareChain(t)
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[0].ID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[1].ID), gomock.Any()).SetArg(1, *chains[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey("errId"), gomock.Any()).Return(false).AnyTimes()

	res := am.IsAvailable(chains[0].ID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	res = am.IsAvailable(chains[1].ID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	assert.Equal(t, "false", string(res.Result))
	res = am.IsAvailable("errId")
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestAppchainManager_FreezeActivateLogout(t *testing.T) {
	am, mockStub, chains, chainsData := prepareChain(t)
	roles, _ := prepareRole(t)
	_, rulesData := prepareRule(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Caller().Return(roles[0].ID).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(roles[0].ID).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[0].ID)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(chains[2].ID)).Return(true, chainsData[2]).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[0].ID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(chains[2].ID), gomock.Any()).SetArg(1, *chains[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(roles[0].ID), gomock.Any()).SetArg(1, chains[0].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(roles[0].ID), gomock.Any()).SetArg(1, chains[2].ID).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(appchainAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Error("get master rule error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Success(rulesData[0])).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Success(rulesData[1])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Error("pause chain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "PauseRule", gomock.Any()).Return(boltvm.Error("pause rule error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "PauseRule", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test FreezeAppchain
	res := am.FreezeAppchain(chains[0].ID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))

	// test ActivateAppchain failed, rule updating
	res = am.ActivateAppchain(chains[2].ID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = am.ActivateAppchain(chains[2].ID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test ActivateAppchain success
	res = am.ActivateAppchain(chains[2].ID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))

	// test LogoutAppchain, pause service error
	res = am.LogoutAppchain(chains[0].ID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test LogoutAppchain, pause rule error
	res = am.LogoutAppchain(chains[0].ID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test LogoutAppchain ok
	res = am.LogoutAppchain(chains[0].ID, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestManageChain_WithoutPermission(t *testing.T) {
	am, mockStub, chains, chainsData := prepareChain(t)
	_, rulesData := prepareRule(t)

	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(caller).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(caller), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(caller), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", gomock.Any()).Return(boltvm.Success(rulesData[1])).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(chains[0].ID, chains[0].ChainName, chains[0].Desc, chains[0].TrustRoot, am.Caller(), reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test FreezeAppchain
	res = am.FreezeAppchain(chains[0].ID, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain("addr", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestManageChain_baseGovernanceError(t *testing.T) {
	am, mockStub, chains, _ := prepareChain(t)
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

func TestAppchainManager_PauseAppchain(t *testing.T) {
	am, mockStub, chains, chainsData := prepareChain(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RuleManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Error("PauseChainService error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(appchainNotExistID), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(appchainAvailableID), gomock.Any()).SetArg(1, *chains[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(appchainAvailableID)).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(appchainAvailableID)).Return(true, chainsData[0]).AnyTimes()
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
	am, mockStub, chains, chainsData := prepareChain(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RuleManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", gomock.Any()).Return(boltvm.Error("PauseChainService error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(appchainNotExistID), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainKey(appchainFrozenID), gomock.Any()).SetArg(1, *chains[2]).Return(true).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(appchainFrozenID)).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(appchainMgr.AppchainKey(appchainFrozenID)).Return(true, chainsData[2]).AnyTimes()
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

func TestAppchainManager_checkPermission(t *testing.T) {
	am, mockStub, chains, _ := prepareChain(t)
	roles, _ := prepareRole(t)

	// 1. PermissionSelf
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(roles[0].ID), gomock.Any()).SetArg(1, chains[0].ID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(appchainMgr.AppchainOccupyAdminKey(roles[1].ID), gomock.Any()).SetArg(1, chains[1].ID).Return(true).AnyTimes()
	// normal
	err := am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, roles[0].ID, nil)
	assert.Nil(t, err)
	err = am.checkPermission([]string{string(PermissionSelf)}, chains[0].ID, roles[1].ID, nil)
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

func prepareChain(t *testing.T) (*AppchainManager, *mock_stub.MockStub, []*appchainMgr.Appchain, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	am := &AppchainManager{
		Stub: mockStub,
	}

	var chains []*appchainMgr.Appchain
	var chainsData [][]byte
	chainStatus := []string{string(governance.GovernanceAvailable), string(governance.GovernanceUpdating), string(governance.GovernanceFrozen), string(governance.GovernanceFreezing), string(governance.GovernanceActivating), string(governance.GovernanceLogouting)}

	fabricBroker := &appchainMgr.FabricBroker{
		ChannelID:     "channelID",
		ChaincodeID:   "chaincodeID",
		BrokerVersion: "brokerVersion",
	}
	fabricBrokerData, err := json.Marshal(fabricBroker)
	assert.Nil(t, err)
	for i := 0; i < 6; i++ {
		chain := &appchainMgr.Appchain{
			ChainName: fmt.Sprintf("应用链%d", i),
			ChainType: appchainMgr.ChainTypeFabric1_4_3,
			Status:    governance.GovernanceStatus(chainStatus[i]),
			ID:        fmt.Sprintf("%s%s", appchainID, types.NewAddress([]byte{byte(i)}).String()),
			TrustRoot: []byte(""),
			Broker:    fabricBrokerData,
			Desc:      "",
			Version:   0,
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
	}

	return am, mockStub, chains, chainsData
}

func prepareRule(t *testing.T) ([]*ruleMgr.Rule, [][]byte) {
	var rulesData [][]byte
	var rules []*ruleMgr.Rule
	rule1 := &ruleMgr.Rule{
		Address: ruleAddr,
		Status:  governance.GovernanceBindable,
	}
	rules = append(rules, rule1)
	data, err := json.Marshal(rule1)
	assert.Nil(t, err)
	rulesData = append(rulesData, data)
	rule2 := &ruleMgr.Rule{
		Address: ruleAddr,
		Status:  governance.GovernanceAvailable,
	}
	rules = append(rules, rule2)
	data, err = json.Marshal(rule2)
	assert.Nil(t, err)
	rulesData = append(rulesData, data)

	return rules, rulesData
}

func prepareRole(t *testing.T) ([]*Role, [][]byte) {
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

	return roles, rolesData
}
