package contracts

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/iancoleman/orderedmap"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

var (
	caller = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	reason = "reason"
)

func TestGovernance_SubmitProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}
	proposalExistent := &Proposal{
		Id:         idExistent,
		Typ:        AppchainMgr,
		Status:     PROPOSED,
		BallotMap:  map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum: 1,
		AgainstNum: 1,
	}
	pData, err := json.Marshal(proposalExistent)
	assert.Nil(t, err)
	pDatas := make([][]byte, 0)
	pDatas = append(pDatas, pData)

	admins := []*Role{
		&Role{
			ID:       "addr1",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr2",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr3",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr4",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)
	adminsErrorData := make([]byte, 0)

	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRolesByType", gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRolesByType", gomock.Any()).Return(boltvm.Success(adminsErrorData)).Times(2)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRolesByType", gomock.Any()).Return(boltvm.Success(adminsData)).AnyTimes()

	strategy := &ProposalStrategy{Extra: repo.DefaultSimpleMajorityExpression, Typ: SimpleMajority}
	data, _ := json.Marshal(strategy)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.ProposalStrategyMgrContractAddr.String()), gomock.Eq("GetProposalStrategy"), gomock.Any()).Return(boltvm.Success(data)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(1)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pDatas[0]).AnyTimes()

	// check permission error
	res := g.SubmitProposal("", string(governance.EventRegister), string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles error
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles unmarshal error
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "", "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.True(t, res.Ok, string(res.Result))

}

