package contracts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/iancoleman/orderedmap"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

type Governance struct {
	boltvm.Stub
}

type ProposalType string
type ProposalStatus string
type EndReason string

const (
	PROPOSAL_PREFIX         = "proposal"
	PROPOSALFREOM_PREFIX    = "from"
	PROPOSALOBJ_PREFIX      = "obj"
	PROPOSALTYPE_PREFIX     = "type"
	PROPOSALSTRATEGY_PREFIX = "strategy"
	PROPOSALSTATUS_PREFIX   = "status"

	AppchainMgr         ProposalType = repo.AppchainMgr
	RuleMgr             ProposalType = repo.RuleMgr
	NodeMgr             ProposalType = repo.NodeMgr
	ServiceMgr          ProposalType = repo.ServiceMgr
	RoleMgr             ProposalType = repo.RoleMgr
	ProposalStrategyMgr ProposalType = repo.ProposalStrategyMgr
	DappMgr             ProposalType = repo.DappMgr

	PROPOSED ProposalStatus = "proposed"
	APPROVED ProposalStatus = "approve"
	REJECTED ProposalStatus = "reject"
	PAUSED   ProposalStatus = "pause"

	BallotApprove = "approve"
	BallotReject  = "reject"

	NormalReason         EndReason = "end of normal voting"
	ZeroPermissionReason EndReason = "zero permission reason"
	WithdrawnReason      EndReason = "withdrawn by the proposal sponsor"
	PriorityReason       EndReason = "forced shut down by a high-priority proposal"
	ElectorateReason     EndReason = "not enough valid electorate"
	ClearReason          EndReason = "the proposal was cleared"

	FALSE = "false"
	TRUE  = "true"
)

var priority = map[governance.EventType]int{
	governance.EventRegister: 1,
	governance.EventUpdate:   2,
	governance.EventFreeze:   2,
	governance.EventActivate: 1,
	governance.EventBind:     1,
	governance.EventUnbind:   1,
	governance.EventLogout:   3,
	governance.EventPause:    3,
	governance.EventUnpause:  1,
	governance.EventCLear:    3,
}

type Ballot struct {
	VoterAddr string `json:"voter_addr"`
	Approve   string `json:"approve"`
	Num       uint64 `json:"num"`
	Reason    string `json:"reason"`
	VoteTime  int64  `json:"vote_time"`
}

type Proposal struct {
	Id            string                      `json:"id"`
	Typ           ProposalType                `json:"typ"`
	Status        ProposalStatus              `json:"status"`
	ObjId         string                      `json:"obj_id"`
	ObjLastStatus governance.GovernanceStatus `json:"obj_last_status"`
	// ballot information: voter address -> ballot
	BallotMap              map[string]Ballot    `json:"ballot_map"`
	ApproveNum             uint64               `json:"approve_num"`
	AgainstNum             uint64               `json:"against_num"`
	ElectorateList         []*Role              `json:"electorate_list"`
	InitialElectorateNum   uint64               `json:"initial_electorate_num"`
	AvaliableElectorateNum uint64               `json:"avaliable_electorate_num"`
	ThresholdApproveNum    uint64               `json:"threshold_approve_num"`
	EventType              governance.EventType `json:"event_type"`
	EndReason              EndReason            `json:"end_reason"`
	LockProposalId         string               `json:"lock_proposal_id"`
	IsSpecial              bool                 `json:"is_special"`
	IsSuperAdminVoted      bool                 `json:"is_super_admin_voted"`
	SubmitReason           string               `json:"submit_reason"`
	WithdrawReason         string               `json:"withdraw_reason"`
	StrategyType           ProposalStrategyType `json:"strategy_type"`
	StrategyExpression     string               `json:"strategy_expression"`
	CreateTime             int64                `json:"create_time"`
	Extra                  []byte               `json:"extra"`
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

type UpdateInfo struct {
	OldInfo interface{}
	NewInfo interface{}
	IsEdit  bool
}

type UpdateMapInfo struct {
	OldInfo map[string]struct{}
	NewInfo map[string]struct{}
	IsEdit  bool
}

func (g *Governance) addProposal(p *Proposal) {
	g.AddObject(ProposalKey(p.Id), *p)

	// from
	from := strings.Split(p.Id, "-")[0]
	proMap := orderedmap.New()
	_ = g.GetObject(ProposalFromKey(from), proMap)
	proMap.Set(p.Id, struct{}{})
	g.SetObject(ProposalFromKey(from), *proMap)

	// obj
	proMap = orderedmap.New()
	_ = g.GetObject(ProposalObjKey(p.ObjId), proMap)
	proMap.Set(p.Id, struct{}{})
	g.SetObject(ProposalObjKey(p.ObjId), *proMap)

	// type
	proMap = orderedmap.New()
	_ = g.GetObject(ProposalTypKey(string(p.Typ)), proMap)
	proMap.Set(p.Id, struct{}{})
	g.SetObject(ProposalTypKey(string(p.Typ)), *proMap)

	// status
	proMap = orderedmap.New()
	_ = g.GetObject(ProposalStatusKey(string(p.Status)), proMap)
	proMap.Set(p.Id, struct{}{})
	g.SetObject(ProposalStatusKey(string(p.Status)), *proMap)
}

func (g *Governance) changeProposalStatus(p *Proposal, newStatus ProposalStatus) {
	lastStatusMap := orderedmap.New()
	_ = g.GetObject(ProposalStatusKey(string(p.Status)), lastStatusMap)
	lastStatusMap.Delete(p.Id)
	g.SetObject(ProposalStatusKey(string(p.Status)), *lastStatusMap)

	newStatusMap := orderedmap.New()
	_ = g.GetObject(ProposalStatusKey(string(newStatus)), newStatusMap)
	newStatusMap.Set(p.Id, struct{}{})
	g.SetObject(ProposalStatusKey(string(newStatus)), *newStatusMap)

	p.Status = newStatus
	g.SetObject(ProposalKey(p.Id), *p)
}

func (g *Governance) checkPermission(permissions []string, regulatedAddr, regulatorAddr string, specificAddrsData []byte) error {
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			if regulatedAddr == regulatorAddr {
				return nil
			}
		case string(PermissionAdmin):
			res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin",
				pb.String(regulatorAddr),
				pb.String(string(GovernanceAdmin)))
			if !res.Ok {
				return fmt.Errorf("cross invoke IsAvailableGovernanceAdmin error:%s", string(res.Result))
			}
			if "true" == string(res.Result) {
				return nil
			}
		case string(PermissionSpecific):
			specificAddrs := []string{}
			if err := json.Unmarshal(specificAddrsData, &specificAddrs); err != nil {
				return err
			}
			for _, addr := range specificAddrs {
				if addr == regulatorAddr {
					return nil
				}
			}
		default:
			return fmt.Errorf("unsupport permission: %s", permission)
		}
	}

	return fmt.Errorf("regulatorAddr(%s) does not have the permission", regulatorAddr)
}

