package contracts

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-kit/bytesutil"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/storage/blockfile"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/repo"
	ledger2 "github.com/meshplus/eth-kit/ledger"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	adminAddr         = "adminAddr"
	appchainAdminAddr = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8"
	noAdminAddr       = "0xc0Ff2e0b3189132D815b8eb325bE17285AC89999"
	ruleAddr          = "0xc0Ff2e0b3189132D815b8eb325bE17285AC12344"
	ruleUrl           = "ruleUrl"
)

func TestRuleManager_Manage(t *testing.T) {
	rm, mockStub, rules, _, _, _, _, rolesData := rulePrepare(t)
	updateExtraInfo0 := UpdateMasterRuleInfo{
		OldRule: rules[0],
		NewRule: rules[1],
		AppchainInfo: &appchainMgr.Appchain{
			ID: rules[1].ChainID,
		},
	}
	updateExtraInfoData0, err := json.Marshal(updateExtraInfo0)
	assert.Nil(t, err)

	updateExtraInfo1 := UpdateMasterRuleInfo{
		OldRule: rules[2],
		NewRule: rules[1],
		AppchainInfo: &appchainMgr.Appchain{
			ID: rules[1].ChainID,
		},
	}
	updateExtraInfoData1, err := json.Marshal(updateExtraInfo1)
	assert.Nil(t, err)

	updateExtraInfo3 := UpdateMasterRuleInfo{
		OldRule: rules[0],
		NewRule: rules[3],
		AppchainInfo: &appchainMgr.Appchain{
			ID: rules[3].ChainID,
		},
	}
	updateExtraInfoData3, err := json.Marshal(updateExtraInfo3)
	assert.Nil(t, err)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "UnPauseAppchain", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	changeMasterErrReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, []*ruleMgr.Rule{rules[1]}).Return(true).Times(1)

	rules[0].Status = governance.GovernanceUnbinding
	changeStatusErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, []*ruleMgr.Rule{rules[0]}).Return(true).Times(2)

	rules[2].Status = governance.GovernanceUnbinding
	okReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, []*ruleMgr.Rule{rules[2]}).Return(true).Times(1)
	okReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, []*ruleMgr.Rule{rules[1]}).Return(true).Times(1)
	gomock.InOrder(changeMasterErrReq1, changeStatusErrReq, okReq1, okReq2)

	// check permission error
	res := rm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceBindable), rules[1].Address, updateExtraInfoData3)
	assert.False(t, res.Ok, string(res.Result))
	// change master error
	res = rm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceBindable), rules[1].Address, updateExtraInfoData0)
	assert.False(t, res.Ok, string(res.Result))
	// change status error
	res = rm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceBindable), rules[1].Address, updateExtraInfoData0)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceBindable), rules[1].Address, updateExtraInfoData1)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_RegisterRule(t *testing.T) {
	rm, mockStub, rules, _, chains, chainsData, account, _ := rulePrepare(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Caller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", gomock.Any()).Return(boltvm.Error("GetAppchain error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", gomock.Any()).Return(boltvm.Success(chainsData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Error("GetAppchainAdmin error")).Times(1)
	appchainRole := &Role{
		ID:         appchainAdminAddr,
		RoleType:   AppchainAdmin,
		Weight:     0,
		NodePid:    "",
		AppchainID: chains[0].ID,
		Status:     governance.GovernanceAvailable,
		FSM:        nil,
	}
	appchainRoleData, err := json.Marshal(appchainRole)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(appchainRoleData)).AnyTimes()
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()
	retRepeatedRegister := make([]*ruleMgr.Rule, 0)
	retRepeatedRegister = append(retRepeatedRegister, rules[0])
	governancePreErrReq := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRepeatedRegister).Return(true).Times(1)
	OkReq := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).AnyTimes()
	gomock.InOrder(governancePreErrReq, OkReq)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// check permission error
	res := rm.RegisterRule(chains[0].ID, rules[0].Address, rules[0].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	// check rule error
	res = rm.RegisterRule(chains[0].ID, rules[0].Address, rules[0].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	// GovernancePre error
	res = rm.RegisterRule(chains[0].ID, rules[0].Address, rules[0].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.RegisterRule(chains[0].ID, rules[3].Address, rules[0].RuleUrl)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_RegisterRuleFirst(t *testing.T) {
	rm, mockStub, rules, _, chains, _, _, _ := rulePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return("").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	// fabric  5 * 2
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(10)
	// othertype 3 * 8
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(24)
	retRulesBindable := make([]*ruleMgr.Rule, 0)
	retRulesBindable = append(retRulesBindable, rules[3])
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).AnyTimes()

	// check permission error
	res := rm.RegisterRuleFirst(chains[0].ID, chains[0].ChainType, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))

	// change status error
	res = rm.RegisterRuleFirst(chains[0].ID, chains[0].ChainType, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeFabric1_4_4, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeHyperchain1_8_3, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeHyperchain1_8_6, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeFlato1_0_0, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeFlato1_0_3, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeFlato1_0_6, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeBCOS2_6_0, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeCITA20_2_2, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.RegisterRuleFirst(chains[0].ID, appchainMgr.ChainTypeETH, rules[3].Address, rules[3].RuleUrl)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.RegisterRuleFirst(chains[0].ID, chains[0].ChainType, rules[3].Address, rules[3].RuleUrl)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_UpdateMasterRule(t *testing.T) {
	rm, mockStub, rules, _, chains, chainsData, account, rolesData := rulePrepare(t)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "PauseAppchain", gomock.Any()).Return(boltvm.Error("pause error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "PauseAppchain", gomock.Any()).Return(boltvm.Success(chainsData[0])).AnyTimes()

	governancePreErrReq1 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesBindable := make([]*ruleMgr.Rule, 0)
	retRulesBindable = append(retRulesBindable, rules[3])
	updatePreOkReq2 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).Times(1)
	getMasterErrReq2 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).Times(1)

	updatePreOkReq3 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).Times(1)
	retRulesMaster := make([]*ruleMgr.Rule, 0)
	retRulesMaster = append(retRulesMaster, rules[2])
	getMasterOKReq3 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesMaster).Return(true).Times(1)

	updatePreOkReq4 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).Times(1)
	getMasterOKReq4 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesMaster).Return(true).Times(1)
	changeStatusErrReq4 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesBindable2 := make([]*ruleMgr.Rule, 0)
	retRulesBindable2 = append(retRulesBindable2, rules[4])
	updatePreOkReq5 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable2).Return(true).Times(1)
	getMasterOKReq5 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesMaster).Return(true).Times(1)
	changeStatusOKReq5 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable2).Return(true).Times(1)
	changeMasterErrReq5 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)

	retRulesBindable3 := make([]*ruleMgr.Rule, 0)
	retRulesBindable3 = append(retRulesBindable3, rules[5])
	updatePreOkReq6 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable3).Return(true).Times(1)
	getMasterOKReq6 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesMaster).Return(true).Times(1)
	changeStatusOKReq6 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable3).Return(true).Times(1)
	changeMasterOKReq6 := mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesMaster).Return(true).Times(1)
	gomock.InOrder(governancePreErrReq1,
		updatePreOkReq2, getMasterErrReq2,
		updatePreOkReq3, getMasterOKReq3,
		updatePreOkReq4, getMasterOKReq4, changeStatusErrReq4,
		updatePreOkReq5, getMasterOKReq5, changeStatusOKReq5, changeMasterErrReq5,
		updatePreOkReq6, getMasterOKReq6, changeStatusOKReq6, changeMasterOKReq6)

	// check permission error
	res := rm.UpdateMasterRule(chains[0].ID, rules[0].Address, reason)
	assert.False(t, res.Ok, string(res.Result))
	// pause appchain error
	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address, reason)
	assert.False(t, res.Ok, string(res.Result))
	// GovernancePre error
	res = rm.UpdateMasterRule(chains[0].ID, rules[0].Address, reason)
	assert.False(t, res.Ok, string(res.Result))
	// no master
	res = rm.UpdateMasterRule(chains[0].ID, rules[3].Address, reason)
	assert.False(t, res.Ok, string(res.Result))
	// SubmitProposal error
	res = rm.UpdateMasterRule(chains[0].ID, rules[3].Address, reason)
	assert.False(t, res.Ok, string(res.Result))
	// changestatus error
	res = rm.UpdateMasterRule(chains[0].ID, rules[3].Address, reason)
	assert.False(t, res.Ok, string(res.Result))
	// change master status error
	res = rm.UpdateMasterRule(chains[0].ID, rules[4].Address, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.UpdateMasterRule(chains[0].ID, rules[5].Address, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_LogoutRule(t *testing.T) {
	rm, mockStub, rules, _, chains, _, _, rolesData := rulePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", gomock.Any()).Return(boltvm.Success(rolesData[0])).AnyTimes()
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)
	retRulesBindable := make([]*ruleMgr.Rule, 0)
	retRulesBindable = append(retRulesBindable, rules[3])
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesBindable).Return(true).AnyTimes()

	// check permission error
	res := rm.LogoutRule(chains[0].ID, rules[0].Address)
	assert.False(t, res.Ok, string(res.Result))
	// change status(get object) error
	res = rm.LogoutRule(chains[0].ID, rules[3].Address)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.LogoutRule(chains[0].ID, rules[3].Address)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_Query(t *testing.T) {
	rm, mockStub, rules, _, chains, _, _, _ := rulePrepare(t)

	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(2)
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, rules).Return(true).AnyTimes()
	rulesData, err := json.Marshal(rules)
	assert.Nil(t, err)
	mockStub.EXPECT().Query(gomock.Any()).Return(true, [][]byte{rulesData}).AnyTimes()

	res := rm.CountAvailableRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "0", string(res.Result))

	res = rm.CountRules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "0", string(res.Result))

	res = rm.Rules(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetRuleByAddr(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetMasterRule(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.HasMasterRule(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.IsAvailableRule(chains[0].ID, rules[0].Address)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetAllRules()
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_PauseRule(t *testing.T) {
	rm, mockStub, rules, _, chains, _, _, _ := rulePrepare(t)
	logger := log.NewWithModule("contracts")

	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Error("LockLowPriorityProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)
	retRulesMaster := make([]*ruleMgr.Rule, 0)
	retRulesMaster = append(retRulesMaster, rules[6])
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesMaster).Return(true).AnyTimes()

	// check permission error
	res := rm.PauseRule(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))
	// get master error
	res = rm.PauseRule(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))
	// LockLowPriorityProposal error
	res = rm.PauseRule(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.PauseRule(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRuleManager_UnPauseRule(t *testing.T) {
	rm, mockStub, rules, _, chains, _, _, _ := rulePrepare(t)
	logger := log.NewWithModule("contracts")

	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Error("UnLockLowPriorityProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).Return(false).Times(1)
	retRulesMaster := make([]*ruleMgr.Rule, 0)
	retRulesMaster = append(retRulesMaster, rules[6])
	mockStub.EXPECT().GetObject(ruleMgr.RuleKey(chains[0].ID), gomock.Any()).SetArg(1, retRulesMaster).Return(true).AnyTimes()

	// check permission error
	res := rm.UnPauseRule(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))
	// get master error
	res = rm.UnPauseRule(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))
	// LockLowPriorityProposal error
	res = rm.UnPauseRule(chains[0].ID)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.UnPauseRule(chains[0].ID)
	assert.True(t, res.Ok, string(res.Result))
}

func rulePrepare(t *testing.T) (*RuleManager, *mock_stub.MockStub, []*ruleMgr.Rule, [][]byte, []*appchainMgr.Appchain, [][]byte, ledger2.IAccount, [][]byte) {
	// 1. prepare stub
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	rm := &RuleManager{
		Stub: mockStub,
	}

	// 2. prepare chain
	var chains []*appchainMgr.Appchain
	var chainsData [][]byte
	chainType := []string{string(governance.GovernanceAvailable), string(governance.GovernanceFrozen)}

	for i := 0; i < 2; i++ {
		addr := appchainID + types.NewAddress([]byte{byte(i)}).String()

		chain := &appchainMgr.Appchain{
			Status:    governance.GovernanceStatus(chainType[i]),
			ChainType: appchainMgr.ChainTypeFabric1_4_3,
			ID:        addr,
			Desc:      "",
			Version:   0,
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
	}

	// 3. prepare rule
	var rules []*ruleMgr.Rule
	var rulesData [][]byte

	for i := 0; i < 7; i++ {
		ruleAddr := types.NewAddress([]byte(strconv.Itoa(i))).String()
		rule := &ruleMgr.Rule{
			Address: ruleAddr,
			RuleUrl: "www.baidu.com",
			ChainID: chains[0].ID,
			Status:  governance.GovernanceAvailable,
			Master:  true,
		}
		switch i {
		case 1:
			rule.Status = governance.GovernanceBinding
			rule.Master = false
		case 3:
			rule.Status = governance.GovernanceBindable
			rule.Master = false
		case 4:
			rule.Status = governance.GovernanceBindable
			rule.Master = false
		case 5:
			rule.Status = governance.GovernanceBindable
			rule.Master = false
		case 6:
			rule.Status = governance.GovernanceUnbinding
			rule.Master = true
		}

		data, err := json.Marshal(rule)
		assert.Nil(t, err)

		rulesData = append(rulesData, data)
		rules = append(rules, rule)
	}

	// 4. prepare account
	account := mockAccount(t)

	// 5. prepare role
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

	return rm, mockStub, rules, rulesData, chains, chainsData, account, rolesData
}

func mockAccount(t *testing.T) ledger2.IAccount {
	addr := types.NewAddress(bytesutil.LeftPadBytes([]byte{1}, 20))
	code := bytesutil.LeftPadBytes([]byte{1}, 120)
	repoRoot, err := ioutil.TempDir("", "contract")
	require.Nil(t, err)
	blockchainStorage, err := leveldb.New(filepath.Join(repoRoot, "contract"))
	require.Nil(t, err)
	ldb, err := leveldb.New(filepath.Join(repoRoot, "ledger"))
	require.Nil(t, err)
	repo.DefaultConfig()
	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	logger := log.NewWithModule("contract_test")
	blockFile, err := blockfile.NewBlockFile(repoRoot, logger)
	assert.Nil(t, err)
	ldg, err := ledger.New(createMockRepo(t), blockchainStorage, ldb, blockFile, accountCache, log.NewWithModule("ledger"))
	account := ldg.GetOrCreateAccount(addr)
	account.SetCodeAndHash(code)

	return account
}

func createMockRepo(t *testing.T) *repo.Repo {
	key := `-----BEGIN EC PRIVATE KEY-----
BcNwjTDCxyxLNjFKQfMAc6sY6iJs+Ma59WZyC/4uhjE=
-----END EC PRIVATE KEY-----`

	privKey, err := libp2pcert.ParsePrivateKey([]byte(key), crypto.Secp256k1)
	require.Nil(t, err)

	address, err := privKey.PublicKey().Address()
	require.Nil(t, err)

	return &repo.Repo{
		Key: &repo.Key{
			PrivKey: privKey,
			Address: address.String(),
		},
	}
}
