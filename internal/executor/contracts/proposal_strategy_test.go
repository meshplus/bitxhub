package contracts

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

func TestGovStrategy_UpdateProposalStrategy(t *testing.T) {
	g, mockStub, strategies, _ := proposalStrategyPrepare(t)
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()

	// 0: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[7]).Return(true).Times(1)
	// 0: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[0]).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin))).Return(boltvm.Error("", "crossinvoke GetRolesByType error")).Times(1)
	roles := make([]*Role, 0)
	roles = append(roles, &Role{
		Status: governance.GovernanceAvailable,
	})
	rolesData, err := json.Marshal(roles)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin))).Return(boltvm.Success(rolesData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().Caller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	// promission error
	res := g.UpdateProposalStrategy(strategies[0].Module, string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.False(t, res.Ok, string(res.Result))

	// update updating strategy error
	res = g.UpdateProposalStrategy(strategies[0].Module, string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.False(t, res.Ok, string(res.Result))

	// not update error
	res = g.UpdateProposalStrategy(strategies[0].Module, string(SimpleMajority), "a==t", "123")
	assert.False(t, res.Ok, string(res.Result))

	// GetRolesByType error
	res = g.UpdateProposalStrategy(strategies[0].Module, string(ZeroPermission), "", "123")
	assert.False(t, res.Ok, string(res.Result))

	// illegal strategy info error
	res = g.UpdateProposalStrategy(strategies[0].Module, "", "", "123")
	assert.False(t, res.Ok, string(res.Result))

	// submit proposal error
	res = g.UpdateProposalStrategy(strategies[0].Module, string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = g.UpdateProposalStrategy(strategies[0].Module, string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovStrategy_UpdateAllProposalStrategy(t *testing.T) {
	g, mockStub, strategies, _ := proposalStrategyPrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()

	// 0: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[7]).Return(true).Times(1)
	// 0: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[0]).Return(true).AnyTimes()
	// 1: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[1].Module), gomock.Any()).SetArg(1, *strategies[1]).Return(true).AnyTimes()
	// 2: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[2].Module), gomock.Any()).SetArg(1, *strategies[2]).Return(true).AnyTimes()
	// 3: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[3].Module), gomock.Any()).SetArg(1, *strategies[3]).Return(true).AnyTimes()
	// 4: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[4].Module), gomock.Any()).SetArg(1, *strategies[4]).Return(true).AnyTimes()
	// 5: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[5].Module), gomock.Any()).SetArg(1, *strategies[5]).Return(true).AnyTimes()
	// 6: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[6].Module), gomock.Any()).SetArg(1, *strategies[6]).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin))).Return(boltvm.Error("", "crossinvoke GetRolesByType error")).Times(1)
	roles := make([]*Role, 0)
	roles = append(roles, &Role{
		Status: governance.GovernanceAvailable,
	})
	rolesData, err := json.Marshal(roles)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin))).Return(boltvm.Success(rolesData)).AnyTimes()

	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "SubmitProposal error")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().Caller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	// promission error
	res := g.UpdateAllProposalStrategy(string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.False(t, res.Ok)

	// update updating strategy error
	res = g.UpdateAllProposalStrategy(string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.False(t, res.Ok)

	// not update error
	res = g.UpdateAllProposalStrategy(string(SimpleMajority), "a==t", "123")
	assert.False(t, res.Ok)

	// GetRolesByType error
	res = g.UpdateAllProposalStrategy(string(ZeroPermission), "", "123")
	assert.False(t, res.Ok, string(res.Result))

	// illegal strategy info error
	res = g.UpdateAllProposalStrategy("", repo.DefaultSimpleMajorityExpression, "123")
	assert.False(t, res.Ok)

	// submit proposal error
	res = g.UpdateAllProposalStrategy(string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.False(t, res.Ok)

	// ok
	res = g.UpdateAllProposalStrategy(string(SimpleMajority), repo.DefaultSimpleMajorityExpression, "123")
	assert.True(t, res.Ok)
}

func TestGovStrategy_UpdateProposalStrategyByRolesChange(t *testing.T) {
	g, mockStub, strategies, _ := proposalStrategyPrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RoleContractAddr.Address().String()).AnyTimes()

	// 0: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[7]).Return(true).AnyTimes()
	// 1: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[1].Module), gomock.Any()).SetArg(1, *strategies[1]).Return(true).AnyTimes()
	// 2: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[2].Module), gomock.Any()).SetArg(1, *strategies[2]).Return(true).AnyTimes()
	// 3: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[3].Module), gomock.Any()).SetArg(1, *strategies[3]).Return(true).AnyTimes()
	// 4: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[4].Module), gomock.Any()).SetArg(1, *strategies[4]).Return(true).AnyTimes()
	// 5: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[5].Module), gomock.Any()).SetArg(1, *strategies[5]).Return(true).AnyTimes()
	// 6: available/a==5
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[6].Module), gomock.Any()).SetArg(1, *strategies[14]).Return(true).AnyTimes()

	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	// promission error
	res := g.UpdateProposalStrategyByRolesChange(4)
	assert.False(t, res.Ok)

	res = g.UpdateProposalStrategyByRolesChange(4)
	assert.True(t, res.Ok)
}

func TestGovStrategy_Manage(t *testing.T) {
	g, mockStub, strategies, _ := proposalStrategyPrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()

	// 0: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[0]).Return(true).Times(2)

	// 0: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[7]).Return(true).AnyTimes()
	// 1: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[1].Module), gomock.Any()).SetArg(1, *strategies[8]).Return(true).AnyTimes()
	// 2: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[2].Module), gomock.Any()).SetArg(1, *strategies[9]).Return(true).AnyTimes()
	// 3: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[3].Module), gomock.Any()).SetArg(1, *strategies[10]).Return(true).AnyTimes()
	// 4: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[4].Module), gomock.Any()).SetArg(1, *strategies[11]).Return(true).AnyTimes()
	// 5: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[5].Module), gomock.Any()).SetArg(1, *strategies[12]).Return(true).AnyTimes()
	// 6: updating
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[6].Module), gomock.Any()).SetArg(1, *strategies[13]).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin))).Return(boltvm.Error("", "crossinvoke GetRolesByType error")).Times(1)
	roles := make([]*Role, 0)
	roles = append(roles, &Role{
		Status: governance.GovernanceAvailable,
	})
	rolesData, err := json.Marshal(roles)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin))).Return(boltvm.Success(rolesData)).AnyTimes()

	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().Caller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	updateStrategyInfo := map[string]UpdateStrategyInfo{
		strategies[0].Module: UpdateStrategyInfo{
			Typ: UpdateInfo{
				OldInfo: ZeroPermission,
				NewInfo: SimpleMajority,
				IsEdit:  true,
			},
			Extra: UpdateInfo{
				OldInfo: "",
				NewInfo: repo.DefaultSimpleMajorityExpression,
				IsEdit:  true,
			},
		},
	}
	extra, err := json.Marshal(updateStrategyInfo)
	assert.Nil(t, err)

	updateStrategyInfo1 := map[string]UpdateStrategyInfo{
		strategies[0].Module: UpdateStrategyInfo{
			Typ: UpdateInfo{
				OldInfo: ZeroPermission,
				NewInfo: SimpleMajority,
				IsEdit:  true,
			},
			Extra: UpdateInfo{
				OldInfo: "",
				NewInfo: "a==4",
				IsEdit:  true,
			},
		},
	}
	extra1, err := json.Marshal(updateStrategyInfo1)
	assert.Nil(t, err)

	updateAllStrategyInfo := map[string]UpdateStrategyInfo{}
	for _, mgr := range mgrs {
		updateAllStrategyInfo[mgr] = UpdateStrategyInfo{
			Typ: UpdateInfo{
				OldInfo: SimpleMajority,
				NewInfo: ZeroPermission,
				IsEdit:  true,
			},
			Extra: UpdateInfo{
				OldInfo: repo.DefaultSimpleMajorityExpression,
				NewInfo: "",
				IsEdit:  true,
			},
		}
	}
	allExtra, err := json.Marshal(updateAllStrategyInfo)
	assert.Nil(t, err)

	// promission error
	res := g.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), strategies[0].Module, extra)
	assert.False(t, res.Ok)

	// not all mgr change status error
	res = g.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), strategies[0].Module, extra)
	assert.False(t, res.Ok)

	// all mgr change status error
	res = g.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), repo.AllMgr, extra)
	assert.False(t, res.Ok)

	// GetRolesByType error
	res = g.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), strategies[0].Module, extra)
	assert.False(t, res.Ok)

	// update to default
	res = g.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), strategies[0].Module, extra1)
	assert.True(t, res.Ok)

	res = g.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), strategies[0].Module, extra)
	assert.True(t, res.Ok)

	res = g.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), repo.AllMgr, allExtra)
	assert.True(t, res.Ok)
}