// =========== SubmitProposal submits kinds of proposal
func (g *Governance) SubmitProposal(from, eventTyp, typ, objId, objLastStatus, reason string, extra []byte) *boltvm.Response {

	// 1. check permission
	specificAddrs := []string{
		constant.AppchainMgrContractAddr.Address().String(),
		constant.RuleManagerContractAddr.Address().String(),
		constant.NodeManagerContractAddr.Address().String(),
		constant.RoleContractAddr.Address().String(),
		constant.DappMgrContractAddr.Address().String(),
		constant.ServiceMgrContractAddr.Address().String(),
		constant.GovernanceContractAddr.Address().String(),
		constant.ProposalStrategyMgrContractAddr.Address().String(),
	}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := g.checkPermission([]string{string(PermissionSpecific)}, "", g.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.GovernanceNoPermissionCode, fmt.Sprintf(string(boltvm.GovernanceNoPermissionMsg), g.CurrentCaller(), err.Error()))
	}

	// 2. get information
	ret, err := g.getProposalsByFrom(from)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	electorateList, eletctorateNum, err := g.getElectorate()
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	expression, strategyTyp, thresholdApproveNum, err := g.getStrategyInfo(ProposalType(typ), uint64(eletctorateNum))
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	// 3. lock low-priority proposals
	lockPId, err := g.lockLowPriorityProposal(objId, eventTyp)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("close low priority proposals error: %v", err)))
	}

	p := &Proposal{
		Id:                     fmt.Sprintf("%s-%s", from, strconv.Itoa(len(ret))),
		EventType:              governance.EventType(eventTyp),
		Typ:                    ProposalType(typ),
		Status:                 PROPOSED,
		ObjId:                  objId,
		ObjLastStatus:          governance.GovernanceStatus(objLastStatus),
		BallotMap:              make(map[string]Ballot, 0),
		ApproveNum:             0,
		AgainstNum:             0,
		ElectorateList:         electorateList,
		InitialElectorateNum:   uint64(eletctorateNum),
		AvaliableElectorateNum: uint64(eletctorateNum),
		ThresholdApproveNum:    uint64(thresholdApproveNum),
		LockProposalId:         lockPId,
		IsSuperAdminVoted:      false,
		SubmitReason:           reason,
		WithdrawReason:         "",
		StrategyType:           strategyTyp,
		StrategyExpression:     expression,
		CreateTime:             g.GetTxTimeStamp(),
		Extra:                  extra,
	}
	p.IsSpecial = isSpecialProposal(p)

	g.addProposal(p)
	g.Logger().WithFields(logrus.Fields{
		"id":         p.Id,
		"objId":      p.ObjId,
		"eventTyp":   p.EventType,
		"typ":        p.Typ,
		"status":     p.Status,
		"createtime": p.CreateTime,
		"extra":      string(p.Extra),
	}).Debug("submit proposal")

	if err := g.postAuditProposalEvent(p.Id); err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
	}

	return boltvm.Success([]byte(p.Id))
}

func (g *Governance) ZeroPermission(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}
	if p.StrategyType == ZeroPermission {
		p.EndReason = ZeroPermissionReason
		g.changeProposalStatus(p, APPROVED)
		if err := g.handleResult(p); err != nil {
			return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
		}
		if err := g.postAuditProposalEvent(p.Id); err != nil {
			return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
		}
	}
	return boltvm.Success(nil)
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

func (g *Governance) getElectorate() ([]*Role, int, error) {
	res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin)))
	if !res.Ok {
		return nil, 0, fmt.Errorf("cross invoke GetRolesByType error: %s", string(res.Result))
	}

	var admins []*Role
	if err := json.Unmarshal(res.Result, &admins); err != nil {
		return nil, 0, fmt.Errorf(err.Error())
	}

	var availableAdmins []*Role
	for _, a := range admins {
		if a.IsAvailable() {
			availableAdmins = append(availableAdmins, a)
		}
	}

	return availableAdmins, len(availableAdmins), nil
}

