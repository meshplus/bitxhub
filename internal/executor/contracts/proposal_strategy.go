package contracts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/looplab/fsm"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

type GovStrategy struct {
	boltvm.Stub
}

// Proposal strategy ===============================================================
type ProposalStrategyType string

const (
	SuperMajorityApprove ProposalStrategyType = repo.SuperMajorityApprove
	SuperMajorityAgainst ProposalStrategyType = repo.SuperMajorityAgainst
	SimpleMajority       ProposalStrategyType = repo.SimpleMajority
	ZeroPermission       ProposalStrategyType = repo.ZeroPermission
)

type ProposalStrategy struct {
	Module string                      `json:"module"`
	Typ    ProposalStrategyType        `json:"typ"`
	Extra  string                      `json:"extra"`
	Status governance.GovernanceStatus `json:"status"`
	FSM    *fsm.FSM                    `json:"fsm"`
}

type UpdateStrategyInfo struct {
	Typ   UpdateInfo `json:"typ"`
	Extra UpdateInfo `json:"extra"`
}

var strategyStateMap = map[governance.EventType][]governance.GovernanceStatus{
	governance.EventUpdate: {governance.GovernanceAvailable},
}

func (d *ProposalStrategy) setFSM(lastStatus governance.GovernanceStatus) {
	d.FSM = fsm.NewFSM(
		string(d.Status),
		fsm.Events{
			// register 3
			{Name: string(governance.EventRegister), Src: []string{string(governance.GovernanceUnavailable)}, Dst: string(governance.GovernanceRegisting)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceRegisting)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceRegisting)}, Dst: string(lastStatus)},

			// update 1
			{Name: string(governance.EventUpdate), Src: []string{string(governance.GovernanceAvailable)}, Dst: string(governance.GovernanceUpdating)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceAvailable)},
		},
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) { d.Status = governance.GovernanceStatus(d.FSM.Current()) },
		},
	)
}

func (g *GovStrategy) changeStatus(pt string, trigger, lastStatus string) (bool, []byte) {
	ps := &ProposalStrategy{}
	if ok := g.GetObject(ProposalStrategyKey(pt), ps); !ok {
		return false, []byte("this proposal strategy does not exist")
	}

	ps.setFSM(governance.GovernanceStatus(lastStatus))
	err := ps.FSM.Event(trigger)
	if err != nil {
		return false, []byte(fmt.Sprintf("change status error: %v", err))
	}

	g.SetObject(ProposalStrategyKey(pt), *ps)
	return true, nil
}

func (g *GovStrategy) checkPermission(permissions []string, regulatedAddr, regulatorAddr string, specificAddrsData []byte) error {
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

var mgrs = []string{repo.AppchainMgr, repo.NodeMgr, repo.DappMgr, repo.RoleMgr, repo.RuleMgr, repo.ServiceMgr, repo.ProposalStrategyMgr}

// =========== Manage does some subsequent operations when the proposal is over
func (g *GovStrategy) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	g.Logger().WithFields(logrus.Fields{
		"id": objId,
	}).Info("proposal strategy is managing")

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := g.checkPermission([]string{string(PermissionSpecific)}, objId, g.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyNoPermissionCode, fmt.Sprintf(string(boltvm.ProposalStrategyNoPermissionMsg), g.CurrentCaller(), err.Error()))
	}

	// 2. change status
	if objId != repo.AllMgr {
		ok, errData := g.changeStatus(objId, proposalResult, lastStatus)
		if !ok {
			return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("change %s status error: %s", objId, string(errData))))
		}
	} else {
		for _, mgr := range mgrs {
			ok, errData := g.changeStatus(mgr, proposalResult, lastStatus)
			if !ok {
				return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("change %s status error: %s", objId, string(errData))))
			}
		}
	}

	if proposalResult == string(APPROVED) {
		if eventTyp == string(governance.EventUpdate) {
			updateStrategyInfo := map[string]UpdateStrategyInfo{}
			if err := json.Unmarshal(extra, &updateStrategyInfo); err != nil {
				return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("unmarshal update strategy")))
			}
			res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin)))
			if !res.Ok {
				return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("cross invoke GetRolesByType error: %s", string(res.Result))))
			}
			roles := []*Role{}
			if err := json.Unmarshal(res.Result, &roles); err != nil {
				return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("json unmarshal error: %s", err.Error())))
			}
			availableNum := 0
			for _, r := range roles {
				if r.IsAvailable() {
					availableNum++
				}
			}
			for mgr, mInfo := range updateStrategyInfo {
				ps := &ProposalStrategy{}
				if ok := g.GetObject(ProposalStrategyKey(mgr), ps); !ok {
					return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("get proposal strategy %s error", objId)))
				}
				if mInfo.Typ.IsEdit {
					ps.Typ = ProposalStrategyType(mInfo.Typ.NewInfo.(string))
				}
				if mInfo.Extra.IsEdit {
					ps.Extra = mInfo.Extra.NewInfo.(string)
				}
				if ps.Typ == ZeroPermission {
					g.SetObject(ProposalStrategyKey(mgr), ps)
				} else if _, err := getCurThresholdApproveNum(ps.Extra, 0, 0, uint64(availableNum), uint64(availableNum)); err != nil {
					g.Logger().WithFields(logrus.Fields{
						"module": mgr,
						"old":    ps,
						"new":    defaultStrategy(mgr),
					}).Info("update module strategy because of roles change")
					g.SetObject(ProposalStrategyKey(mgr), defaultStrategy(mgr))
				} else {
					g.SetObject(ProposalStrategyKey(mgr), ps)
				}
			}
		}
	}
	return boltvm.Success(nil)
}