func TestGovStrategy_GetAllProposalStrategy(t *testing.T) {
	g, mockStub, strategies, _ := proposalStrategyPrepare(t)

	// 0: not set
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).Return(false).Times(1)
	// 0: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[0]).Return(true).AnyTimes()
	// 1: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[1].Module), gomock.Any()).SetArg(1, *strategies[1]).Return(true).AnyTimes()
	// 2: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[2].Module), gomock.Any()).SetArg(1, *strategies[2]).Return(true).AnyTimes()
	// 3: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[3].Module), gomock.Any()).SetArg(1, *strategies[3]).Return(true).AnyTimes()
	// 4: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[4].Module), gomock.Any()).SetArg(1, *strategies[4]).Return(true).AnyTimes()
	// 5: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[5].Module), gomock.Any()).SetArg(1, *strategies[5]).Return(true).AnyTimes()
	// 6: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[6].Module), gomock.Any()).SetArg(1, *strategies[6]).Return(true).AnyTimes()

	res := g.GetAllProposalStrategy()
	assert.True(t, res.Ok)

	res = g.GetAllProposalStrategy()
	assert.True(t, res.Ok)
}

func TestGovStrategy_GetProposalStrategy(t *testing.T) {
	g, mockStub, strategies, _ := proposalStrategyPrepare(t)

	// 0: not set
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).Return(false).Times(1)
	// 0: available
	mockStub.EXPECT().GetObject(ProposalStrategyKey(strategies[0].Module), gomock.Any()).SetArg(1, *strategies[0]).Return(true).AnyTimes()

	res := g.GetProposalStrategy(strategies[0].Module)
	assert.True(t, res.Ok)

	res = g.GetProposalStrategy(strategies[0].Module)
	assert.True(t, res.Ok)
}