func (g *Governance) getStrategyInfo(module ProposalType, eletctorateNum uint64) (string, ProposalStrategyType, uint64, error) {
	if err := repo.CheckManageModule(string(module)); err != nil {
		return "", "", 0, fmt.Errorf(err.Error())
	}
	ps := &ProposalStrategy{}
	res := g.CrossInvoke(constant.ProposalStrategyMgrContractAddr.Address().String(), "GetProposalStrategy", pb.String(string(module)))
	if !res.Ok {
		// SimpleMajority is used by default
		ps = defaultStrategy(string(module))
	} else {
		if err := json.Unmarshal(res.Result, &ps); err != nil {
			return "", "", 0, fmt.Errorf("unmashal proposal strategy")
		}
	}

	if ps.Typ == ZeroPermission {
		return ps.Extra, ps.Typ, 0, nil
	} else {
		thresholdApproveNum, err := getCurThresholdApproveNum(ps.Extra, 0, 0, eletctorateNum, eletctorateNum)
		if err != nil {
			return "", "", 0, err
		}
		return ps.Extra, ps.Typ, thresholdApproveNum, nil
	}
}

// only can be call when the strategy type is not ZeroPermission
func getCurThresholdApproveNum(strategyExp string, approve, reject, tolal, availableNum uint64) (uint64, error) {
	thresholdApproveNum := approve
	for ; thresholdApproveNum <= tolal; thresholdApproveNum++ {
		_, isPass, err := repo.MakeStrategyDecision(strategyExp, thresholdApproveNum, reject, tolal, availableNum)
		if err != nil {
			return 0, err
		}
		if isPass {
			return thresholdApproveNum, nil
		}
	}
	return 0, fmt.Errorf("illegal strategy expressopm(%s), the proposal may mot pass", strategyExp)
}

// =========== WithdrawProposal withdraws the designated proposal
func (g *Governance) WithdrawProposal(id, reason string) *boltvm.Response {
	if err := g.checkPermission([]string{string(PermissionSelf)}, id[0:strings.Index(id, "-")], g.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.GovernanceNoPermissionCode, fmt.Sprintf(string(boltvm.GovernanceNoPermissionMsg), g.CurrentCaller(), err.Error()))
	}

	p, berr := g.endProposal(id, string(WithdrawnReason), []byte(reason))
	if berr != nil {
		return boltvm.Error(berr.Code, berr.Error())
	}

	if err := g.handleResult(p); err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	if err := g.postAuditProposalEvent(id); err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
	}
	return boltvm.Success(nil)
}

func (g *Governance) endProposal(id, endReason string, extra []byte) (*Proposal, *boltvm.BxhError) {
	// 1. Determine if the proposal exists
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return nil, boltvm.BError(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	// 2. Determine if the proposal has been cloesd
	if p.Status == APPROVED || p.Status == REJECTED {
		return nil, boltvm.BError(boltvm.GovernanceEndEndedProposalCode, fmt.Sprintf(string(boltvm.GovernanceEndEndedProposalMsg), id, string(p.Status)))
	}

	// 3. End Proposal
	p.EndReason = EndReason(endReason)
	switch EndReason(endReason) {
	case WithdrawnReason:
		p.WithdrawReason = string(extra)
	}
	g.changeProposalStatus(p, REJECTED)
	return p, nil
}

// =========== EndCurrentProposal forces proposals about an object to be ended and does not process other proposals associated with the proposal
func (g *Governance) EndObjProposal(objId, endReason string, extra []byte) *boltvm.Response {
	// 1. check permission
	specificAddrs := []string{
		//constant.AppchainMgrContractAddr.Address().String(),
		constant.RuleManagerContractAddr.Address().String(),
		//constant.NodeManagerContractAddr.Address().String(),
		constant.RoleContractAddr.Address().String(),
		//constant.DappMgrContractAddr.Address().String(),
		constant.ServiceMgrContractAddr.Address().String(),
	}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := g.checkPermission([]string{string(PermissionSpecific)}, "", g.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.GovernanceNoPermissionCode, fmt.Sprintf(string(boltvm.GovernanceNoPermissionMsg), g.CurrentCaller(), err.Error()))
	}

	// 2. get proposals
	proposals, err := g.getProposalsByObjId(objId)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf("getProposalsByObjId err: %v", err))
	}

	// 3. end proposals
	for _, p := range proposals {
		if p.Status == PAUSED || p.Status == PROPOSED {
			if _, berr := g.endProposal(p.Id, endReason, extra); berr != nil {
				return boltvm.Error(berr.Code, berr.Error())
			}

			if err := g.postAuditProposalEvent(p.Id); err != nil {
				return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
			}

			g.Logger().WithFields(logrus.Fields{
				"objId":      objId,
				"proposalID": p.Id,
			}).Info("reject proposal")
		}
	}

	return boltvm.Success(nil)
}

