package contracts

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
)

type Governance struct {
	boltvm.Stub
}

type ProposalType string
type ProposalStatus string

const (
	PROPOSAL_PREFIX = "proposal-"

	AppchainMgr         ProposalType = "AppchainMgr"
	RuleMgr             ProposalType = "RuleMgr"
	NodeMgr             ProposalType = "NodeMgr"
	ServiceMgr          ProposalType = "ServiceMgr"
	RoleMgr             ProposalType = "RoleMgr"
	ProposalStrategyMgr ProposalType = "ProposalStrategyMgr"

	PROPOSED ProposalStatus = "proposed"
	APPOVED  ProposalStatus = "approve"
	REJECTED ProposalStatus = "reject"
	PAUSED   ProposalStatus = "pause"

	BallotApprove = "approve"
	BallotReject  = "reject"
)

var priority = map[governance.EventType]int{
	governance.EventRegister: 3,
	governance.EventUpdate:   1,
	governance.EventFreeze:   2,
	governance.EventActivate: 1,
	governance.EventBind:     1,
	governance.EventUnbind:   1,
	governance.EventLogout:   3,
}

type Ballot struct {
	VoterAddr string `json:"voter_addr"`
	Approve   string `json:"approve"`
	Num       uint64 `json:"num"`
	Reason    string `json:"reason"`
}

func (g *Governance) GetBallot(voterAddr, proposalId string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(proposalId), p) {
		return boltvm.Error("proposal does not exist")
	}

	ballot, ok := p.BallotMap[voterAddr]
	if !ok {
		return boltvm.Error("administrator of the address has not voted")
	}

	bData, err := json.Marshal(ballot)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(bData)
}

type Proposal struct {
	Id            string                      `json:"id"`
	Des           string                      `json:"des"`
	Typ           ProposalType                `json:"typ"`
	Status        ProposalStatus              `json:"status"`
	ObjId         string                      `json:"obj_id"`
	ObjLastStatus governance.GovernanceStatus `json:"obj_last_status"`
	// ballot information: voter address -> ballot
	BallotMap         map[string]Ballot    `json:"ballot_map"`
	ApproveNum        uint64               `json:"approve_num"`
	AgainstNum        uint64               `json:"against_num"`
	ElectorateNum     uint64               `json:"electorate_num"`
	ThresholdNum      uint64               `json:"threshold_num"`
	EventType         governance.EventType `json:"event_type"`
	EndReason         string               `json:"end_reason"`
	LockProposalId    string               `json:"lock_proposal_id"`
	IsSpecial         bool                 `json:"is_special"`
	IsSuperAdminVoted bool                 `json:"is_super_admin_voted"`
	CreateTime        int64                `json:"create_time"`
	Extra             []byte               `json:"extra"`
}

var SpecialProposalEventType = []governance.EventType{
	governance.EventFreeze,
	governance.EventActivate,
	governance.EventLogout,
}

var SpecialProposalProposalType = []ProposalType{
	RoleMgr,
	ProposalStrategyMgr,
}

func (g *Governance) SubmitProposal(from, eventTyp, des, typ, objId, objLastStatus string, extra []byte) *boltvm.Response {

	// 1. check permission
	specificAddrs := []string{
		constant.AppchainMgrContractAddr.Address().String(),
		constant.RuleManagerContractAddr.Address().String(),
		constant.NodeManagerContractAddr.Address().String(),
	}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + string(err.Error()))
	}
	res := g.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSpecific)),
		pb.String(""),
		pb.String(g.CurrentCaller()),
		pb.Bytes(addrsData))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	// 2. get information
	ret, err := g.getProposalsByFrom(from)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	en, err := g.getElectorateNum(from)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	tn, err := g.getThresholdNum(en, ProposalType(typ))
	if err != nil {
		return boltvm.Error(err.Error())
	}

	// 3. lock low-priority proposals
	lockPId, err := g.lockLowPriorityProposal(objId, eventTyp)
	if err != nil {
		return boltvm.Error("close low priority proposals error:" + err.Error())
	}

	p := &Proposal{
		Id:                from + "-" + strconv.Itoa(len(ret)),
		EventType:         governance.EventType(eventTyp),
		Des:               des,
		Typ:               ProposalType(typ),
		Status:            PROPOSED,
		ObjId:             objId,
		ObjLastStatus:     governance.GovernanceStatus(objLastStatus),
		BallotMap:         make(map[string]Ballot, 0),
		ApproveNum:        0,
		AgainstNum:        0,
		ElectorateNum:     en,
		ThresholdNum:      tn,
		LockProposalId:    lockPId,
		IsSuperAdminVoted: false,
		CreateTime:        g.GetTxTimeStamp(),
		Extra:             extra,
	}
	p.IsSpecial = isSpecialProposal(p)

	g.AddObject(ProposalKey(p.Id), *p)

	return boltvm.Success([]byte(p.Id))
}