func (g *GovStrategy) governancePre(module, event string) (*ProposalStrategy, *boltvm.BxhError) {
	ps := &ProposalStrategy{}
	if ok := g.GetObject(ProposalStrategyKey(module), ps); !ok {
		// If the strategy does not exist, the current module has used the default SimpleMajority strategy until now.
		ps = defaultStrategy(module)
	}

	for _, s := range strategyStateMap[governance.EventType(event)] {
		if ps.Status == s {
			return ps, nil
		}
	}

	return nil, boltvm.BError(boltvm.ProposalStrategyStatusErrorCode, fmt.Sprintf(string(boltvm.ProposalStrategyStatusErrorMsg), module, ps.Status, event))
}

// update proposal strategy for a proposal type
func (g *GovStrategy) UpdateProposalStrategy(module string, typ string, strategyExtra string, reason string) *boltvm.Response {
	// 1. check premission
	if err := g.checkPermission([]string{string(PermissionAdmin)}, module, g.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyNoPermissionCode, fmt.Sprintf(string(boltvm.ProposalStrategyNoPermissionMsg), g.CurrentCaller(), fmt.Sprintf("check permission error:%v", err)))
	}

	// 2. check strategy status (check whether the strategy is being updated)
	strategy, bxhErr := g.governancePre(module, string(governance.EventUpdate))
	if bxhErr != nil {
		return boltvm.Error(bxhErr.Code, string(bxhErr.Msg))
	}

	info := UpdateStrategyInfo{}
	info.Typ = UpdateInfo{
		OldInfo: strategy.Typ,
		NewInfo: typ,
		IsEdit:  string(strategy.Typ) != typ,
	}
	info.Extra = UpdateInfo{
		OldInfo: strings.Replace(strategy.Extra, " ", "", -1),
		NewInfo: strings.Replace(strategyExtra, " ", "", -1),
		IsEdit:  strings.Replace(strategy.Extra, " ", "", -1) != strings.Replace(strategyExtra, " ", "", -1),
	}

	// 3. check whether the updated information is consistent with the previous information
	if !info.Typ.IsEdit && !info.Extra.IsEdit {
		return boltvm.Error(boltvm.ProposalStrategyNotUpdateCode, string(boltvm.ProposalStrategyNotUpdateMsg))
	}

	// 4. check strategy info
	availableAdminNum := 0
	res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin)))
	if !res.Ok {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("cross invoke GetRolesByType error: %s", string(res.Result))))
	}
	roles := make([]*Role, 0)
	if err := json.Unmarshal(res.Result, &roles); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("unmarshal error: %v", err)))
	}
	for _, role := range roles {
		if role.IsAvailable() {
			availableAdminNum++
		}
	}
	if err := repo.CheckStrategyInfo(typ, module, strategyExtra, availableAdminNum); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyIllegalProposalStrategyInfoCode, fmt.Sprintf(string(boltvm.ProposalStrategyIllegalProposalStrategyInfoMsg), err.Error()))
	}

	// 5. submit proposal
	infoMap := map[string]UpdateStrategyInfo{}
	infoMap[module] = info
	extra, err := json.Marshal(infoMap)
	if err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf("unmarshal update strategy error: %v", err))
	}
	res = g.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(g.Caller()),
		pb.String(string(governance.EventUpdate)),
		pb.String(string(ProposalStrategyMgr)),
		pb.String(module),
		pb.String(string(strategy.Status)),
		pb.String(reason),
		pb.Bytes(extra),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	g.changeStatus(module, string(governance.EventUpdate), string(strategy.Status))

	g.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	return getGovernanceRet(string(res.Result), []byte(module))
}