//// =========== EndCurrentProposal forces the current proposal to be ended and does not process other proposals associated with the proposal
//func (g *Governance) EndCurrentProposal(id, endReason string, extra []byte) *boltvm.Response {
//	// 1. check permission
//	specificAddrs := []string{
//		//constant.AppchainMgrContractAddr.Address().String(),
//		constant.RuleManagerContractAddr.Address().String(),
//		//constant.NodeManagerContractAddr.Address().String(),
//		//constant.RoleContractAddr.Address().String(),
//		//constant.DappMgrContractAddr.Address().String(),
//		constant.ServiceMgrContractAddr.Address().String(),
//	}
//	addrsData, err := json.Marshal(specificAddrs)
//	if err != nil {
//		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
//	}
//	if err := g.checkPermission([]string{string(PermissionSpecific)}, "", g.CurrentCaller(), addrsData); err != nil {
//		return boltvm.Error(boltvm.GovernanceNoPermissionCode, fmt.Sprintf(string(boltvm.GovernanceNoPermissionMsg), g.CurrentCaller(), err.Error()))
//	}
//
//	// 2. end proposal
//	if _, berr := g.endProposal(id, endReason, extra); berr != nil {
//		return boltvm.Error(berr.Code, berr.Error())
//	}
//
//	if err := g.postAuditProposalEvent(id); err != nil {
//		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
//	}
//
//	return boltvm.Success(nil)
//}

// =========== Vote add someone's voting information (each person can only vote once)
func (g *Governance) Vote(id, approve string, reason string) *boltvm.Response {
	// 0. check role
	addr := g.Caller()
	res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(addr), pb.String(string(GovernanceAdmin)))
	if !res.Ok {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("cross invoke IsAvailable error: %s", string(res.Result))))
	}
	if string(res.Result) != "true" {
		return boltvm.Error(boltvm.GovernanceUnavailableAdminVoteCode, fmt.Sprintf(string(boltvm.GovernanceUnavailableAdminVoteMsg), addr))
	}

	// 1. Determine if the proposal exists
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	// 2. Set vote
	if res := g.setVote(p, addr, approve, reason); !res.Ok {
		return res
	}

	// 3. Count votes
	// If the threshold for participation is reached, the result of the vote can be judged.
	// If the policy determines that the current vote has closed, the proposal state is modified.
	ok, err := g.countVote(p)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("count vote error: %v", err)))
	}
	if !ok {
		// the round of the voting is not over, wait the next vote
		g.Logger().WithFields(logrus.Fields{
			"id": p.Id,
		}).Info("wait next vote")
		if err := g.postAuditProposalEvent(id); err != nil {
			return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
		}
		return boltvm.Success(nil)
	}

	// 4. Handle result
	if err = g.handleResult(p); err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	g.Logger().WithFields(logrus.Fields{
		"id": p.Id,
	}).Info("vote end")
	if err := g.postAuditProposalEvent(id); err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
	}
	return boltvm.Success(nil)
}

// Set vote of an administrator
func (g *Governance) setVote(p *Proposal, addr string, approve string, reason string) *boltvm.Response {
	// 1. Determine if the proposal has been approved or rejected
	if p.Status != PROPOSED {
		return boltvm.Error(boltvm.GovernanceVoteEndProposalCode, fmt.Sprintf(string(boltvm.GovernanceVoteEndProposalMsg), p.Status))
	}

	// 2. Determine if the administrator can vote
	for _, e := range p.ElectorateList {
		if addr == e.ID {

			// 3. Determine if the administrator has voted
			if _, ok := p.BallotMap[addr]; ok {
				return boltvm.Error(boltvm.GovernanceAdminRepeatVoteCode, fmt.Sprintf(string(boltvm.GovernanceAdminRepeatVoteMsg), addr))
			}

			// 4. Record Voting Information
			ballot := Ballot{
				VoterAddr: addr,
				Approve:   approve,
				Num:       e.Weight,
				Reason:    reason,
				VoteTime:  g.GetTxTimeStamp(),
			}
			if repo.SuperAdminWeight == e.Weight {
				p.IsSuperAdminVoted = true
			}
			p.BallotMap[addr] = ballot
			switch approve {
			case BallotApprove:
				p.ApproveNum++
			case BallotReject:
				p.AgainstNum++
			default:
				return boltvm.Error(boltvm.GovernanceIllegalVoteInfoCode, fmt.Sprintf(string(boltvm.GovernanceIllegalVoteInfoMsg), approve))
			}

			thresholdApproveNum, err := getCurThresholdApproveNum(p.StrategyExpression, p.ApproveNum, p.AgainstNum, p.InitialElectorateNum, p.AvaliableElectorateNum)
			if err != nil {
				return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
			}
			p.ThresholdApproveNum = uint64(thresholdApproveNum)

			g.SetObject(ProposalKey(p.Id), *p)
			return boltvm.Success(nil)
		}
	}

	return boltvm.Error(boltvm.GovernanceAdminNoVotePermissonCode, fmt.Sprintf(string(boltvm.GovernanceAdminNoVotePermissonMsg), addr, p.Id))
}

// Count votes to see if this round is over.
// If the vote is over change the status of the proposal.
func (g *Governance) countVote(p *Proposal) (bool, error) {
	// 1. Special types of proposals require super administrator voting
	if p.IsSpecial {
		if !p.IsSuperAdminVoted {
			return false, nil
		}
	}

	// 2. Determine whether the vote is end or not
	isEnd, isApprove, err := repo.MakeStrategyDecision(p.StrategyExpression, p.ApproveNum, p.AgainstNum, p.InitialElectorateNum, p.AvaliableElectorateNum)
	if err != nil {
		return false, err
	}

	if isEnd {
		p.EndReason = NormalReason
		if isApprove {
			g.changeProposalStatus(p, APPROVED)
		} else {
			g.changeProposalStatus(p, REJECTED)
		}
		return true, nil
	} else {
		// wait next vote
		return false, nil
	}
}

