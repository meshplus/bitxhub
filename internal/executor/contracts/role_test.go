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

func TestRoleManager_Manage(t *testing.T) {
	rm, mockStub, _, _, aRoles, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *aRoles[1]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals").Return(boltvm.Error("GetNotClosedProposals error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals").Return(boltvm.Success(nil)).Times(1)
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
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UpdateAvaliableElectorateNum", gomock.Any()).Return(boltvm.Error("UpdateAvaliableElectorateNum error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UpdateAvaliableElectorateNum", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// check permission error
	res := rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// change status error
	res = rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// GetNotClosedProposals error
	res = rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// unmarshal error
	res = rm.Manage(string(governance.EventLogout), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))
	// UpdateAvaliableElectorateNum error
	res = rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.Manage(string(governance.EventFreeze), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
	res = rm.Manage(string(governance.EventLogout), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
	res = rm.Manage(string(governance.EventActivate), string(APPROVED), string(governance.GovernanceAvailable), aRoles[1].ID, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_RegisterRole(t *testing.T) {
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)
	account := mockAccount(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("100000000000000000000000000000000000")).AnyTimes()
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()

	mockStub.EXPECT().GetObject(RoleKey(noAdminAddr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()

	governancePreErrReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[2]).Return(true).Times(1)
	getErrReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[0]).Return(true).Times(1)
	submitErrReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[0]).Return(true).Times(1)
	changeStatusErrReq1 := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[0]).Return(true).Times(1)
	changeStatusErrReq2 := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).Return(false).Times(1)
	okReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[0]).Return(true).Times(2)
	gomock.InOrder(governancePreErrReq, getErrReq, submitErrReq, changeStatusErrReq1, changeStatusErrReq2, okReq)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()

	// check permission error
	res := rm.RegisterRole(ROLE_ID1, string(GovernanceAdmin), NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// check info error
	res = rm.RegisterRole(ROLE_ID1, "", NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// governance pre error
	res = rm.RegisterRole(ROLE_ID1, string(GovernanceAdmin), NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))

	// GovernanceAdmin
	// get error
	res = rm.RegisterRole(ROLE_ID1, string(GovernanceAdmin), NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// submit proposal error
	res = rm.RegisterRole(ROLE_ID1, string(GovernanceAdmin), NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// change status error
	res = rm.RegisterRole(ROLE_ID1, string(GovernanceAdmin), NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = rm.RegisterRole(ROLE_ID1, string(GovernanceAdmin), NODEPID, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_UpdateAppchainAdmin(t *testing.T) {
	rm, mockStub, _, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()

	// check permission error
	res := rm.UpdateAppchainAdmin(appchainID, ROLE_ID1)
	assert.False(t, res.Ok, string(res.Result))
	res = rm.UpdateAppchainAdmin(appchainID, ROLE_ID1)
	assert.True(t, res.Ok, string(res.Result))

}

func TestRoleManager_FreezeRole(t *testing.T) {
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(noAdminAddr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()

	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	governancePreErrReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).Return(false).Times(1)
	submitErrreq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).Times(1)
	changeStatusErrReq1 := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).Times(1)
	changeStatusErrReq2 := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).Return(false).Times(1)
	okReq := mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).Times(2)
	gomock.InOrder(governancePreErrReq, submitErrreq, changeStatusErrReq1, changeStatusErrReq2, okReq)
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

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
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[4]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	res := rm.ActivateRole(ROLE_ID1, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_LogoutRole(t *testing.T) {
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	res := rm.LogoutRole(ROLE_ID1, reason)
	assert.True(t, res.Ok, string(res.Result))
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
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *gRoles[4]).Return(true).AnyTimes()
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
	mockStub.EXPECT().GetObject(RoleKey(appchainAdminAddr), gomock.Any()).SetArg(1, Role{
		ID:       appchainAdminAddr,
		RoleType: AppchainAdmin,
	}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(RoleKey(SUPER_ADMIN_ROLE_ID), gomock.Any()).SetArg(1, *gRoles[4]).Return(true).AnyTimes()
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

	mockStub.EXPECT().GetObject(RoleTypeKey(string(AuditAdmin)), gomock.Any()).Return(false).Times(1)
	res = rm.GetRolesByType(string(AuditAdmin))
	err = json.Unmarshal(res.Result, &roles)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(roles), string(res.Result))

	appchainAdminIdMap := orderedmap.New()
	appchainAdminIdMap.Set(appchainAdminAddr, struct{}{})
	mockStub.EXPECT().GetObject(RoleAppchainAdminKey(appchainID), gomock.Any()).SetArg(1, appchainAdminIdMap).Return(true).AnyTimes()
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
	err := rm.checkPermission([]string{string(PermissionAdmin)}, adminAddr, adminAddr, nil)
	assert.NotNil(t, err)
	err = rm.checkPermission([]string{string(PermissionAdmin)}, adminAddr, adminAddr, nil)
	assert.Nil(t, err)

	err = rm.checkPermission([]string{string(PermissionSelf)}, noAdminAddr, noAdminAddr, nil)
	assert.Nil(t, err)

	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	err = rm.checkPermission([]string{string(PermissionSpecific)}, "", constant.GovernanceContractAddr.Address().String(), addrsData)
	assert.Nil(t, err)

	err = rm.checkPermission([]string{""}, "", "", nil)
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
		string(governance.GovernanceAvailable)}

	var governanceRoles []*Role
	var governanceRolesData [][]byte
	for i := 0; i < 4; i++ {
		governanceRole := &Role{
			ID:       ROLE_ID1,
			RoleType: GovernanceAdmin,
			Weight:   repo.NormalAdminWeight,
			NodePid:  NODEPID,
			Status:   governance.GovernanceStatus(governanceRoleStatus[i]),
		}

		data, err := json.Marshal(governanceRole)
		assert.Nil(t, err)

		governanceRolesData = append(governanceRolesData, data)
		governanceRoles = append(governanceRoles, governanceRole)
	}
	governanceRole := &Role{
		ID:       SUPER_ADMIN_ROLE_ID,
		RoleType: GovernanceAdmin,
		Weight:   repo.SuperAdminWeight,
		NodePid:  NODEPID,
		Status:   governance.GovernanceAvailable,
	}

	data, err := json.Marshal(governanceRole)
	assert.Nil(t, err)

	governanceRolesData = append(governanceRolesData, data)
	governanceRoles = append(governanceRoles, governanceRole)

	// 3. prepare audit admin role
	auditRoleStatus := []string{
		string(governance.GovernanceAvailable),
		string(governance.GovernanceFreezing),
	}

	var auditRoles []*Role
	var auditRolesData [][]byte
	for i := 0; i < 2; i++ {
		role := &Role{
			ID:       fmt.Sprintf("%s%d", ROLE_ID1, i),
			RoleType: AuditAdmin,
			Weight:   repo.SuperAdminWeight,
			NodePid:  NODEPID,
			Status:   governance.GovernanceStatus(auditRoleStatus[i]),
		}

		data, err := json.Marshal(role)
		assert.Nil(t, err)

		auditRolesData = append(auditRolesData, data)
		auditRoles = append(auditRoles, role)
	}

	return rm, mockStub, governanceRoles, governanceRolesData, auditRoles, auditRolesData
}