// update proposal strategy for a proposal type
func (g *GovStrategy) UpdateAllProposalStrategy(typ string, strategyExtra string, reason string) *boltvm.Response {
	pt := repo.AllMgr

	// 1. check premission
	if err := g.checkPermission([]string{string(PermissionAdmin)}, pt, g.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyNoPermissionCode, fmt.Sprintf(string(boltvm.ProposalStrategyNoPermissionMsg), g.CurrentCaller(), fmt.Sprintf("check permission error:%v", err)))
	}

	// 2. check strategy status (check whether the strategy is being updated)
	infoMap := map[string]UpdateStrategyInfo{}
	for _, module := range mgrs {
		moduleStrategy, bxhErr := g.governancePre(module, string(governance.EventUpdate))
		if bxhErr != nil {
			return boltvm.Error(bxhErr.Code, string(bxhErr.Msg))
		}
		infoMap[module] = UpdateStrategyInfo{
			Typ: UpdateInfo{
				OldInfo: moduleStrategy.Typ,
				NewInfo: typ,
				IsEdit:  string(moduleStrategy.Typ) != typ,
			},
			Extra: UpdateInfo{
				OldInfo: strings.Replace(moduleStrategy.Extra, " ", "", -1),
				NewInfo: strings.Replace(strategyExtra, " ", "", -1),
				IsEdit:  strings.Replace(moduleStrategy.Extra, " ", "", -1) != strings.Replace(strategyExtra, " ", "", -1),
			},
		}
	}

	// 3. check whether the updated information is consistent with the previous information
	isEdit := false
	for _, info := range infoMap {
		if info.Typ.IsEdit || info.Extra.IsEdit {
			isEdit = true
			break
		}
	}
	if !isEdit {
		return boltvm.Error(boltvm.ProposalStrategyNotUpdateCode, string(boltvm.ProposalStrategyNotUpdateMsg))
	}

	// 4. check strategy info
	availableAdminNum := 0
	res := g.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRolesByType", pb.String(string(GovernanceAdmin)))
	if !res.Ok {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("cross invoke GetRolesByType error: %s", string(res.Result))))
	}
	roles := make([]*Role, 0)
	if err := json.Unmarshal(res.Result, &roles); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("unmarshal error: %v", err)))
	}
	for _, role := range roles {
		if role.IsAvailable() {
			availableAdminNum++
		}
	}
	if err := repo.CheckStrategyType(typ, strategyExtra, availableAdminNum); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyIllegalProposalStrategyInfoCode, fmt.Sprintf(string(boltvm.ProposalStrategyIllegalProposalStrategyInfoMsg), err.Error()))
	}

	// 5. submit proposal
	extra, err := json.Marshal(infoMap)
	if err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf("unmarshal update strategy error: %v", err))
	}
	res = g.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(g.Caller()),
		pb.String(string(governance.EventUpdate)),
		pb.String(string(ProposalStrategyMgr)),
		pb.String(pt),
		pb.String(string(governance.GovernanceAvailable)),
		pb.String(reason),
		pb.Bytes(extra),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	for _, mgr := range mgrs {
		g.changeStatus(mgr, string(governance.EventUpdate), string(governance.GovernanceAvailable))
	}

	g.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	return getGovernanceRet(string(res.Result), []byte(pt))
}

// update proposal strategy for a proposal type
func (g *GovStrategy) UpdateProposalStrategyByRolesChange(availableNum uint64) *boltvm.Response {
	// 1. check premission
	specificAddrs := []string{constant.RoleContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}

	if err := g.checkPermission([]string{string(PermissionSpecific)}, repo.AllMgr, g.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyNoPermissionCode, fmt.Sprintf(string(boltvm.ProposalStrategyNoPermissionMsg), g.CurrentCaller(), fmt.Sprintf("check permission error:%v", err)))
	}

	// 2. update strategy
	for _, module := range mgrs {
		strategy := defaultStrategy(module)
		ok := g.GetObject(ProposalStrategyKey(module), strategy)
		if !ok || strategy.Status == governance.GovernanceUpdating || strategy.Typ == ZeroPermission {
			continue
		}
		if _, err := getCurThresholdApproveNum(strategy.Extra, 0, 0, availableNum, availableNum); err != nil {
			g.Logger().WithFields(logrus.Fields{
				"module": module,
				"old":    strategy,
				"new":    defaultStrategy(module),
			}).Info("update module strategy because of roles change")
			g.SetObject(ProposalStrategyKey(module), defaultStrategy(module))
		}
	}

	return boltvm.Success(nil)
}

func (g *GovStrategy) GetAllProposalStrategy() *boltvm.Response {
	ret := make([]*ProposalStrategy, 0)
	for _, module := range mgrs {
		ps := defaultStrategy(module)
		_ = g.GetObject(ProposalStrategyKey(module), ps)
		ret = append(ret, ps)
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf("marshal proposal strategy error: %v", err))
	}
	return boltvm.Success(data)
}

func (g *GovStrategy) GetProposalStrategy(pt string) *boltvm.Response {
	if err := repo.CheckManageModule(pt); err != nil {
		return boltvm.Error(boltvm.ProposalStrategyIllegalProposalTypeCode, fmt.Sprintf(string(boltvm.ProposalStrategyIllegalProposalTypeMsg), pt))
	}

	ps := defaultStrategy(pt)
	_ = g.GetObject(ProposalStrategyKey(pt), ps)

	pData, err := json.Marshal(ps)
	if err != nil {
		return boltvm.Error(boltvm.ProposalStrategyInternalErrCode, fmt.Sprintf(string(boltvm.ProposalStrategyInternalErrMsg), err.Error()))
	}
	return boltvm.Success(pData)
}

func ProposalStrategyKey(ps string) string {
	return fmt.Sprintf("%s-%s", PROPOSALSTRATEGY_PREFIX, ps)
}

func defaultStrategy(module string) *ProposalStrategy {
	return &ProposalStrategy{
		Module: module,
		Typ:    SimpleMajority,
		Extra:  repo.DefaultSimpleMajorityExpression,
		Status: governance.GovernanceAvailable,
	}
}