func (g *Governance) handleResult(p *Proposal) (err error) {
	// unlock low-priority proposal
	nextEventType := governance.EventType(p.Status)
	if p.LockProposalId != "" {
		if p.Status == APPROVED {
			notRestoreReason := fmt.Sprintf("%s(%s)", string(PriorityReason), p.Id)
			_, err = g.unlockLowPriorityProposal(p.LockProposalId, false, notRestoreReason)
		} else {
			nextEventType, err = g.unlockLowPriorityProposal(p.LockProposalId, true, "")
		}
		if err != nil {
			return fmt.Errorf("unlock low priority proposals error: %v", err)
		}
	}

	// manage object
	return g.manageObj(p.Typ, p.EventType, nextEventType, p.ObjLastStatus, p.ObjId, p.Extra)
}

func (g *Governance) manageObj(proposalTyp ProposalType, eventType, nextEventType governance.EventType, objLastStatus governance.GovernanceStatus, objId string, extra []byte) error {
	switch proposalTyp {
	case RoleMgr:
		res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "Manage", pb.String(string(eventType)), pb.String(string(nextEventType)), pb.String(string(objLastStatus)), pb.String(objId), pb.Bytes(extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	case ServiceMgr:
		res := g.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "Manage", pb.String(string(eventType)), pb.String(string(nextEventType)), pb.String(string(objLastStatus)), pb.String(objId), pb.Bytes(extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	case NodeMgr:
		res := g.CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "Manage", pb.String(string(eventType)), pb.String(string(nextEventType)), pb.String(string(objLastStatus)), pb.String(objId), pb.Bytes(extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	case RuleMgr:
		res := g.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage", pb.String(string(eventType)), pb.String(string(nextEventType)), pb.String(string(objLastStatus)), pb.String(objId), pb.Bytes(extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	case DappMgr:
		res := g.CrossInvoke(constant.DappMgrContractAddr.Address().String(), "Manage", pb.String(string(eventType)), pb.String(string(nextEventType)), pb.String(string(objLastStatus)), pb.String(objId), pb.Bytes(extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		return nil
	case ProposalStrategyMgr:
		res := g.CrossInvoke(constant.ProposalStrategyMgrContractAddr.Address().String(), "Manage", pb.String(string(eventType)), pb.String(string(nextEventType)), pb.String(string(objLastStatus)), pb.String(objId), pb.Bytes(extra))
		if !res.Ok {
			return fmt.Errorf("invoke Manager error: %s", string(res.Result))
		}
		return nil
	default: // APPCHAIN_MGR
		res := g.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "Manage", pb.String(string(eventType)), pb.String(string(nextEventType)), pb.String(string(objLastStatus)), pb.String(objId), pb.Bytes(extra))
		if !res.Ok {
			return fmt.Errorf("cross invoke Manager error: %s", string(res.Result))
		}
		if eventType == governance.EventRegister {
			g.Logger().WithFields(logrus.Fields{
				"chainID": string(res.Result),
			}).Info("Appchain registering ok")
		}
		return nil
	}
}

// =========== LockLowPriorityProposal locks a proposed proposal for an object
func (g *Governance) LockLowPriorityProposal(objId, eventTyp string) *boltvm.Response {
	// 1. check permission
	specificAddrs := []string{
		constant.ServiceMgrContractAddr.Address().String(),
		constant.RoleContractAddr.Address().String(),
		constant.RuleManagerContractAddr.Address().String(),
	}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := g.checkPermission([]string{string(PermissionSpecific)}, "", g.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.GovernanceNoPermissionCode, fmt.Sprintf(string(boltvm.GovernanceNoPermissionMsg), g.CurrentCaller(), err.Error()))
	}

	lockedProId, err := g.lockLowPriorityProposal(objId, eventTyp)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("lock low priority proposal error: %v", err)))
	}

	return boltvm.Success([]byte(lockedProId))
}

func (g *Governance) lockLowPriorityProposal(objId, eventTyp string) (string, error) {
	proposals, err := g.getProposalsByObjId(objId)
	if err != nil {
		return "", err
	}

	for _, p := range proposals {
		if p.Status == PROPOSED {
			if priority[p.EventType] < priority[governance.EventType(eventTyp)] {
				g.changeProposalStatus(p, PAUSED)
				g.Logger().WithFields(logrus.Fields{
					"proposal": p.Id,
					"type":     p.Typ,
					"status":   p.Status,
					"objID":    p.ObjId,
					"eventTyp": p.EventType,
				}).Info("lock low priority proposal")

				if err := g.postAuditProposalEvent(p.Id); err != nil {
					return "", fmt.Errorf("post audit proposal event error: %v", err)
				}
				return p.Id, nil
			} else {
				g.Logger().WithFields(logrus.Fields{
					"proposal": p.Id,
					"type":     p.Typ,
					"status":   p.Status,
					"objID":    p.ObjId,
					"eventTyp": p.EventType,
				}).Info("there is an equal or higher proposal which is in progress currently")
				//return "", fmt.Errorf("the obj(%s) has an equal or higher priority proposal(%s,%s) is in progress currently, please submit it later", objId, p.EventType, p.Id)
			}
		}
	}

	return "", nil
}