func TestGovernance_QueryProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	idNonexistent := "idNonexistent-2"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	addrNotVoted := "addrNotVoted"

	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}
	proposalExistent := Proposal{
		Id:                     idExistent,
		Typ:                    AppchainMgr,
		Status:                 PROPOSED,
		ObjId:                  "objId",
		BallotMap:              map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:             1,
		AgainstNum:             1,
		InitialElectorateNum:   4,
		AvaliableElectorateNum: 4,
		StrategyExpression:     repo.DefaultSimpleMajorityExpression,
	}

	pData, err := json.Marshal(proposalExistent)
	assert.Nil(t, err)
	pDatas := make([][]byte, 0)
	pDatas = append(pDatas, pData)

	admins := []*Role{
		&Role{
			ID:       "addr1",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr2",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr3",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr4",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)

	idMap := orderedmap.New()
	idMap.Set(idExistent, struct{}{})

	mockStub.EXPECT().GetObject(ProposalObjKey("objId"), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalFromKey("idExistent"), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalTypKey(string(AppchainMgr)), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PROPOSED)), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PAUSED)), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(""), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).SetArg(1, proposalExistent).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNonexistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	res := g.GetProposal(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposal(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalsByObjId("objId")
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByObjIdInCreateTimeOrder("objId")
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByFrom("idExistent")
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByTyp("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalsByTyp(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByStatus("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalsByStatus(string((PROPOSED)))
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetNotClosedProposals()
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetApprove(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetApprove(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetAgainst(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetAgainst(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetVotedNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetVotedNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetVoted(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetVoted(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetApproveNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetApproveNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetAgainstNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetAgainstNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetPrimaryElectorateNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetPrimaryElectorateNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetAvaliableElectorateNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetAvaliableElectorateNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	var v = &Ballot{}
	res = g.GetBallot(addrApproved, idNonexistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetBallot(addrNotVoted, idExistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetBallot(addrApproved, idExistent)
	assert.True(t, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, v)
	assert.Nil(t, err)
	assert.Equal(t, BallotApprove, v.Approve)
	assert.Equal(t, uint64(1), v.Num)
	res = g.GetBallot(addrAganisted, idExistent)
	assert.True(t, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, v)
	assert.Nil(t, err)
	assert.Equal(t, BallotReject, v.Approve)
	assert.Equal(t, uint64(1), v.Num)

	res = g.GetUnvote(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))
	// get admin roles error
	res = g.GetUnvote(idExistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetUnvote(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetUnvoteNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetUnvoteNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_Vote(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PROPOSED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PAUSED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(APPROVED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(REJECTED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	idNonexistent := "idNonexistent-2"
	idClosed := "idClosed-3"
	idNotReachThreshold := "idNotReachThreshold-4"
	idSpecial := "idSpecial-5"
	idLockProposalNonexistent := "idLockProposalNonexistent-7"
	idRoleMgr := "idRoleMgr-8"
	idServiceMgr := "idServiceMgr-9"
	idNodeMgr := "idNodeMgr-10"
	idRuleMgr := "idRuleMgr-11"
	idDappMgr := "idDappMgr-12"
	idStrategyMgr := "idStrategyMgr-13"

	addrUnavaliable := "addrUnavaliable"
	addrCanNotVote := "addrCanNotVote"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	addrNotVoted := "addrNotVoted"
	addrNotVoted1 := "addrNotVoted1"
	addrNotVoted2 := "addrNotVoted2"
	addrNotVoted3 := "addrNotVoted3"
	addrNotVoted4 := "addrNotVoted4"
	addrNotVoted5 := "addrNotVoted5"
	addrNotVoted6 := "addrNotVoted6"
	addrNotVoted7 := "addrNotVoted7"
	addrNotVoted8 := "addrNotVoted8"
	addrNotVoted9 := "addrNotVoted9"
	addrNotVoted10 := "addrNotVoted10"
	addrNotVoted11 := "addrNotVoted11"
	addrNotVoted12 := "addrNotVoted12"
	addrNotVoted13 := "addrNotVoted13"
	addrNotVoted14 := "addrNotVoted14"

	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}
	proposalExistent := Proposal{
		Id:                     idExistent,
		Typ:                    AppchainMgr,
		Status:                 PROPOSED,
		ObjId:                  "objId",
		IsSpecial:              false,
		BallotMap:              map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		LockProposalId:         idNotReachThreshold,
		ApproveNum:             1,
		AgainstNum:             1,
		InitialElectorateNum:   4,
		AvaliableElectorateNum: 4,
		StrategyExpression:     repo.DefaultSimpleMajorityExpression,
		ElectorateList: []*Role{
			{
				ID:     addrApproved,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrAganisted,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted1,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted2,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted3,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted4,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted5,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted6,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted7,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted8,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted9,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted10,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted11,
				Status: governance.GovernanceAvailable,
			},
			{
				ID:     addrNotVoted12,
				Status: governance.GovernanceAvailable,
			},
			{
				ID: addrNotVoted13,
			},
			{
				ID: addrNotVoted14,
			},
		},
	}

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).SetArg(1, proposalExistent).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idClosed), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idClosed
			pro.Status = APPROVED
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNonexistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idSpecial), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idSpecial
			pro.Typ = AppchainMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.IsSpecial = true
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.StrategyType = SimpleMajority
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			pro.InitialElectorateNum = 4
			pro.AvaliableElectorateNum = 4
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNotReachThreshold), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idNotReachThreshold
			pro.Typ = AppchainMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 1
			pro.AgainstNum = 1
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idLockProposalNonexistent), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idLockProposalNonexistent
			pro.Typ = AppchainMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 2
			pro.AgainstNum = 1
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.LockProposalId = idNonexistent
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idRoleMgr), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idRoleMgr
			pro.Typ = RoleMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 1
			pro.AgainstNum = 1
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNodeMgr), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idNodeMgr
			pro.Typ = NodeMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 1
			pro.AgainstNum = 1
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idServiceMgr), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idServiceMgr
			pro.Typ = ServiceMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 1
			pro.AgainstNum = 1
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idRuleMgr), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idRuleMgr
			pro.Typ = RuleMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 1
			pro.AgainstNum = 1
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idDappMgr), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idDappMgr
			pro.Typ = DappMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 1
			pro.AgainstNum = 1
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idStrategyMgr), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idStrategyMgr
			pro.Typ = ProposalStrategyMgr
			pro.Status = PROPOSED
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = 1
			pro.AgainstNum = 1
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			return true
		}).Return(true).AnyTimes()

	mockStub.EXPECT().GetObject(string(AppchainMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(RuleMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(NodeMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(ServiceMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(RoleMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(addrUnavaliable), pb.String(string(GovernanceAdmin))).Return(boltvm.Error("", "cross invoke IsAvailable error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(addrUnavaliable), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte("false"))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte("true"))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.DappMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.ProposalStrategyMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Caller().Return(addrUnavaliable).Times(2)
	mockStub.EXPECT().Caller().Return(addrCanNotVote).Times(3)
	mockStub.EXPECT().Caller().Return(addrApproved).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted1).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted2).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted3).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted4).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted5).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted6).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted7).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted8).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted9).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted10).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted11).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted12).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted13).Times(1)
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	pData, err := json.Marshal(proposalExistent)
	assert.Nil(t, err)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	// 1.cross invoke IsAvailable error
	res := g.Vote(idNonexistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// 2.addr is unavailable
	res = g.Vote(idNonexistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// 3.nonexistent error
	res = g.Vote(idNonexistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))

	// setVote
	// 4.closed error
	res = g.Vote(idClosed, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// 5.can not vote
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// 6.has voted error
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// 7.vote info error
	res = g.Vote(idExistent, "", "")
	assert.False(t, res.Ok, string(res.Result))

	// countVote
	// 8.special proposal not end
	res = g.Vote(idSpecial, BallotApprove, "")
	assert.True(t, res.Ok, string(res.Result))

	// handle result
	// 10.lock proposal not existent
	res = g.Vote(idLockProposalNonexistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// 11.rule error
	res = g.Vote(idRuleMgr, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// 12.rule success
	res = g.Vote(idRuleMgr, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// 13.node error
	res = g.Vote(idNodeMgr, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// 14.node success
	res = g.Vote(idNodeMgr, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// 15.node error
	res = g.Vote(idRoleMgr, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// 16.node success
	res = g.Vote(idRoleMgr, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// 17.appchain error
	res = g.Vote(idExistent, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// 18.appchain success
	res = g.Vote(idExistent, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// 19.service success
	res = g.Vote(idServiceMgr, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// 20. dapp success
	res = g.Vote(idDappMgr, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// 21. strategy success
	res = g.Vote(idStrategyMgr, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_SubmitProposal_LockLowPriorityProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"

	chain := &appchainMgr.Appchain{
		ID:      appchainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "",
		Version: 0,
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalFreeze := &Proposal{
		Id:         idExistent,
		EventType:  governance.EventFreeze,
		ObjId:      appchainID,
		Typ:        AppchainMgr,
		Status:     PROPOSED,
		ApproveNum: 1,
		AgainstNum: 1,
		Extra:      chainData,
	}
	pData, err := json.Marshal(proposalFreeze)
	assert.Nil(t, err)

	pDatas := make([][]byte, 0)
	pDatas = append(pDatas, pData)

	admins := []*Role{
		&Role{
			ID:       "addr1",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr2",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr3",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr4",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)
	strategy := &ProposalStrategy{Extra: repo.DefaultSimpleMajorityExpression, Typ: SimpleMajority}
	data, _ := json.Marshal(strategy)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.ProposalStrategyMgrContractAddr.String()), gomock.Eq("GetProposalStrategy"), gomock.Any()).Return(boltvm.Success(data)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.AppchainMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRolesByType", gomock.Any()).Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(1)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	idMap1 := orderedmap.New()
	idMap2 := orderedmap.New()
	idMap2.Set(idExistent, struct{}{})
	mockStub.EXPECT().GetObject(ProposalStatusKey(string((PAUSED))), gomock.Any()).SetArg(1, *idMap1).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string((PROPOSED))), gomock.Any()).SetArg(1, *idMap2).Return(true).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	res := g.SubmitProposal(idExistent, string(governance.EventUpdate), string(AppchainMgr), appchainID, string(governance.GovernanceAvailable), "reason", chainData)
	assert.True(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventLogout), string(AppchainMgr), appchainID, string(governance.GovernanceAvailable), "reason", chainData)
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_WithdrawProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	idExistent2 := "idExistent-4"
	idNonexistent := "idNonexistent-2"
	idClosed := "idClosed-3"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"

	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}

	chain := &appchainMgr.Appchain{
		ID:      appchainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "",
		Version: 0,
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalFreeze := &Proposal{
		Id:             idExistent,
		EventType:      governance.EventFreeze,
		ObjId:          appchainID,
		Typ:            AppchainMgr,
		Status:         PROPOSED,
		BallotMap:      map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:     1,
		AgainstNum:     1,
		LockProposalId: idExistent2,
		Extra:          chainData,
	}
	pData, err := json.Marshal(proposalFreeze)
	assert.Nil(t, err)

	proposalUpdate := &Proposal{
		Id:         idExistent2,
		EventType:  governance.EventUpdate,
		ObjId:      appchainID,
		Typ:        AppchainMgr,
		Status:     PAUSED,
		BallotMap:  map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum: 1,
		AgainstNum: 1,
		Extra:      chainData,
	}
	pData1, err := json.Marshal(proposalUpdate)
	assert.Nil(t, err)

	pDatas := make([][]byte, 0)
	pDatas = append(pDatas, pData, pData1)

	admins := []*Role{
		&Role{
			ID:       "addr1",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr2",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr3",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
		&Role{
			ID:       "addr4",
			RoleType: GovernanceAdmin,
			Status:   governance.GovernanceAvailable,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return("idExistent").AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).SetArg(1, *proposalFreeze).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent2), gomock.Any()).SetArg(1, *proposalUpdate).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNonexistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idClosed), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idClosed
			pro.Status = APPROVED
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	//mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	idMap1 := orderedmap.New()
	idMap2 := orderedmap.New()
	idMap2.Set(idExistent, struct{}{})
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PAUSED)), gomock.Any()).SetArg(1, *idMap1).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(REJECTED)), gomock.Any()).SetArg(1, *idMap1).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(APPROVED)), gomock.Any()).SetArg(1, *idMap1).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PROPOSED)), gomock.Any()).SetArg(1, *idMap2).Return(true).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	res := g.WithdrawProposal(idExistent, "reason")
	assert.False(t, res.Ok, string(res.Result))
	res = g.WithdrawProposal(idNonexistent, "reason")
	assert.False(t, res.Ok, string(res.Result))
	res = g.WithdrawProposal(idClosed, "reason")
	assert.False(t, res.Ok, string(res.Result))

	res = g.WithdrawProposal(idExistent, "reason")
	assert.False(t, res.Ok, string(res.Result))

	res = g.WithdrawProposal(idExistent, "reason")
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_UpdateAvaliableElectorateNum(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	idExistent2 := "idExistent-2"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"

	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}

	chain := &appchainMgr.Appchain{
		ID:      appchainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "",
		Version: 0,
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalFreeze := &Proposal{
		Id:                     idExistent,
		EventType:              governance.EventFreeze,
		ObjId:                  appchainID,
		Typ:                    AppchainMgr,
		Status:                 PROPOSED,
		BallotMap:              map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:             1,
		AgainstNum:             1,
		LockProposalId:         idExistent2,
		Extra:                  chainData,
		StrategyExpression:     repo.DefaultSimpleMajorityExpression,
		InitialElectorateNum:   4,
		AvaliableElectorateNum: 4,
	}

	proposalUpdate := &Proposal{
		Id:                     idExistent2,
		EventType:              governance.EventUpdate,
		ObjId:                  appchainID,
		Typ:                    AppchainMgr,
		Status:                 PROPOSED,
		BallotMap:              map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:             1,
		AgainstNum:             1,
		Extra:                  chainData,
		StrategyExpression:     repo.DefaultSimpleMajorityExpression,
		InitialElectorateNum:   4,
		AvaliableElectorateNum: 4,
	}

	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PROPOSED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PAUSED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(APPROVED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(REJECTED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RoleContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).SetArg(1, *proposalFreeze).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent2), gomock.Any()).SetArg(1, *proposalUpdate).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	pData, err := json.Marshal(proposalFreeze)
	assert.Nil(t, err)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	// check permission error
	res := g.UpdateAvaliableElectorateNum(idExistent, 0)
	assert.False(t, res.Ok, string(res.Result))
	// get proposal id error
	res = g.UpdateAvaliableElectorateNum(idExistent, 0)
	assert.False(t, res.Ok, string(res.Result))

	// subtract num error: manage error
	res = g.UpdateAvaliableElectorateNum(idExistent, 2)
	assert.False(t, res.Ok, string(res.Result))
	// subtract num ok
	res = g.UpdateAvaliableElectorateNum(idExistent, 2)
	assert.True(t, res.Ok, string(res.Result))

	// add num ok
	res = g.UpdateAvaliableElectorateNum(idExistent, 4)
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_LockLowPriorityProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idNotExistent := "idNotExistent-1"
	idExistent2 := "idExistent-2"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"

	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}

	chain := &appchainMgr.Appchain{
		ID:      appchainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "",
		Version: 0,
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalUpdate := &Proposal{
		Id:                 idExistent2,
		EventType:          governance.EventUpdate,
		ObjId:              appchainID,
		Typ:                AppchainMgr,
		Status:             PROPOSED,
		BallotMap:          map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:         1,
		AgainstNum:         1,
		Extra:              chainData,
		StrategyExpression: repo.DefaultSimpleMajorityExpression,
	}

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.ServiceMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNotExistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent2), gomock.Any()).SetArg(1, *proposalUpdate).Return(true).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	idMapErr := orderedmap.New()
	idMapErr.Set(idNotExistent, struct{}{})
	idMapErr.Set(idExistent2, struct{}{})
	idMapOk := orderedmap.New()
	idMapOk.Set(idExistent2, struct{}{})
	mockStub.EXPECT().GetObject(ProposalObjKey(appchainID), gomock.Any()).SetArg(1, *idMapErr).Return(true).Times(1)
	mockStub.EXPECT().GetObject(ProposalObjKey(appchainID), gomock.Any()).SetArg(1, *idMapOk).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PAUSED)), gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(REJECTED)), gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(APPROVED)), gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PROPOSED)), gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	pData, err := json.Marshal(proposalUpdate)
	assert.Nil(t, err)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	// check permission error
	res := g.LockLowPriorityProposal(appchainID, string(governance.EventFreeze))
	assert.False(t, res.Ok, string(res.Result))
	// get proposal id error
	res = g.LockLowPriorityProposal(appchainID, string(governance.EventFreeze))
	assert.False(t, res.Ok, string(res.Result))

	res = g.LockLowPriorityProposal(appchainID, string(governance.EventFreeze))
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_UnLockLowPriorityProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idNotExistent := "idNotExistent-1"
	idExistent2 := "idExistent-2"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"

	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}

	chain := &appchainMgr.Appchain{
		ID:      appchainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "",
		Version: 0,
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalUpdate := &Proposal{
		Id:                 idExistent2,
		EventType:          governance.EventUpdate,
		ObjId:              appchainID,
		Typ:                AppchainMgr,
		Status:             PAUSED,
		BallotMap:          map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:         1,
		AgainstNum:         1,
		Extra:              chainData,
		StrategyExpression: repo.DefaultSimpleMajorityExpression,
	}

	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PROPOSED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(PAUSED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(APPROVED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalStatusKey(string(REJECTED)), gomock.Any()).SetArg(1, *orderedmap.New()).Return(false).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.ServiceMgrContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNotExistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent2), gomock.Any()).SetArg(1, *proposalUpdate).Return(true).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	idMapErr := orderedmap.New()
	idMapErr.Set(idNotExistent, struct{}{})
	idMapErr.Set(idExistent2, struct{}{})
	idMapOk := orderedmap.New()
	idMapOk.Set(idExistent2, struct{}{})
	mockStub.EXPECT().GetObject(ProposalObjKey(appchainID), gomock.Any()).SetArg(1, *idMapErr).Return(true).Times(1)
	mockStub.EXPECT().GetObject(ProposalObjKey(appchainID), gomock.Any()).SetArg(1, *idMapOk).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	pData, err := json.Marshal(proposalUpdate)
	assert.Nil(t, err)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	// check permission error
	res := g.UnLockLowPriorityProposal(appchainID, string(governance.EventFreeze))
	assert.False(t, res.Ok, string(res.Result))
	// get proposal id error
	res = g.UnLockLowPriorityProposal(appchainID, string(governance.EventFreeze))
	assert.False(t, res.Ok, string(res.Result))
	// manage error
	res = g.UnLockLowPriorityProposal(appchainID, string(governance.EventFreeze))
	assert.False(t, res.Ok, string(res.Result))

	res = g.UnLockLowPriorityProposal(appchainID, string(governance.EventFreeze))
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_ZeroPermission(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	g := Governance{mockStub}

	idNotExistent := "idNotExistent-1"
	idExistent2 := "idExistent-2"

	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}

	chain := &appchainMgr.Appchain{
		ID:      appchainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "",
		Version: 0,
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalUpdate := &Proposal{
		Id:           idExistent2,
		EventType:    governance.EventUpdate,
		ObjId:        appchainID,
		Typ:          AppchainMgr,
		Status:       PAUSED,
		BallotMap:    map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:   1,
		AgainstNum:   1,
		Extra:        chainData,
		StrategyType: ZeroPermission,
	}

	mockStub.EXPECT().GetObject(ProposalKey(idNotExistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent2), gomock.Any()).SetArg(1, *proposalUpdate).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	pData, err := json.Marshal(proposalUpdate)
	assert.Nil(t, err)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	// proposal not existent
	res := g.ZeroPermission(idNotExistent)
	assert.False(t, res.Ok, string(res.Result))
	// handle result error
	res = g.ZeroPermission(idExistent2)
	assert.False(t, res.Ok, string(res.Result))
	// ok
	res = g.ZeroPermission(idExistent2)
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_EndObjProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	g := Governance{mockStub}

	idNotExistent := "idNotExistent-1"
	idExistent2 := "idExistent-2"

	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}

	chain := &appchainMgr.Appchain{
		ID:      appchainID,
		Status:  governance.GovernanceAvailable,
		Desc:    "",
		Version: 0,
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalUpdate := &Proposal{
		Id:           idExistent2,
		EventType:    governance.EventUpdate,
		ObjId:        appchainID,
		Typ:          AppchainMgr,
		Status:       PAUSED,
		BallotMap:    map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:   1,
		AgainstNum:   1,
		Extra:        chainData,
		StrategyType: ZeroPermission,
	}

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.ServiceMgrContractAddr.Address().String()).AnyTimes()

	errorIdMap := orderedmap.New()
	errorIdMap.Set(idNotExistent, struct{}{})
	idMap := orderedmap.New()
	idMap.Set(idExistent2, struct{}{})
	mockStub.EXPECT().GetObject(ProposalObjKey(serviceID), gomock.Any()).SetArg(1, *errorIdMap).Return(true).Times(1)
	mockStub.EXPECT().GetObject(ProposalObjKey(serviceID), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNotExistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent2), gomock.Any()).SetArg(1, *proposalUpdate).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	pData, err := json.Marshal(proposalUpdate)
	assert.Nil(t, err)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, pData).AnyTimes()

	// check permission error
	res := g.EndObjProposal(serviceID, string(ClearReason), nil)
	assert.False(t, res.Ok, string(res.Result))
	// get proposals error
	res = g.EndObjProposal(serviceID, string(ClearReason), nil)
	assert.False(t, res.Ok, string(res.Result))
	// ok
	res = g.EndObjProposal(serviceID, string(ClearReason), nil)
	assert.True(t, res.Ok, string(res.Result))
}

func getPubKey(keyPath string) (string, error) {
	privKey, err := asym.RestorePrivateKey(keyPath, "bitxhub")
	if err != nil {
		return "", err
	}

	pubBytes, err := privKey.PublicKey().Bytes()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pubBytes), nil
}
