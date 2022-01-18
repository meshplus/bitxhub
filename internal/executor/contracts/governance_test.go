package contracts

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

var (
	caller           = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	appchainMethod   = "did:bitxhub:appchain1:."
	appchainMethod2  = "did:bitxhub:appchain2:."
	appchainAdminDID = "did:bitxhub:appchain1:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	relayAdminDID    = "did:bitxhub:relay:0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"
	docAddr          = "/ipfs/QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
	docHash          = "QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
	fakeSig          = []byte("fake signature")
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
		Des:        "des",
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

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "addr1",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr2",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr3",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr4",
			Weight:  1,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)
	adminsErrorData := make([]byte, 0)

	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAvailableRoles", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAvailableRoles", gomock.Any()).Return(boltvm.Success(adminsErrorData)).Times(2)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAvailableRoles", gomock.Any()).Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(1)).AnyTimes()

	// check permission error
	res := g.SubmitProposal("", string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles error
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles unmarshal error
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", "", "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), "reason", []byte{})
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
		Des:                    "des",
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

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "addr1",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr2",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr3",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr4",
			Weight:  1,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)

	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).SetArg(1, proposalExistent).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNonexistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()

	res := g.GetProposal(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposal(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalsByObjId("objId")
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByTyp("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalsByTyp(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByStatus("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalsByStatus(string((PROPOSED)))
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetDes(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetDes(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetTyp(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetTyp(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetStatus(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetStatus(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

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
		Des:                    "des",
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
				ID: addrApproved,
			},
			{
				ID: addrAganisted,
			},
			{
				ID: addrNotVoted,
			},
			{
				ID: addrNotVoted1,
			},
			{
				ID: addrNotVoted2,
			},
			{
				ID: addrNotVoted3,
			},
			{
				ID: addrNotVoted4,
			},
			{
				ID: addrNotVoted5,
			},
			{
				ID: addrNotVoted6,
			},
			{
				ID: addrNotVoted7,
			},
			{
				ID: addrNotVoted8,
			},
			{
				ID: addrNotVoted9,
			},
			{
				ID: addrNotVoted10,
			},
			{
				ID: addrNotVoted11,
			},
			{
				ID: addrNotVoted12,
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
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
			pro.LockProposalId = idNonexistent
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
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
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
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
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
			pro.StrategyExpression = repo.DefaultSimpleMajorityExpression
			pro.AvaliableElectorateNum = 4
			pro.InitialElectorateNum = 4
			pro.ElectorateList = proposalExistent.ElectorateList
			return true
		}).Return(true).AnyTimes()

	mockStub.EXPECT().GetObject(string(AppchainMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(RuleMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(NodeMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(ServiceMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(RoleMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAvailable", pb.String(addrUnavaliable)).Return(boltvm.Error("cross invoke IsAvailable error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAvailable", pb.String(addrUnavaliable)).Return(boltvm.Success([]byte("false"))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAvailable", gomock.Any()).Return(boltvm.Success([]byte("true"))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.NodeManagerContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
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
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()

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
	// 19.service error
	res = g.Vote(idServiceMgr, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
}

func TestGovernance_ProposalStrategy(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}
	mockStub.EXPECT().GetObject(string(AppchainMgr), gomock.Any()).Return(true).AnyTimes()

	res := g.GetProposalStrategy(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategy("")
	assert.False(t, res.Ok, string(res.Result))
}

func TestGovernance_SubmitProposal_LockLowPriorityProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"

	chain := &appchainMgr.Appchain{
		ID:            appchainMethod,
		Status:        governance.GovernanceAvailable,
		Name:          "appchain A",
		Validators:    "",
		ConsensusType: "",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "11111",
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalFreeze := &Proposal{
		Id:         idExistent,
		EventType:  governance.EventFreeze,
		ObjId:      appchainMethod,
		Des:        "des",
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

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "addr1",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr2",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr3",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr4",
			Weight:  1,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAvailableRoles", gomock.Any()).Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(1)).AnyTimes()

	res := g.SubmitProposal(idExistent, string(governance.EventUpdate), "des", string(AppchainMgr), appchainMethod, string(governance.GovernanceAvailable), "reason", chainData)
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventLogout), "des", string(AppchainMgr), appchainMethod, string(governance.GovernanceAvailable), "reason", chainData)
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
		ID:            appchainMethod,
		Status:        governance.GovernanceAvailable,
		Name:          "appchain A",
		Validators:    "",
		ConsensusType: "",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "11111",
	}
	chainData, err := json.Marshal(chain)
	assert.Nil(t, err)

	proposalFreeze := &Proposal{
		Id:             idExistent,
		EventType:      governance.EventFreeze,
		ObjId:          appchainMethod,
		Des:            "des",
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
		ObjId:      appchainMethod,
		Des:        "des",
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

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "addr1",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr2",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr3",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr4",
			Weight:  1,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	//mockStub.EXPECT().Query(gomock.Any()).Return(true, nil).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
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
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

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