// =========== UnLockLowPriorityProposal unlocks a suspended proposal for an object
func (g *Governance) UnLockLowPriorityProposal(objId, eventTyp string) *boltvm.Response {
	// 1. check permission
	specificAddrs := []string{
		constant.ServiceMgrContractAddr.Address().String(),
		constant.RoleContractAddr.Address().String(),
		constant.RuleManagerContractAddr.Address().String(),
	}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := g.checkPermission([]string{string(PermissionSpecific)}, "", g.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.GovernanceNoPermissionCode, fmt.Sprintf(string(boltvm.GovernanceNoPermissionMsg), g.CurrentCaller(), err.Error()))
	}

	// 2. unlock low proiority proposak
	lockedProposal, err := g.getHightestPriorityPausedProposalByObjId(objId)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("getHightestPriorityPausedProposalByObjId err: %v", err)))
	}
	if lockedProposal == nil {
		return boltvm.Success(nil)
	}

	nextEventType, err := g.unlockLowPriorityProposal(lockedProposal.Id, true, "")
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("unlock low priority proposal error: %v", err)))
	}

	// manage object
	// - The fourth parameter takes effect when nextEvent is reject. This parameter is not required here.
	if err := g.manageObj(lockedProposal.Typ, governance.EventType(eventTyp), nextEventType, "", lockedProposal.ObjId, nil); err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("manage object error: %v", err)))
	}

	if err := g.postAuditProposalEvent(lockedProposal.Id); err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
	}
	return boltvm.Success(nil)
}

// Only the state is unlocked, and function manageObj needs to be called later
func (g *Governance) unlockLowPriorityProposal(lockedProposalId string, restore bool, notRestoreReason string) (governance.EventType, error) {
	var nextEventType governance.EventType
	lockP := &Proposal{}
	if !g.GetObject(ProposalKey(lockedProposalId), lockP) {
		return "", fmt.Errorf("proposal does not exist")
	}

	if !restore {
		lockP.EndReason = EndReason(notRestoreReason)
		g.changeProposalStatus(lockP, REJECTED)
	} else {
		nextEventType = lockP.EventType
		g.changeProposalStatus(lockP, PROPOSED)
	}
	g.Logger().WithFields(logrus.Fields{
		"proposal": lockedProposalId,
		"type":     lockP.Typ,
		"status":   lockP.Status,
		"restore":  restore,
	}).Info("unlock low priority proposal")
	return nextEventType, nil
}

// Queries suspended proposals for an administrated object and returns only the one with the highest priority
func (g *Governance) getHightestPriorityPausedProposalByObjId(objID string) (*Proposal, error) {
	pauseProposals, err := g.getProposalsByObjStatus(objID, PAUSED)
	if err != nil {
		return nil, err
	}

	var retP *Proposal
	if len(pauseProposals) != 0 {
		retP = pauseProposals[0]
		for i := 1; i < len(pauseProposals); i++ {
			if priority[retP.EventType] < priority[pauseProposals[i].EventType] {
				retP = pauseProposals[i]
			}
		}
	}

	return retP, nil
}

func (g *Governance) getProposalsByObjStatus(objID string, status ProposalStatus) ([]*Proposal, error) {
	ret := make([]*Proposal, 0)

	proposals, err := g.getProposalsByObjId(objID)
	if err != nil {
		return nil, err
	}
	for _, p := range proposals {
		if status == p.Status {
			ret = append(ret, p)
		}
	}

	return ret, nil
}

// =========== UpdateAvaliableElectorateNum updates AvaliableElectorateNum for a proposal, because of changes in the number of administrators available
func (g *Governance) UpdateAvaliableElectorateNum(id string, num uint64) *boltvm.Response {
	// 1. check permission
	specificAddrs := []string{
		constant.RoleContractAddr.Address().String(),
	}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	if err := g.checkPermission([]string{string(PermissionSpecific)}, "", g.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.GovernanceNoPermissionCode, fmt.Sprintf(string(boltvm.GovernanceNoPermissionMsg), g.CurrentCaller(), err.Error()))
	}

	// 2. update num
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	p.AvaliableElectorateNum = num
	if p.StrategyType != ZeroPermission {
		thresholdApproveNum, err := getCurThresholdApproveNum(p.StrategyExpression, p.ApproveNum, p.AgainstNum, p.InitialElectorateNum, p.AvaliableElectorateNum)
		if err != nil {
			return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("getCurThresholdApproveNum err: %s", err)))
		}
		p.ThresholdApproveNum = uint64(thresholdApproveNum)

		g.Logger().WithFields(logrus.Fields{
			"proposalId":             id,
			"AvaliableElectorateNum": p.AvaliableElectorateNum,
		}).Info("Update avaliable electorate num")
		isEnd, isApprove, err := repo.MakeStrategyDecision(p.StrategyExpression, p.ApproveNum, p.AgainstNum, p.InitialElectorateNum, p.AvaliableElectorateNum)
		if err != nil {
			return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("MakeStrategyDecision err: %v", err)))
		}

		if isEnd {
			p.EndReason = ElectorateReason
			if isApprove {
				g.changeProposalStatus(p, APPROVED)
			} else {
				g.changeProposalStatus(p, REJECTED)
			}
			err := g.handleResult(p)
			if err != nil {
				return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
			}
		} else {
			g.SetObject(ProposalKey(p.Id), *p)
		}

		if err := g.postAuditProposalEvent(id); err != nil {
			return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), fmt.Sprintf("post audit proposal event error: %v", err)))
		}
	}
	return boltvm.Success([]byte(strconv.Itoa(int(p.AvaliableElectorateNum))))
}