func isSpecialProposal(p *Proposal) bool {
	for _, pt := range SpecialProposalProposalType {
		if pt == p.Typ {
			return true
		}
	}
	for _, et := range SpecialProposalEventType {
		if et == p.EventType {
			return true
		}
	}

	return false
}

func (g *Governance) lockLowPriorityProposal(objId, eventTyp string) (string, error) {
	proposals, err := g.getProposalsByObjId(objId)
	if err != nil {
		return "", err
	}

	for _, p := range proposals {
		if p.Status == PROPOSED {
			if priority[p.EventType] < priority[governance.EventType(eventTyp)] {
				p.Status = PAUSED
				g.SetObject(ProposalKey(p.Id), *p)
				return p.Id, nil
			} else {
				return "", fmt.Errorf("an equal or higher priority proposal is in progress currently, please submit it later")
			}
		}
	}

	return "", nil
}

func (g *Governance) getElectorateNum(from string) (uint64, error) {
	res := g.CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles")
	if !res.Ok {
		return 0, fmt.Errorf("get admin roles error: %s", string(res.Result))
	}

	var admins []*repo.Admin
	if err := json.Unmarshal(res.Result, &admins); err != nil {
		return 0, fmt.Errorf(err.Error())
	}

	electorateNum := uint64(0)
	for _, admin := range admins {
		if from != admin.Address {
			electorateNum++
		}
	}
	return electorateNum, nil
}

func (g *Governance) getThresholdNum(electorateNum uint64, proposalTyp ProposalType) (uint64, error) {
	if err := checkProposalType(proposalTyp); err != nil {
		return 0, fmt.Errorf(err.Error())
	}
	ps := ProposalStrategy{}
	if !g.GetObject(string(proposalTyp), &ps) {
		// SimpleMajority is used by default
		ps.Typ = SimpleMajority
		ps.ParticipateThreshold = repo.DefaultParticipateThreshold
		g.AddObject(string(proposalTyp), ps)
	}

	return uint64(math.Ceil(float64(electorateNum) * ps.ParticipateThreshold)), nil
}

// Withdraw the proposal
func (g *Governance) WithdrawProposal(id string) *boltvm.Response {
	// 1. check permission
	res := g.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelf)),
		pb.String(id[0:strings.Index(id, "-")]),
		pb.String(g.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	// 2. Determine if the proposal exists
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	// 3。 Determine if the proposal has been cloesd
	if p.Status == APPOVED || p.Status == REJECTED {
		return boltvm.Error("the current status of the proposal is " + string(p.Status) + " and cannot be withdrawed")
	}

	// 4. Withdraw
	p.Status = REJECTED
	p.EndReason = "withdrawn by the proposal sponsor"
	g.SetObject(ProposalKey(p.Id), *p)

	// 5. handel result
	err := g.handleResult(p)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(nil)
}

// GetProposal query proposal by id
func (g *Governance) GetProposal(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	pData, err := json.Marshal(p)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(pData)
}

// Query proposals by the ID of the managed chain, returning a list of proposal for that type
func (g *Governance) GetProposalsByObjId(objId string) *boltvm.Response {
	ret, err := g.getProposalsByObjId(objId)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(retData)
}

func (g *Governance) getProposalsByObjId(objId string) ([]*Proposal, error) {
	ok, datas := g.Query(PROPOSAL_PREFIX)
	if !ok {
		return make([]*Proposal, 0), nil
	}

	ret := make([]*Proposal, 0)
	for _, d := range datas {
		p := &Proposal{}
		if err := json.Unmarshal(d, &p); err != nil {
			return nil, err
		}

		if objId == p.ObjId {
			ret = append(ret, p)
		}
	}

	return ret, nil
}

// Query proposals by proposal type, returning a list of proposal for that type
func (g *Governance) GetProposalsByFrom(from string) *boltvm.Response {
	ret, err := g.getProposalsByFrom(from)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(retData)
}

func (g *Governance) getProposalsByFrom(from string) ([]Proposal, error) {
	ok, datas := g.Query(PROPOSAL_PREFIX)
	if !ok {
		return make([]Proposal, 0), nil
	}

	ret := make([]Proposal, 0)
	for _, d := range datas {
		p := Proposal{}
		if err := json.Unmarshal(d, &p); err != nil {
			return nil, err
		}

		if from == p.Id[0:strings.Index(p.Id, "-")] {
			ret = append(ret, p)
		}
	}

	return ret, nil
}

