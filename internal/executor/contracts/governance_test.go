package contracts

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
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
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsErrorData)).Times(2)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()

	// check permission error
	res := g.SubmitProposal("", string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles error
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles unmarshal error
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", "", "objId", string(governance.GovernanceUnavailable), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventRegister), "des", string(AppchainMgr), "objId", string(governance.GovernanceUnavailable), []byte{})
	assert.True(t, res.Ok, string(res.Result))

}

func TestGovernance_Proposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	idNonexistent := "idNonexistent-2"
	idClosed := "idClosed-3"
	idNotReachThreshold := "idNotReachThreshold-4"
	idSuperMajorityApprove := "idSuperMajorityApprove-5"
	idSuperMajorityAgainst := "idSuperMajorityAgainst-6"
	idUnupportedType := "idUnsupportedType-7"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	addrNotVoted := "addrNotVoted"
	addrNotVoted1 := "addrNotVoted1"
	addrNotVoted2 := "addrNotVoted2"
	addrNotVoted3 := "addrNotVoted3"
	addrNotVoted4 := "addrNotVoted4"
	addrNotVoted5 := "addrNotVoted5"
	addrNotVoted6 := "addrNotVoted6"

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
		Id:            idExistent,
		Des:           "des",
		Typ:           AppchainMgr,
		Status:        PROPOSED,
		ObjId:         "objId",
		BallotMap:     map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:    1,
		AgainstNum:    1,
		ElectorateNum: 4,
		ThresholdNum:  3,
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

	mockStub.EXPECT().Has(ProposalKey(idExistent)).Return(true).AnyTimes()
	mockStub.EXPECT().Has(ProposalKey(idNonexistent)).Return(false).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).SetArg(1, proposalExistent).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idClosed), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idClosed
			pro.Des = proposalExistent.Des
			pro.Typ = proposalExistent.Typ
			pro.Status = APPOVED
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNonexistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNotReachThreshold), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idNotReachThreshold
			pro.Des = proposalExistent.Des
			pro.Typ = RuleMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = 4
			pro.ThresholdNum = 4
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idSuperMajorityApprove), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idSuperMajorityApprove
			pro.Des = proposalExistent.Des
			pro.Typ = NodeMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = proposalExistent.ElectorateNum
			pro.ThresholdNum = proposalExistent.ThresholdNum
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idSuperMajorityAgainst), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idSuperMajorityAgainst
			pro.Des = proposalExistent.Des
			pro.Typ = NodeMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = proposalExistent.ElectorateNum
			pro.ThresholdNum = proposalExistent.ThresholdNum
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idUnupportedType), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idUnupportedType
			pro.Des = proposalExistent.Des
			pro.Typ = ServiceMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = proposalExistent.ElectorateNum
			pro.ThresholdNum = proposalExistent.ThresholdNum
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(string(AppchainMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(RuleMgr), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			proStrategy := ret.(*ProposalStrategy)
			proStrategy.Typ = SimpleMajority
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(string(NodeMgr), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			proStrategy := ret.(*ProposalStrategy)
			proStrategy.Typ = SuperMajorityApprove
			return true
		}).Return(true).Times(1)
	mockStub.EXPECT().GetObject(string(NodeMgr), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			proStrategy := ret.(*ProposalStrategy)
			proStrategy.Typ = SuperMajorityAgainst
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(string(ServiceMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRoleWeight", gomock.Any()).Return(boltvm.Error("get role weight")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRoleWeight", gomock.Any()).Return(boltvm.Success([]byte(""))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRoleWeight", gomock.Any()).Return(boltvm.Success([]byte(strconv.Itoa(1)))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	res := g.GetProposal(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposal(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalsByObjId("objId")
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

	res = g.GetElectorateNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetElectorateNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetThresholdNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetThresholdNum(idNonexistent)
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
	mockStub.EXPECT().Caller().Return(addrApproved).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrApproved).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted1).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted2).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted3).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted4).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted5).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted6).Times(1)

	// nonexistent error
	res = g.Vote(idNonexistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// closed error
	res = g.Vote(idClosed, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// has voted error
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))

	// get weight error
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// get weight parse int error
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))

	// not reach threshold (approve:1)
	res = g.Vote(idNotReachThreshold, BallotApprove, "")
	assert.True(t, res.Ok, string(res.Result))
	// SuperMajorityApprove (reject:1)
	res = g.Vote(idSuperMajorityApprove, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// SuperMajorityAgainst (approve:2)
	res = g.Vote(idSuperMajorityAgainst, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// UnupportedType (reject:2)
	res = g.Vote(idUnupportedType, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// Manager error (approve:3)
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// reject (reject:3)
	res = g.Vote(idExistent, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// approve (approve:4)
	res = g.Vote(idExistent, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_ProposalStrategy(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}
	ps := &ProposalStrategy{
		Typ:                  SimpleMajority,
		ParticipateThreshold: 0.5,
	}
	psData, err := json.Marshal(ps)
	assert.Nil(t, err)

	psError := &ProposalStrategy{
		Typ:                  SimpleMajority,
		ParticipateThreshold: 1.5,
	}
	psErrorData, err := json.Marshal(psError)
	assert.Nil(t, err)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(string(RuleMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(AppchainMgr), gomock.Any()).Return(true).AnyTimes()

	res := g.NewProposalStrategy(string(SimpleMajority), 0.5, []byte{})
	assert.True(t, res.Ok, string(res.Result))
	res = g.NewProposalStrategy("", 0.5, []byte{})
	assert.False(t, res.Ok, string(res.Result))

	res = g.SetProposalStrategy(string(AppchainMgr), psData)
	assert.True(t, res.Ok, string(res.Result))
	res = g.SetProposalStrategy("", psData)
	assert.False(t, res.Ok, string(res.Result))
	res = g.SetProposalStrategy(string(AppchainMgr), psErrorData)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalStrategy(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategy("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategy(string(RuleMgr))
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalStrategyType(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategyType("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategyType(string(RuleMgr))
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
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()

	res := g.SubmitProposal(idExistent, string(governance.EventUpdate), "des", string(AppchainMgr), appchainMethod, string(governance.GovernanceAvailable), chainData)
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, string(governance.EventLogout), "des", string(AppchainMgr), appchainMethod, string(governance.GovernanceAvailable), chainData)
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
			pro.Status = APPOVED
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	res := g.WithdrawProposal(idExistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.WithdrawProposal(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.WithdrawProposal(idClosed)
	assert.False(t, res.Ok, string(res.Result))

	res = g.WithdrawProposal(idExistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.WithdrawProposal(idExistent)
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
