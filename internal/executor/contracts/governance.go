package contracts

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/meshplus/bitxhub-core/boltvm"
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

	AppchainMgr ProposalType = "AppchainMgr"
	RuleMgr     ProposalType = "RuleMgr"
	NodeMgr     ProposalType = "NodeMgr"
	ServiceMgr  ProposalType = "ServiceMgr"

	PROPOSED ProposalStatus = "proposed"
	APPOVED  ProposalStatus = "approve"
	REJECTED ProposalStatus = "reject"

	BallotApprove = "approve"
	BallotReject  = "reject"
)

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
	Id     string         `json:"id"`
	Des    string         `json:"des"`
	Typ    ProposalType   `json:"typ"`
	Status ProposalStatus `json:"status"`
	// ballot information: voter address -> ballot
	BallotMap     map[string]Ballot `json:"ballot_map"`
	ApproveNum    uint64            `json:"approve_num"`
	AgainstNum    uint64            `json:"against_num"`
	ElectorateNum uint64            `json:"electorate_num"`
	ThresholdNum  uint64            `json:"threshold_num"`
	Extra         []byte            `json:"extra"`
}

func (g *Governance) SubmitProposal(from, des string, typ string, extra []byte) *boltvm.Response {
	specificAddrs := []string{constant.AppchainMgrContractAddr.Address().String()}
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

	ret, err := g.getProposalsByFrom(from)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	en, err := g.getElectorateNum()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	tn, err := g.getThresholdNum(en, ProposalType(typ))
	if err != nil {
		return boltvm.Error(err.Error())
	}

	p := &Proposal{
		Id:            from + "-" + strconv.Itoa(len(ret)),
		Des:           des,
		Typ:           ProposalType(typ),
		Status:        PROPOSED,
		BallotMap:     make(map[string]Ballot, 0),
		ApproveNum:    0,
		AgainstNum:    0,
		ElectorateNum: en,
		ThresholdNum:  tn,
		Extra:         extra,
	}

	g.AddObject(ProposalKey(p.Id), *p)

	return boltvm.Success([]byte(p.Id))
}

func (g *Governance) getElectorateNum() (uint64, error) {
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
		electorateNum = electorateNum + admin.Weight
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
		ps.ParticipateThreshold = 0.75
		g.AddObject(string(proposalTyp), ps)
	}

	return uint64(math.Ceil(float64(electorateNum) * ps.ParticipateThreshold)), nil
}

// ModifyProposal modify a proposal
func (g *Governance) ModifyProposal(id, des string, typ string, extra []byte) *boltvm.Response {
	en, err := g.getElectorateNum()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	tn, err := g.getThresholdNum(en, ProposalType(typ))
	if err != nil {
		return boltvm.Error(err.Error())
	}

	p := &Proposal{
		Id:            id,
		Des:           des,
		Typ:           ProposalType(typ),
		Status:        PROPOSED,
		BallotMap:     make(map[string]Ballot, 0),
		ApproveNum:    0,
		AgainstNum:    0,
		ElectorateNum: en,
		ThresholdNum:  tn,
		Extra:         extra,
	}

	if err := checkProposalInfo(p); err != nil {
		return boltvm.Error(err.Error())
	}

	if !g.Has(ProposalKey(p.Id)) {
		return boltvm.Error(fmt.Sprintf("proposal does not exists"))
	}

	g.SetObject(ProposalKey(p.Id), *p)
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
	switch p.Typ {
	case RuleMgr, NodeMgr, ServiceMgr:
		return boltvm.Error("waiting for subsequent implementation")
	default: // APPCHAIN_MGR
		res := g.CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manager", pb.String(p.Des), pb.String(string(p.Status)), pb.Bytes(p.Extra))
		if !res.Ok {
			return boltvm.Error("cross invoke Manager error:" + string(res.Result))
		}
		return boltvm.Success(nil)
	}
}

// Set vote of an administrator
func (g *Governance) setVote(p *Proposal, addr string, approve string, reason string) error {
	// Determine if the proposal has been approved or rejected
	if p.Status != PROPOSED {
		return fmt.Errorf("the vote on the proposal has been closed")
	}

	// Determine if the administrator has voted
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

	// Record Voting Information
	ballot := Ballot{
		VoterAddr: addr,
		Approve:   approve,
		Num:       uint64(num),
		Reason:    reason,
	}
	p.BallotMap[addr] = ballot
	switch approve {
	case BallotApprove:
		p.ApproveNum = p.ApproveNum + uint64(num)
	case BallotReject:
		p.AgainstNum = p.AgainstNum + uint64(num)
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
		ps.ParticipateThreshold = 0.75
		g.SetObject(string(p.Typ), ps)
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
		} else {
			p.Status = REJECTED
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
func checkProposalInfo(p *Proposal) error {
	if checkProposalType(p.Typ) != nil || strings.Index(p.Id, "-") == -1 || p.Id[0:strings.Index(p.Id, "-")] == "" {
		return fmt.Errorf("illegal proposal info")
	}

	return nil
}

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