// Query proposals by proposal type, returning a list of proposal for that type
func (g *Governance) GetProposalsByTyp(typ string) *boltvm.Response {
	if err := checkProposalType(ProposalType(typ)); err != nil {
		return boltvm.Error(err.Error())
	}

	ret := make([]Proposal, 0)

	ok, datas := g.Query(PROPOSAL_PREFIX)
	if ok {
		for _, d := range datas {
			p := Proposal{}
			if err := json.Unmarshal(d, &p); err != nil {
				return boltvm.Error(err.Error())
			}

			if ProposalType(typ) == p.Typ {
				ret = append(ret, p)
			}
		}
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(retData)
}

// Query proposals based on proposal status, returning a list of proposal for that status
func (g *Governance) GetProposalsByStatus(status string) *boltvm.Response {
	if err := checkProposalStauts(ProposalStatus(status)); err != nil {
		return boltvm.Error(err.Error())
	}

	ret := make([]Proposal, 0)

	ok, datas := g.Query(PROPOSAL_PREFIX)
	if ok {
		for _, d := range datas {
			p := Proposal{}
			if err := json.Unmarshal(d, &p); err != nil {
				return boltvm.Error(err.Error())
			}

			if ProposalStatus(status) == p.Status {
				ret = append(ret, p)
			}
		}
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(retData)
}

// Get proposal description information
func (g *Governance) GetDes(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(p.Des))
}

// Get Proposal Type
func (g *Governance) GetTyp(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(p.Typ))
}

// Get proposal status
func (g *Governance) GetStatus(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(p.Status))
}

// Get affirmative vote information
func (g *Governance) GetApprove(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	approveMap := map[string]Ballot{}
	for k, v := range p.BallotMap {
		if v.Approve == BallotApprove {
			approveMap[k] = v
		}
	}

	retData, err := json.Marshal(approveMap)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(retData)
}

// Get negative vote information
func (g *Governance) GetAgainst(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	againstMap := map[string]Ballot{}
	for k, v := range p.BallotMap {
		if v.Approve == BallotReject {
			againstMap[k] = v
		}
	}

	retData, err := json.Marshal(againstMap)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(retData)
}

// Get the total number of affirmative votes
func (g *Governance) GetApproveNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.ApproveNum))))
}

// Get the total number of negative votes
func (g *Governance) GetAgainstNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.AgainstNum))))
}

// Get the number of total votes, include all votes cast and all votes not cast
func (g *Governance) GetElectorateNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.ElectorateNum))))
}

// Get the minimum number of votes required for the current voting strategy
func (g *Governance) GetThresholdNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.ThresholdNum))))
}

// Get the number of people who have voted
func (g *Governance) GetVotedNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	return boltvm.Success([]byte(strconv.Itoa(len(p.BallotMap))))
}

// Get voted information
func (g *Governance) GetVoted(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	retData, err := json.Marshal(p.BallotMap)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(retData)
}

// Get Unvoted information
func (g *Governance) GetUnvote(id string) *boltvm.Response {
	ret, err := g.getUnvote(id)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(retData)
}

func (g *Governance) getUnvote(id string) ([]*repo.Admin, error) {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return nil, fmt.Errorf("proposal does not exist")
	}

	res := g.CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles")
	if !res.Ok {
		return nil, fmt.Errorf("get admin roles error: " + string(res.Result))
	}
	var admins []*repo.Admin
	if err := json.Unmarshal(res.Result, &admins); err != nil {
		return nil, fmt.Errorf("get admin roles error: " + err.Error())
	}

	ret := make([]*repo.Admin, 0)
	for _, admin := range admins {
		if _, ok := p.BallotMap[admin.Address]; !ok {
			ret = append(ret, admin)
		}
	}

	return ret, nil
}

// Get Unvoted information
func (g *Governance) GetUnvoteNum(id string) *boltvm.Response {
	ret, err := g.getUnvote(id)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success([]byte(strconv.Itoa(len(ret))))
}

// Add someone's voting information (each person can only vote once)
func (g *Governance) Vote(id, approve string, reason string) *boltvm.Response {
	addr := g.Caller()
	// 1. Determine if the proposal exists
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error("proposal does not exist")
	}

	// 2. Set vote
	if err := g.setVote(p, addr, approve, reason); err != nil {
		return boltvm.Error("get vote error: " + err.Error())
	}

	// 3. Count votes
	// If the threshold for participation is reached, the result of the vote can be judged.
	// If the policy determines that the current vote has closed, the proposal state is modified.
	ok, err := g.countVote(p)
	if err != nil {
		return boltvm.Error("count vote error: " + err.Error())
	}
	if !ok {
		// the round of the voting is not over, wait the next vote
		return boltvm.Success(nil)
	}

	// 4. Handle result
	if err = g.handleResult(p); err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(nil)
}

