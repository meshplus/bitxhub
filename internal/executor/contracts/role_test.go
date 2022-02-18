package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/iancoleman/orderedmap"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	nodemgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

var (
	AUDIT_ADMIN_ROLE_ID  = "audit_admin_role_id"
	SUPER_ADMIN_ROLE_ID  = "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"
	SUPER_ADMIN_ROLE_ID1 = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8"
	ROLE_ID1             = "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D014"
	NEW_NODE_PID         = "QmWjeMdhS3L244WyFJGfasU4wDvaZfLTC7URq8aKxWvKmh"
)

func TestRoleManager_ManageRegisterAuditAdmin(t *testing.T) {
	rm, mockStub, _, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().GetAccount(gomock.Any()).Return(mockAccount(t)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(aRoles[1].ID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(RoleKey(aRoles[1].ID), gomock.Any()).SetArg(1, *aRoles[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleTypeKey(string(AuditAdmin)), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "ManageBindNode", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "ManageBindNode error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "ManageBindNode", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(aRoles[1].ID)).Return(true, aRolesData[0]).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("100000000000000000000000000000000000")).AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).AnyTimes()

	// check permission error
	res := rm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// change status error
	res = rm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// unpause ok
	res = rm.Manage(string(governance.EventUnpause), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// approve
	res = rm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// reject
	res = rm.Manage(string(governance.EventRegister), string(REJECTED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_ManageRegisterGvernanceAdmin(t *testing.T) {
	rm, mockStub, gRoles, gRolesData, _, _ := rolePrepare(t)

	mockStub.EXPECT().GetAccount(gomock.Any()).Return(mockAccount(t)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()

	// 9: registering
	mockStub.EXPECT().GetObject(RoleKey(gRoles[9].ID), gomock.Any()).SetArg(1, *gRoles[9]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleTypeKey(string(GovernanceAdmin)), gomock.Any()).Return(false).AnyTimes()

	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(gRoles[9].ID)).Return(true, gRolesData[9]).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("100000000000000000000000000000000000")).AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ProposalStrategyMgrContractAddr.Address().String(), "UpdateProposalStrategyByRolesChange", gomock.Any()).Return(boltvm.Error("", "crossinvoke UpdateProposalStrategyByRolesChange error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ProposalStrategyMgrContractAddr.Address().String(), "UpdateProposalStrategyByRolesChange", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// approve
	res := rm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), gRoles[9].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), gRoles[9].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// reject
	res = rm.Manage(string(governance.EventRegister), string(REJECTED), string(governance.GovernanceAvailable), gRoles[9].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_ManageBindAuditAdmin(t *testing.T) {
	rm, mockStub, _, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().GetAccount(gomock.Any()).Return(mockAccount(t)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(aRoles[3].ID), gomock.Any()).SetArg(1, *aRoles[3]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "ManageBindNode", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "ManageBindNode error")).Times(2)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "ManageBindNode", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(aRoles[3].ID)).Return(true, aRolesData[0]).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("100000000000000000000000000000000000")).AnyTimes()

	// approve
	res := rm.Manage(string(governance.EventBind), string(APPROVED), string(governance.GovernanceFrozen), aRoles[3].ID, nil)
	assert.False(t, res.Ok, string(res.Result))

	// reject
	res = rm.Manage(string(governance.EventBind), string(REJECTED), string(governance.GovernanceFrozen), aRoles[3].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.Manage(string(governance.EventBind), string(REJECTED), string(governance.GovernanceFrozen), aRoles[3].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_ManageLogoutAuditAdmin(t *testing.T) {
	rm, mockStub, _, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().GetAccount(gomock.Any()).Return(mockAccount(t)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(aRoles[4].ID), gomock.Any()).SetArg(1, *aRoles[4]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "EndObjProposal", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "EndObjProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "EndObjProposal", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "UnbindNode", gomock.Any()).Return(boltvm.Error("", "UnbindNode error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "UnbindNode", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "GetNode", gomock.Any()).Return(boltvm.Error("", "GetNode error")).Times(1)
	forbiddenNode := &nodemgr.Node{
		Account:  aRoles[4].NodeAccount,
		NodeType: nodemgr.NVPNode,
		Status:   governance.GovernanceForbidden,
	}
	forbiddenNodeData, err := json.Marshal(forbiddenNode)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "GetNode", gomock.Any()).Return(boltvm.Success(forbiddenNodeData)).Times(1)
	bindintNode := &nodemgr.Node{
		Account:  aRoles[4].NodeAccount,
		NodeType: nodemgr.NVPNode,
		Status:   governance.GovernanceBinding,
	}
	bindintNodeData, err := json.Marshal(bindintNode)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "GetNode", gomock.Any()).Return(boltvm.Success(bindintNodeData)).AnyTimes()

	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(aRoles[4].ID)).Return(true, aRolesData[4]).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("100000000000000000000000000000000000")).AnyTimes()

	// approve
	// EndObjProposal error
	res := rm.Manage(string(governance.EventLogout), string(APPROVED), string(governance.GovernanceFrozen), aRoles[4].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// UnbindNode error
	res = rm.Manage(string(governance.EventLogout), string(APPROVED), string(governance.GovernanceFrozen), aRoles[4].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// ok
	res = rm.Manage(string(governance.EventLogout), string(APPROVED), string(governance.GovernanceFrozen), aRoles[4].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// reject
	// GetNode error
	res = rm.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceBinding), aRoles[4].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// binding forbidden node
	res = rm.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceBinding), aRoles[4].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
	// binding bindg node
	res = rm.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceBinding), aRoles[4].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_ManageFreezeActivateLogoutGovernanceAdmin(t *testing.T) {
	rm, mockStub, gRoles, gRolesData, _, _ := rolePrepare(t)

	mockStub.EXPECT().GetAccount(gomock.Any()).Return(mockAccount(t)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[4].ID), gomock.Any()).SetArg(1, *gRoles[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[5].ID), gomock.Any()).SetArg(1, *gRoles[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[6].ID), gomock.Any()).SetArg(1, *gRoles[6]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[7].ID), gomock.Any()).SetArg(1, *gRoles[7]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[8].ID), gomock.Any()).SetArg(1, *gRoles[8]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(gRoles[4].ID)).Return(true, gRolesData[0]).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(gRoles[5].ID)).Return(true, gRolesData[0]).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(gRoles[6].ID)).Return(true, gRolesData[0]).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(gRoles[7].ID)).Return(true, gRolesData[0]).AnyTimes()
	mockStub.EXPECT().Get(RoleKey(gRoles[8].ID)).Return(true, gRolesData[0]).AnyTimes()

	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("100000000000000000000000000000000000")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals").Return(boltvm.Error("", "GetNotClosedProposals error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals").Return(boltvm.Success(nil)).Times(1)
	var proposals []*Proposal
	proposals = append(proposals, &Proposal{
		ElectorateList: []*Role{
			&Role{
				ID: gRoles[1].ID,
			},
		},
	})
	proposalsData, err := json.Marshal(proposals)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals").Return(boltvm.Success(proposalsData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ProposalStrategyMgrContractAddr.Address().String(), "UpdateProposalStrategyByRolesChange", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	roleIdMap := orderedmap.New()
	roleIdMap.Set(gRoles[4].ID, struct{}{})
	roleIdMap.Set(gRoles[5].ID, struct{}{})
	roleIdMap.Set(gRoles[6].ID, struct{}{})
	roleIdMap.Set(gRoles[7].ID, struct{}{})
	roleIdMap.Set(gRoles[8].ID, struct{}{})
	mockStub.EXPECT().GetObject(RoleTypeKey(string(GovernanceAdmin)), gomock.Any()).SetArg(1, *roleIdMap).Return(true).AnyTimes()

	// freeze, approve
	res := rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), gRoles[4].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), gRoles[7].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), gRoles[8].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// activate, approve
	res = rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceFrozen), gRoles[5].ID, nil)
	assert.True(t, res.Ok, string(res.Result))

	// logout, reject
	res = rm.Manage(string(governance.EventLogout), string(REJECTED), string(governance.GovernanceAvailable), gRoles[6].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_RegisterRole(t *testing.T) {
	rm, mockStub, gRoles, gRolesData, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(gRoles[3].ID).AnyTimes()
	mockStub.EXPECT().Caller().Return(gRoles[3].ID).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, gRolesData[0]).AnyTimes()
	node := &nodemgr.Node{
		NodeType: nodemgr.NVPNode,
		Status:   governance.GovernanceAvailable,
	}
	nodeData, err := json.Marshal(node)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "GetNode", gomock.Any()).Return(boltvm.Success(nodeData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "BindNode", gomock.Any()).Return(boltvm.Error("", "BindNode error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "BindNode", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().GetObject(RoleKey(gRoles[0].ID), gomock.Any()).SetArg(1, *gRoles[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[1].ID), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[3].ID), gomock.Any()).SetArg(1, *gRoles[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()

	// check permission error
	res := rm.RegisterRole(gRoles[0].ID, string(GovernanceAdmin), "", reason)
	assert.False(t, res.Ok, string(res.Result))

	// check info error
	res = rm.RegisterRole(gRoles[0].ID, "", "", reason)
	assert.False(t, res.Ok, string(res.Result))

	// governance pre error
	res = rm.RegisterRole(gRoles[1].ID, string(GovernanceAdmin), "", reason)
	assert.False(t, res.Ok, string(res.Result))

	// submit proposal error
	res = rm.RegisterRole(gRoles[0].ID, string(GovernanceAdmin), "", reason)
	assert.False(t, res.Ok, string(res.Result))

	// bind error
	res = rm.RegisterRole(gRoles[0].ID, string(AuditAdmin), "", reason)
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = rm.RegisterRole(gRoles[0].ID, string(GovernanceAdmin), "", reason)
	assert.True(t, res.Ok, string(res.Result))

	// audit admin
	res = rm.RegisterRole(gRoles[0].ID, string(AuditAdmin), NODE_ACCOUNT, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_UpdateAppchainAdmin(t *testing.T) {
	rm, mockStub, _, gRolesData, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, gRolesData[0]).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// check permission error
	res := rm.UpdateAppchainAdmin(appchainID, ROLE_ID1)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.UpdateAppchainAdmin(appchainID, ROLE_ID1)
	assert.True(t, res.Ok, string(res.Result))

}

func TestRoleManager_FreezeRole(t *testing.T) {
	rm, mockStub, gRoles, gRolesData, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(ROLE_ID1).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()

	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	governancePreErrReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).Return(false).Times(1)
	submitErrreq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).Times(1)
	changeStatusErrReq1 := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).Times(1)
	changeStatusErrReq2 := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).Return(false).Times(1)
	okReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).Times(2)
	gomock.InOrder(governancePreErrReq, submitErrreq, changeStatusErrReq1, changeStatusErrReq2, okReq)
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, gRolesData[0]).AnyTimes()

	// check permission error
	res := rm.FreezeRole(ROLE_ID1, reason)
	assert.False(t, res.Ok, string(res.Result))
	// governance pre error
	res = rm.FreezeRole(ROLE_ID1, reason)
	assert.False(t, res.Ok, string(res.Result))
	// submit error
	res = rm.FreezeRole(ROLE_ID1, reason)
	assert.False(t, res.Ok, string(res.Result))
	// changestatus error
	res = rm.FreezeRole(ROLE_ID1, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.FreezeRole(ROLE_ID1, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_ActivateRole(t *testing.T) {
	rm, mockStub, gRoles, gRolesData, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[4]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, gRolesData[0]).AnyTimes()

	res := rm.ActivateRole(ROLE_ID1, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_LogoutRole(t *testing.T) {
	rm, mockStub, gRoles, _, aRoles, rRolesData := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	roleIdMap := orderedmap.New()
	roleIdMap.Set(gRoles[1].ID, struct{}{})
	roleIdMap.Set(gRoles[5].ID, struct{}{})
	roleIdMap.Set(gRoles[6].ID, struct{}{})
	roleIdMap.Set(gRoles[7].ID, struct{}{})
	roleIdMap.Set(gRoles[8].ID, struct{}{})
	mockStub.EXPECT().GetObject(RoleTypeKey(string(GovernanceAdmin)), gomock.Any()).SetArg(1, *roleIdMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals").Return(boltvm.Error("", "GetNotClosedProposals error")).Times(1)
	var proposals []*Proposal
	proposals = append(proposals, &Proposal{
		ElectorateList: []*Role{
			&Role{
				ID: aRoles[1].ID,
			},
		},
	})
	proposalsData, err := json.Marshal(proposals)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals").Return(boltvm.Success(proposalsData)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, rRolesData[0]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ProposalStrategyMgrContractAddr.Address().String(), "UpdateProposalStrategyByRolesChange", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	res := rm.LogoutRole(ROLE_ID1, reason)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.LogoutRole(ROLE_ID1, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_BindRole(t *testing.T) {
	rm, mockStub, gRoles, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(gRoles[3].ID).AnyTimes()
	mockStub.EXPECT().Caller().Return(gRoles[3].ID).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(aRoles[2].ID), gomock.Any()).SetArg(1, *aRoles[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(gRoles[3].ID), gomock.Any()).SetArg(1, *gRoles[3]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission",
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "BindNode", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, aRolesData[2]).AnyTimes()

	res := rm.BindRole(aRoles[2].ID, aRoles[2].NodeAccount, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_PauseAuditAdmin(t *testing.T) {
	rm, mockStub, _, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Caller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()

	// 0: available
	mockStub.EXPECT().GetObject(RoleKey(aRoles[0].ID), gomock.Any()).SetArg(1, *aRoles[0]).Return(true).AnyTimes()
	// 2: frozen
	mockStub.EXPECT().GetObject(RoleKey(aRoles[2].ID), gomock.Any()).SetArg(1, *aRoles[2]).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "EndObjProposal", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "EndObjProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "EndObjProposal", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, aRolesData[0]).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Query(gomock.Any()).Return(true, aRolesData).AnyTimes()

	// check permission error
	res := rm.PauseAuditAdmin(aRoles[0].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// get the audit admin which is bound or would be bound error
	res = rm.PauseAuditAdmin(aRoles[0].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// pause frozen node, not ok
	res = rm.PauseAuditAdmin(aRoles[2].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// EndObjProposal error
	res = rm.PauseAuditAdmin(aRoles[0].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = rm.PauseAuditAdmin(aRoles[0].NodeAccount)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_PauseAuditAdminBinding(t *testing.T) {
	rm, mockStub, _, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Caller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()

	// 3: binding
	mockStub.EXPECT().GetObject(RoleKey(aRoles[3].ID), gomock.Any()).SetArg(1, *aRoles[3]).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Error("", "LockLowPriorityProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, aRolesData[3]).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Query(gomock.Any()).Return(true, aRolesData).AnyTimes()

	// check permission error
	res := rm.PauseAuditAdminBinding(aRoles[3].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// get the audit admin which is bound or would be bound error
	res = rm.PauseAuditAdminBinding(aRoles[3].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// LockLowPriorityProposal error
	res = rm.PauseAuditAdminBinding(aRoles[3].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = rm.PauseAuditAdminBinding(aRoles[3].NodeAccount)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_RestoreAuditAdminBinding(t *testing.T) {
	rm, mockStub, _, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Caller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()

	// 3: binding
	mockStub.EXPECT().GetObject(RoleKey(aRoles[3].ID), gomock.Any()).SetArg(1, *aRoles[3]).Return(true).AnyTimes()
	// 4: logouting
	mockStub.EXPECT().GetObject(RoleKey(aRoles[4].ID), gomock.Any()).SetArg(1, *aRoles[4]).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Error("", "UnLockLowPriorityProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UnLockLowPriorityProposal", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, aRolesData[3]).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Query(gomock.Any()).Return(true, aRolesData).AnyTimes()

	// check permission error
	res := rm.RestoreAuditAdminBinding(aRoles[3].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// get the audit admin which is bound or would be bound error
	res = rm.RestoreAuditAdminBinding(aRoles[3].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// restore logouting admin, not ok
	res = rm.RestoreAuditAdminBinding(aRoles[4].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// UnLockLowPriorityProposal error
	res = rm.RestoreAuditAdminBinding(aRoles[3].NodeAccount)
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = rm.RestoreAuditAdminBinding(aRoles[3].NodeAccount)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_OccupyAccount(t *testing.T) {
	rm, mockStub, _, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// check permission error
	res := rm.OccupyAccount(adminAddr, string(GovernanceAdmin))
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = rm.OccupyAccount(adminAddr, string(GovernanceAdmin))
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_FreeAccount(t *testing.T) {
	rm, mockStub, _, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.NodeManagerContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// check permission error
	res := rm.FreeAccount(adminAddr)
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = rm.FreeAccount(adminAddr)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_CheckOccupiedAccount(t *testing.T) {
	rm, mockStub, _, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, string(GovernanceAdmin)).Return(true).Times(1)

	// not occupied account
	res := rm.CheckOccupiedAccount(adminAddr)
	assert.True(t, res.Ok, string(res.Result))
	// occupied account
	res = rm.CheckOccupiedAccount(adminAddr)
	assert.False(t, res.Ok, string(res.Result))
}

func TestRoleManager_GetRole(t *testing.T) {
	rm, mockStub, gRoles, _, aRoles, _ := rolePrepare(t)

	adminReq := mockStub.EXPECT().Caller().Return(adminAddr).Times(1)
	auditAdminReq := mockStub.EXPECT().Caller().Return(AUDIT_ADMIN_ROLE_ID).Times(1)
	appAdminReq := mockStub.EXPECT().Caller().Return(appchainAdminAddr).Times(1)
	superAdminReq := mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID).Times(1)
	gomock.InOrder(adminReq, auditAdminReq, appAdminReq, superAdminReq)
	mockStub.EXPECT().GetObject(RoleKey(adminAddr), gomock.Any()).SetArg(1, *gRoles[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(AUDIT_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *aRoles[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(appchainAdminAddr), gomock.Any()).SetArg(1, Role{
		ID:       appchainAdminAddr,
		RoleType: AppchainAdmin,
	}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *gRoles[10]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(noAdminAddr), gomock.Any()).Return(false).AnyTimes()

	res := rm.GetRole()
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, string(GovernanceAdmin), string(res.Result))
	res = rm.GetRole()
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, string(AuditAdmin), string(res.Result))
	res = rm.GetRole()
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, string(AppchainAdmin), string(res.Result))
	res = rm.GetRole()
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, string(SuperGovernanceAdmin), string(res.Result))
	res = rm.GetRoleByAddr(noAdminAddr)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, string(NoRole), string(res.Result))
}

func TestRoleManager_Query(t *testing.T) {
	rm, mockStub, gRoles, gRolesData, aRoles, _ := rolePrepare(t)

	mockStub.EXPECT().GetObject(RoleKey(adminAddr), gomock.Any()).SetArg(1, *gRoles[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(AUDIT_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *aRoles[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *gRoles[10]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(noAdminAddr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(noAdminAddr), gomock.Any()).Return(false).AnyTimes()
	res := rm.GetRoleInfoById(noAdminAddr)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.GetRoleInfoById(adminAddr)
	assert.True(t, res.Ok, string(res.Result))

	getAllErrReq := mockStub.EXPECT().Query(RolePrefix).Return(false, nil).Times(1)
	getAllOkReq := mockStub.EXPECT().Query(RolePrefix).Return(true, gRolesData).Times(1)
	gomock.InOrder(getAllErrReq, getAllOkReq)
	res = rm.GetAllRoles()
	assert.True(t, res.Ok, string(res.Result))
	roles := []*Role{}
	err := json.Unmarshal(res.Result, &roles)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(roles), string(res.Result))
	res = rm.GetAllRoles()
	assert.True(t, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, &roles)
	assert.Nil(t, err)
	assert.Equal(t, len(gRolesData), len(roles))

	res = rm.GetRolesByType("")
	assert.False(t, res.Ok, string(res.Result))
	roleIdMap := orderedmap.New()
	roleIdMap.Set(aRoles[0].ID, struct{}{})
	mockStub.EXPECT().GetObject(RoleTypeKey(string(AuditAdmin)), gomock.Any()).SetArg(1, *roleIdMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(aRoles[0].ID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(RoleKey(aRoles[0].ID), gomock.Any()).SetArg(1, *aRoles[0]).Return(true).AnyTimes()
	res = rm.GetRolesByType(string(AuditAdmin))
	assert.False(t, res.Ok, string(res.Result))
	res = rm.GetRolesByType(string(AuditAdmin))
	assert.True(t, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, &roles)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(roles), string(res.Result))

	mockStub.EXPECT().GetObject(RoleAppchainAdminKey(appchainID), gomock.Any()).Return(false).Times(1)
	res = rm.GetAppchainAdmin(appchainID)
	assert.False(t, res.Ok, string(res.Result))
	appchainAdminIdMap := orderedmap.New()
	appchainAdminIdMap.Set(appchainAdminAddr, struct{}{})
	mockStub.EXPECT().GetObject(RoleAppchainAdminKey(appchainID), gomock.Any()).SetArg(1, appchainAdminIdMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(appchainAdminAddr), gomock.Any()).Return(false).Times(1)
	res = rm.GetAppchainAdmin(appchainID)
	assert.False(t, res.Ok, string(res.Result))
	mockStub.EXPECT().GetObject(RoleKey(appchainAdminAddr), gomock.Any()).SetArg(1, Role{
		ID:       appchainAdminAddr,
		RoleType: AppchainAdmin,
	}).Return(true).AnyTimes()
	res = rm.GetAppchainAdmin(appchainID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.IsAnyAdmin(adminAddr, string(GovernanceAdmin))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))
	res = rm.IsAnyAdmin(SUPER_ADMIN_ROLE_ID, string(SuperGovernanceAdmin))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))

	res = rm.IsAnyAvailableAdmin(adminAddr, string(GovernanceAdmin))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))
	res = rm.IsAnyAvailableAdmin(SUPER_ADMIN_ROLE_ID, string(GovernanceAdmin))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))

	res = rm.GetRoleWeight(adminAddr)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, strconv.Itoa(repo.NormalAdminWeight), string(res.Result))
	res = rm.GetRoleWeight(AUDIT_ADMIN_ROLE_ID)
	assert.False(t, res.Ok, string(res.Result))
}

func TestRoleManager_checkPermission(t *testing.T) {
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *gRoles[0]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()
	err := rm.checkPermission([]string{string(PermissionAdmin)}, gRoles[0].ID, adminAddr, nil)
	assert.NotNil(t, err)
	err = rm.checkPermission([]string{string(PermissionAdmin)}, gRoles[0].ID, adminAddr, nil)
	assert.Nil(t, err)

	err = rm.checkPermission([]string{string(PermissionSelf)}, gRoles[0].ID, gRoles[0].ID, nil)
	assert.Nil(t, err)

	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	err = rm.checkPermission([]string{string(PermissionSpecific)}, gRoles[0].ID, constant.GovernanceContractAddr.Address().String(), addrsData)
	assert.Nil(t, err)

	err = rm.checkPermission([]string{""}, gRoles[0].ID, "", nil)
	assert.NotNil(t, err)
}

func rolePrepare(t *testing.T) (*RoleManager, *mock_stub.MockStub, []*Role, [][]byte, []*Role, [][]byte) {
	// 1. prepare stub
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	rm := &RoleManager{
		Stub: mockStub,
	}

	// 2. prepare governance admin role
	governanceRoleStatus := []string{
		string(governance.GovernanceUnavailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceFrozen),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceFreezing),
		string(governance.GovernanceActivating),
		string(governance.GovernanceLogouting),
		string(governance.GovernanceFreezing),
		string(governance.GovernanceFreezing),
		string(governance.GovernanceRegisting),
	}

	var governanceRoles []*Role
	var governanceRolesData [][]byte
	for i := 0; i < 10; i++ {
		governanceRole := &Role{
			ID:       fmt.Sprintf("%s%d", ROLE_ID1[0:len(ROLE_ID1)-1], i),
			RoleType: GovernanceAdmin,
			Weight:   repo.NormalAdminWeight,
			Status:   governance.GovernanceStatus(governanceRoleStatus[i]),
		}

		data, err := json.Marshal(governanceRole)
		assert.Nil(t, err)

		governanceRolesData = append(governanceRolesData, data)
		governanceRoles = append(governanceRoles, governanceRole)
	}
	governanceRole := &Role{
		ID:          SUPER_ADMIN_ROLE_ID,
		RoleType:    GovernanceAdmin,
		Weight:      repo.SuperAdminWeight,
		NodeAccount: NODE_ACCOUNT,
		Status:      governance.GovernanceAvailable,
	}

	data, err := json.Marshal(governanceRole)
	assert.Nil(t, err)

	governanceRolesData = append(governanceRolesData, data)
	governanceRoles = append(governanceRoles, governanceRole)

	// 3. prepare audit admin role
	auditRoleStatus := []string{
		string(governance.GovernanceAvailable),
		string(governance.GovernanceRegisting),
		string(governance.GovernanceFrozen),
		string(governance.GovernanceBinding),
		string(governance.GovernanceLogouting),
	}

	var auditRoles []*Role
	var auditRolesData [][]byte
	for i := 0; i < 5; i++ {
		role := &Role{
			ID:          fmt.Sprintf("%s%d", ROLE_ID1, i),
			RoleType:    AuditAdmin,
			Weight:      repo.SuperAdminWeight,
			NodeAccount: fmt.Sprintf("%s%d", NODE_ACCOUNT, i),
			Status:      governance.GovernanceStatus(auditRoleStatus[i]),
		}

		data, err := json.Marshal(role)
		assert.Nil(t, err)

		auditRolesData = append(auditRolesData, data)
		auditRoles = append(auditRoles, role)
	}

	return rm, mockStub, governanceRoles, governanceRolesData, auditRoles, auditRolesData
}
