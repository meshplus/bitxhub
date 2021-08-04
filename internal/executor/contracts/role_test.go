package contracts

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

var (
	SUPER_ADMIN_ROLE_ID  = "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"
	SUPER_ADMIN_ROLE_ID1 = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8"
	ROLE_ID1             = "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D014"
	NEW_NODE_PID         = "QmWjeMdhS3L244WyFJGfasU4wDvaZfLTC7URq8aKxWvKmh"
)

func TestRoleManager_Manage(t *testing.T) {
	rm, mockStub, _, _, aRoles, aRolesData := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *aRoles[1]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	res := rm.Manage(string(governance.EventUpdate), string(APPOVED), string(governance.GovernanceAvailable), aRolesData[0])
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_RegisterRole(t *testing.T) {
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)
	account := mockAccount(t)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().GetObject(rm.roleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(rm.roleKey(SUPER_ADMIN_ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[4]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, []byte("100000000000000000000000000000000000")).AnyTimes()
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()

	res := rm.RegisterRole(ROLE_ID1, string(GovernanceAdmin), NODEPID)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_UpdateAuditAdminNode(t *testing.T) {
	rm, mockStub, _, _, aRoles, _ := rolePrepare(t)

	nvpNode := &node_mgr.Node{
		VPNodeId: uint64(1),
		Pid:      NODEPID,
		Account:  NODEACCOUNT,
		NodeType: node_mgr.NVPNode,
		Status:   governance.GovernanceAvailable,
	}

	nvpNodeData, err := json.Marshal(nvpNode)
	assert.Nil(t, err)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *aRoles[0]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.String(), "GetNode", gomock.Any()).Return(boltvm.Success(nvpNodeData)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	res := rm.UpdateAuditAdminNode(SUPER_ADMIN_ROLE_ID1, NEW_NODE_PID)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_FreezeRole(t *testing.T) {
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *gRoles[1]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	res := rm.FreezeRole(ROLE_ID1)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_ActivateRole(t *testing.T) {
	rm, mockStub, gRoles, _, _, _ := rolePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().Caller().Return(SUPER_ADMIN_ROLE_ID1).AnyTimes()
	mockStub.EXPECT().GetObject(rm.roleKey(ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(rm.roleKey(SUPER_ADMIN_ROLE_ID1), gomock.Any()).SetArg(1, *gRoles[4]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	res := rm.ActivateRole(ROLE_ID1)
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

	res := rm.LogoutRole(ROLE_ID1)
	assert.True(t, res.Ok, string(res.Result))
}

func TestRoleManager_GetRole(t *testing.T) {
	rm, mockStub, gRoles, gRolesData, _, _ := rolePrepare(t)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *gRoles[3]).Return(true).AnyTimes()
	mockStub.EXPECT().Query(ROLEPREFIX).Return(true, gRolesData).AnyTimes()

	res := rm.GetRoleById(SUPER_ADMIN_ROLE_ID)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetAdminRoles()
	assert.True(t, res.Ok, string(res.Result))

	res = rm.GetAuditAdminRoles()
	assert.True(t, res.Ok, string(res.Result))

	res = rm.IsAvailable(SUPER_ADMIN_ROLE_ID1)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.IsSuperAdmin(SUPER_ADMIN_ROLE_ID1)
	assert.True(t, res.Ok, string(res.Result))

	res = rm.IsAdmin(SUPER_ADMIN_ROLE_ID1)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))

	res = rm.IsAuditAdmin(SUPER_ADMIN_ROLE_ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "false", string(res.Result))

	res = rm.IsAnyAdmin(SUPER_ADMIN_ROLE_ID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))
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
		ID:       SUPER_ADMIN_ROLE_ID1,
		RoleType: GovernanceAdmin,
		Weight:   repo.NormalAdminWeight,
		NodePid:  NODEPID,
		Status:   governance.GovernanceAvailable,
	}

	data, err := json.Marshal(governanceRole)
	assert.Nil(t, err)

	governanceRolesData = append(governanceRolesData, data)
	governanceRoles = append(governanceRoles, governanceRole)

	// 2. prepare audit admin role
	auditRoleStatus := []string{
		string(governance.GovernanceAvailable),
		string(governance.GovernanceUpdating),
	}

	var auditRoles []*Role
	var auditRolesData [][]byte
	for i := 0; i < 2; i++ {
		role := &Role{
			ID:       ROLE_ID1,
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