func (g *Governance) handleResult(p *Proposal) error {
	// Unlock low-priority proposal
	nextEventType, err := g.unlockLowPriorityProposal(p)
	if err != nil {
		return fmt.Errorf("unlock low priority proposals error: %v", err)
	}

	// manage object
	switch p.Typ {
	case ServiceMgr:
		return fmt.Errorf("waiting for subsequent implementation")
	case NodeMgr:
		res := g.CrossInvoke(constant.NodeManagerContractAddr.String(), "Manage", pb.String(string(p.EventType)), pb.String(string(nextEventType)), pb.String(string(p.ObjLastStatus)), pb.Bytes(p.Extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	case RuleMgr:
		res := g.CrossInvoke(constant.RuleManagerContractAddr.String(), "Manage", pb.String(string(p.EventType)), pb.String(string(nextEventType)), pb.String(string(p.ObjLastStatus)), pb.Bytes(p.Extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	default: // APPCHAIN_MGR
		res := g.CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manage", pb.String(string(p.EventType)), pb.String(string(nextEventType)), pb.String(string(p.ObjLastStatus)), pb.Bytes(p.Extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	}
}

func (g *Governance) unlockLowPriorityProposal(p *Proposal) (governance.EventType, error) {
	nextEventType := governance.EventType(p.Status)

	if p.LockProposalId != "" {
		lockP := &Proposal{}
		if !g.GetObject(ProposalKey(p.LockProposalId), lockP) {
			return "", fmt.Errorf("proposal does not exist")
		}

		if p.Status == APPOVED {
			lockP.Status = REJECTED
			lockP.EndReason = "forced shut down by a high-priority proposal (" + p.Id + ")"
		} else {
			lockP.Status = PROPOSED
			nextEventType = lockP.EventType
		}
		g.SetObject(ProposalKey(lockP.Id), lockP)
	}

	return nextEventType, nil
}

// Set vote of an administrator
func (g *Governance) setVote(p *Proposal, addr string, approve string, reason string) error {
	// 1. Determine if the proposal has been approved or rejected
	if p.Status != PROPOSED {
		return fmt.Errorf("the current status of the proposal is %s and cannot be voted on", p.Status)
	}

	// 2. Determine if the administrator can vote
	if addr == p.Id[0:strings.Index(p.Id, "-")] {
		return fmt.Errorf("sponsors can not vote on their own sponsored proposals")
	}

	// 3. Determine if the administrator has voted
	if _, ok := p.BallotMap[addr]; ok {
		return fmt.Errorf("administrator of the address has voted")
	}

	res := g.CrossInvoke(constant.RoleContractAddr.String(), "GetRoleWeight", pb.String(addr))
	if !res.Ok {
		return fmt.Errorf(string(res.Result))
	}
	num, err := strconv.Atoi(string(res.Result))
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	// 4. Record Voting Information
	ballot := Ballot{
		VoterAddr: addr,
		Approve:   approve,
		Num:       uint64(num),
		Reason:    reason,
	}
	if repo.SuperAdminWeight == num {
		p.IsSuperAdminVoted = true
	}
	p.BallotMap[addr] = ballot
	switch approve {
	case BallotApprove:
		p.ApproveNum++
	case BallotReject:
		p.AgainstNum++
	default:
		return fmt.Errorf("the info of vote should be approve or reject")
	}

	g.SetObject(ProposalKey(p.Id), *p)
	return nil
}

// Count votes to see if this round is over.
// If the vote is over change the status of the proposal.
func (g *Governance) countVote(p *Proposal) (bool, error) {
	// Get proposal strategy
	ps := ProposalStrategy{}
	if !g.GetObject(string(p.Typ), &ps) {
		// SimpleMajority is used by default
		ps.Typ = SimpleMajority
		ps.ParticipateThreshold = repo.DefaultParticipateThreshold
		g.SetObject(string(p.Typ), ps)
	}

	// Special types of proposals require super administrator voting
	if p.IsSpecial {
		if !p.IsSuperAdminVoted {
			return false, nil
		}
	}

	// Determine whether the participation threshold for the strategy has been met
	if p.ApproveNum+p.AgainstNum < p.ThresholdNum {
		return false, nil
	}

	// Votes are counted according to strategy
	switch ps.Typ {
	case SuperMajorityApprove:
		// TODO: SUPER_MAJORITY_APPROVE
		return false, fmt.Errorf("this policy is not supported currently")
	case SuperMajorityAgainst:
		// TODO: SUPER_MAJORITY_AGAINST
		return false, fmt.Errorf("this policy is not supported currently")
	default: // SIMPLE_MAJORITY
		if p.ApproveNum > p.AgainstNum {
			p.Status = APPOVED
			p.EndReason = "end of normal voting"
		} else {
			p.Status = REJECTED
			p.EndReason = "end of normal voting"
		}
		g.SetObject(ProposalKey(p.Id), *p)
		return true, nil
	}
}

// Proposal strategy ===============================================================

type ProposalStrategyType string

const (
	SuperMajorityApprove ProposalStrategyType = "SuperMajorityApprove"
	SuperMajorityAgainst ProposalStrategyType = "SuperMajorityAgainst"
	SimpleMajority       ProposalStrategyType = "SimpleMajority"
)

type ProposalStrategy struct {
	Typ ProposalStrategyType `json:"typ"`
	// The minimum participation threshold.
	// Only when the number of voting participants reaches this proportion,
	// the proposal will take effect. That is, the proposal can be judged
	// according to the voting situation.
	ParticipateThreshold float64 `json:"participate_threshold"`
	Extra                []byte  `json:"extra"`
}

func (g *Governance) NewProposalStrategy(typ string, participateThreshold float64, extra []byte) *boltvm.Response {
	ps := &ProposalStrategy{
		Typ:                  ProposalStrategyType(typ),
		ParticipateThreshold: participateThreshold,
		Extra:                extra,
	}
	if err := checkStrategyInfo(ps); err != nil {
		return boltvm.Error(err.Error())
	}

	pData, err := json.Marshal(ps)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(pData)
}

// set proposal strategy for a proposal type
func (g *Governance) SetProposalStrategy(pt string, psData []byte) *boltvm.Response {
	ps := &ProposalStrategy{}
	if err := json.Unmarshal(psData, ps); err != nil {
		return boltvm.Error(err.Error())
	}

	if err := checkProposalType(ProposalType(pt)); err != nil {
		return boltvm.Error(err.Error())
	}

	if err := checkStrategyInfo(ps); err != nil {
		return boltvm.Error(err.Error())
	}

	g.SetObject(string(pt), *ps)
	return boltvm.Success(nil)
}

func (g *Governance) GetProposalStrategy(pt string) *boltvm.Response {
	if err := checkProposalType(ProposalType(pt)); err != nil {
		return boltvm.Error(err.Error())
	}

	ps := &ProposalStrategy{}
	if !g.GetObject(string(pt), ps) {
		return boltvm.Error("strategy does not exists")
	}

	pData, err := json.Marshal(ps)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(pData)
}

func (g *Governance) GetProposalStrategyType(pt string) *boltvm.Response {
	if err := checkProposalType(ProposalType(pt)); err != nil {
		return boltvm.Error(err.Error())
	}

	ps := &ProposalStrategy{}
	if !g.GetObject(string(pt), ps) {
		return boltvm.Error("strategy does not exists")
	}

	return boltvm.Success([]byte(ps.Typ))
}

// Key ====================================================================
func ProposalKey(id string) string {
	return fmt.Sprintf("%s-%s", PROPOSAL_PREFIX, id)
}

// Check info =============================================================
func checkProposalType(pt ProposalType) error {
	if pt != AppchainMgr &&
		pt != RuleMgr &&
		pt != NodeMgr &&
		pt != ServiceMgr {
		return fmt.Errorf("illegal proposal type")
	}
	return nil
}

func checkProposalStauts(ps ProposalStatus) error {
	if ps != PROPOSED &&
		ps != APPOVED &&
		ps != REJECTED {
		return fmt.Errorf("illegal proposal status")
	}
	return nil
}

func checkStrategyInfo(ps *ProposalStrategy) error {
	if checkStrategyType(ps.Typ) != nil ||
		ps.ParticipateThreshold < 0 ||
		ps.ParticipateThreshold > 1 {
		return fmt.Errorf("illegal proposal strategy info")
	}
	return nil
}

func checkStrategyType(pst ProposalStrategyType) error {
	if pst != SuperMajorityApprove &&
		pst != SuperMajorityAgainst &&
		pst != SimpleMajority {
		return fmt.Errorf("illegal proposal strategy type")
	}
	return nil
}

func getGovernanceRet(proposalID string, extra []byte) *boltvm.Response {
	res1 := governance.GovernanceResult{
		ProposalID: proposalID,
		Extra:      extra,
	}
	resData, err := json.Marshal(res1)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(resData)
}