func proposalStrategyPrepare(t *testing.T) (*GovStrategy, *mock_stub.MockStub, []*ProposalStrategy, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	g := &GovStrategy{
		Stub: mockStub,
	}

	strategies := make([]*ProposalStrategy, 0)
	strategiesData := make([][]byte, 0)
	for i := 0; i < 7; i++ {
		ps := &ProposalStrategy{
			Module: mgrs[i],
			Typ:    SimpleMajority,
			Status: governance.GovernanceAvailable,
			Extra:  "a == t",
		}
		strategies = append(strategies, ps)
		data, err := json.Marshal(ps)
		assert.Nil(t, err)
		strategiesData = append(strategiesData, data)
	}

	for i := 0; i < 7; i++ {
		ps := &ProposalStrategy{
			Module: mgrs[i],
			Typ:    ZeroPermission,
			Status: governance.GovernanceUpdating,
			Extra:  "",
		}
		strategies = append(strategies, ps)

		data, err := json.Marshal(ps)
		assert.Nil(t, err)
		strategiesData = append(strategiesData, data)
	}

	ps := &ProposalStrategy{
		Module: mgrs[0],
		Typ:    SimpleMajority,
		Status: governance.GovernanceAvailable,
		Extra:  "a==5",
	}
	strategies = append(strategies, ps)

	data, err := json.Marshal(ps)
	assert.Nil(t, err)
	strategiesData = append(strategiesData, data)

	return g, mockStub, strategies, strategiesData
}