// ========================== Query interface ========================
func (g *Governance) GetBallot(voterAddr, proposalId string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(proposalId), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), proposalId, ""))
	}

	ballot, ok := p.BallotMap[voterAddr]
	if !ok {
		return boltvm.Error(boltvm.GovernanceNotVoteAdminCode, fmt.Sprintf(string(boltvm.GovernanceNotVoteAdminMsg), voterAddr))
	}

	bData, err := json.Marshal(ballot)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(bData)
}

// GetProposal query proposal by id
func (g *Governance) GetProposal(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	pData, err := json.Marshal(p)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(pData)
}

// Query proposals by the ID of the managed chain, returning a list of proposal for that type
func (g *Governance) GetProposalsByObjId(objId string) *boltvm.Response {
	ret, err := g.getProposalsByObjId(objId)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

// Query proposals by the ID of the managed chain in create time order, returning a list of proposal for that type
func (g *Governance) GetProposalsByObjIdInCreateTimeOrder(objId string) *boltvm.Response {
	ret, err := g.getProposalsByObjId(objId)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	sort.Sort(Proposals(ret))

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

func (g *Governance) getProposalsByObjId(objId string) ([]*Proposal, error) {
	ret := make([]*Proposal, 0)

	idMap := orderedmap.New()
	if ok := g.GetObject(ProposalObjKey(objId), idMap); !ok {
		return ret, nil
	}

	for _, id := range idMap.Keys() {
		p := &Proposal{}
		if ok := g.GetObject(ProposalKey(id), p); !ok {
			return nil, fmt.Errorf("the obj has proposal %s but not exist", id)
		}
		ret = append(ret, p)
	}

	return ret, nil
}

// Query proposals by proposal type, returning a list of proposal for that type
func (g *Governance) GetProposalsByFrom(from string) *boltvm.Response {
	ret, err := g.getProposalsByFrom(from)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

func (g *Governance) getProposalsByFrom(from string) ([]*Proposal, error) {
	ret := make([]*Proposal, 0)

	idMap := orderedmap.New()
	if ok := g.GetObject(ProposalFromKey(from), idMap); !ok {
		return ret, nil
	}

	for _, id := range idMap.Keys() {
		p := &Proposal{}
		if ok := g.GetObject(ProposalKey(id), p); !ok {
			return nil, fmt.Errorf("proposal %s is not exist", id)
		}
		ret = append(ret, p)
	}

	return ret, nil
}

// Query proposals by proposal type, returning a list of proposal for that type
func (g *Governance) GetProposalsByTyp(typ string) *boltvm.Response {
	if err := repo.CheckManageModule(typ); err != nil {
		return boltvm.Error(boltvm.GovernanceIllegalProposalTypeCode, fmt.Sprintf(string(boltvm.GovernanceIllegalProposalTypeMsg), typ))
	}

	ret, err := g.getProposalsByType(typ)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

func (g *Governance) getProposalsByType(typ string) ([]*Proposal, error) {
	ret := make([]*Proposal, 0)

	idMap := orderedmap.New()
	if ok := g.GetObject(ProposalTypKey(typ), idMap); !ok {
		return ret, nil
	}

	for _, id := range idMap.Keys() {
		p := &Proposal{}
		if ok := g.GetObject(ProposalKey(id), p); !ok {
			return nil, fmt.Errorf("proposal %s is not exist", id)
		}
		ret = append(ret, p)
	}

	return ret, nil
}

// Query proposals based on proposal status, returning a list of proposal for that status
func (g *Governance) GetProposalsByStatus(status string) *boltvm.Response {
	if err := checkProposalStatus(ProposalStatus(status)); err != nil {
		return boltvm.Error(boltvm.GovernanceIllegalProposalStatusCode, fmt.Sprintf(string(boltvm.GovernanceIllegalProposalStatusMsg), status))
	}

	ret, err := g.getProposalsByStatus(status)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

func (g *Governance) getProposalsByStatus(status string) ([]*Proposal, error) {
	ret := make([]*Proposal, 0)

	idMap := orderedmap.New()
	if ok := g.GetObject(ProposalStatusKey(status), idMap); !ok {
		return ret, nil
	}

	for _, id := range idMap.Keys() {
		p := &Proposal{}
		if ok := g.GetObject(ProposalKey(id), p); !ok {
			return nil, fmt.Errorf("proposal %s is not exist", id)
		}
		ret = append(ret, p)
	}

	return ret, nil
}

// get proposal which is not closed (status is proposed or paused)
func (g *Governance) GetNotClosedProposals() *boltvm.Response {
	ret, err := g.getProposalsByStatus(string(PROPOSED))
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	for _, p := range ret {
		g.Logger().WithFields(logrus.Fields{
			"proposalId":     p.Id,
			"proposalStatus": p.Status,
		}).Debug("get proposed proposal")
	}

	ret2, err := g.getProposalsByStatus(string(PAUSED))
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	for _, p := range ret2 {
		g.Logger().WithFields(logrus.Fields{
			"proposalId":     p.Id,
			"proposalStatus": p.Status,
		}).Debug("get paused proposal")
	}

	ret = append(ret, ret2...)

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

// Get affirmative vote information
func (g *Governance) GetApprove(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	approveMap := map[string]Ballot{}
	for k, v := range p.BallotMap {
		if v.Approve == BallotApprove {
			approveMap[k] = v
		}
	}

	retData, err := json.Marshal(approveMap)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

// Get negative vote information
func (g *Governance) GetAgainst(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	againstMap := map[string]Ballot{}
	for k, v := range p.BallotMap {
		if v.Approve == BallotReject {
			againstMap[k] = v
		}
	}

	retData, err := json.Marshal(againstMap)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(retData)
}

// Get the total number of affirmative votes
func (g *Governance) GetApproveNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.ApproveNum))))
}

// Get the total number of negative votes
func (g *Governance) GetAgainstNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.AgainstNum))))
}

// Get the number of total votes, include all votes cast and all votes not cast
func (g *Governance) GetPrimaryElectorateNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.InitialElectorateNum))))
}

// Get the number of total votes, include all votes cast and all votes not cast
func (g *Governance) GetAvaliableElectorateNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	return boltvm.Success([]byte(strconv.Itoa(int(p.AvaliableElectorateNum))))
}

// Get the number of people who have voted
func (g *Governance) GetVotedNum(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	return boltvm.Success([]byte(strconv.Itoa(len(p.BallotMap))))
}

// Get voted information
func (g *Governance) GetVoted(id string) *boltvm.Response {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, ""))
	}

	retData, err := json.Marshal(p.BallotMap)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	return boltvm.Success(retData)
}

// Get Unvoted information
func (g *Governance) GetUnvote(id string) *boltvm.Response {
	ret, err := g.getUnvote(id)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, err.Error()))
	}

	retData, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}

	return boltvm.Success(retData)
}

func (g *Governance) getUnvote(id string) ([]*repo.Admin, error) {
	p := &Proposal{}
	if !g.GetObject(ProposalKey(id), p) {
		return nil, fmt.Errorf("proposal does not exist")
	}

	res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAdminRoles")
	if !res.Ok {
		return nil, fmt.Errorf(fmt.Sprintf("get admin roles error: %s", string(res.Result)))
	}
	var admins []*repo.Admin
	if err := json.Unmarshal(res.Result, &admins); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("get admin roles error: %v", err))
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
		return boltvm.Error(boltvm.GovernanceNonexistentProposalCode, fmt.Sprintf(string(boltvm.GovernanceNonexistentProposalMsg), id, err.Error()))
	}

	return boltvm.Success([]byte(strconv.Itoa(len(ret))))
}

// Key ====================================================================
func ProposalKey(id string) string {
	return fmt.Sprintf("%s-%s", PROPOSAL_PREFIX, id)
}

func ProposalFromKey(id string) string {
	return fmt.Sprintf("%s-%s", PROPOSALFREOM_PREFIX, id)
}

func ProposalObjKey(id string) string {
	return fmt.Sprintf("%s-%s", PROPOSALOBJ_PREFIX, id)
}

func ProposalTypKey(typ string) string {
	return fmt.Sprintf("%s-%s", PROPOSALTYPE_PREFIX, typ)
}

func ProposalStatusKey(status string) string {
	return fmt.Sprintf("%s-%s", PROPOSALSTATUS_PREFIX, status)
}

func getGovernanceRet(proposalID string, extra []byte) *boltvm.Response {
	res1 := governance.GovernanceResult{
		ProposalID: proposalID,
		Extra:      extra,
	}
	resData, err := json.Marshal(res1)
	if err != nil {
		return boltvm.Error(boltvm.GovernanceInternalErrCode, fmt.Sprintf(string(boltvm.GovernanceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(resData)
}

// Others =============================================================
type Proposals []*Proposal

func (ps Proposals) Len() int { return len(ps) }

func (ps Proposals) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }

func (ps Proposals) Less(i, j int) bool {
	return ps[i].CreateTime > ps[j].CreateTime
}

// post event
func (g *Governance) postAuditProposalEvent(proposalID string) error {
	ok, proposalData := g.Get(ProposalKey(proposalID))
	if !ok {
		return fmt.Errorf("not found proposal %s", proposalID)
	}
	auditInfo := &pb.AuditRelatedObjInfo{
		AuditObj:           proposalData,
		RelatedChainIDList: map[string][]byte{},
		RelatedNodeIDList:  map[string][]byte{},
	}

	proposal := &Proposal{}
	if err := json.Unmarshal(proposalData, proposal); err != nil {
		return err
	}

	switch proposal.Typ {
	case AppchainMgr:
		auditInfo.RelatedChainIDList[proposal.ObjId] = []byte{}
	case RuleMgr:
		appchainID := strings.Split(proposal.ObjId, ":")[0]
		auditInfo.RelatedChainIDList[appchainID] = []byte{}
	case ServiceMgr:
		appchainID := strings.Split(proposal.ObjId, ":")[0]
		auditInfo.RelatedChainIDList[appchainID] = []byte{}
	case NodeMgr:
		auditInfo.RelatedNodeIDList[proposal.ObjId] = []byte{}
	}

	g.PostEvent(pb.Event_AUDIT_PROPOSAL, auditInfo)

	return nil
}

// Check info =============================================================
func checkProposalStatus(ps ProposalStatus) error {
	if ps != PROPOSED &&
		ps != APPROVED &&
		ps != REJECTED &&
		ps != PAUSED {
		return fmt.Errorf("illegal proposal status")
	}
	return nil
}
