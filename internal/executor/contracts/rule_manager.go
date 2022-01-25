package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/eth-kit/ledger"
)

// RuleManager is the contract manage validation rules
type RuleManager struct {
	boltvm.Stub
	ruleMgr.RuleManager
}

type UpdateMasterRuleInfo struct {
	OldRule      *ruleMgr.Rule         `json:"old_rule"`
	NewRule      *ruleMgr.Rule         `json:"new_rule"`
	AppchainInfo *appchainMgr.Appchain `json:"appchain_info"`
}

func (rm *RuleManager) checkPermission(permissions []string, appchainID string, regulatorAddr string, specificAddrsData []byte) error {
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			res := rm.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", pb.String(appchainID))
			if !res.Ok {
				return fmt.Errorf("cross invoke GetAppchainAdmin error:%s", string(res.Result))
			}
			roles := []*Role{}
			if err := json.Unmarshal(res.Result, &roles); err != nil {
				return err
			}
			for _, r := range roles {
				if regulatorAddr == r.ID {
					return nil
				}
			}
		case string(PermissionAdmin):
			res := rm.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin",
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

// =========== Manage does some subsequent operations when the proposal is over
// Currently here are only update master rule events
// extra: update :UpdateMasterRuleInfo
func (rm *RuleManager) Manage(eventTyp, proposalResult, lastStatus, chainRuleID string, extra []byte) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{
		constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, "", rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.RuleNoPermissionCode, fmt.Sprintf(string(boltvm.RuleNoPermissionMsg), rm.CurrentCaller(), err.Error()))
	}

	// 2. other operation
	switch eventTyp {
	case string(governance.EventUpdate):
		info := &UpdateMasterRuleInfo{}
		if err := json.Unmarshal(extra, &info); err != nil {
			return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
		}

		// 2.1 change old master rule status
		ok, errData := rm.RuleManager.ChangeStatus(info.OldRule.Address, proposalResult, string(info.OldRule.Status), []byte(info.OldRule.ChainID))
		if !ok {
			return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("change master error : %s", string(errData))))
		}

		// 2.2 change new master status, old master rule status change may influence new master rule, so do change status after other operation
		ok, errData = rm.RuleManager.ChangeStatus(info.NewRule.Address, proposalResult, lastStatus, []byte(info.NewRule.ChainID))
		if !ok {
			return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("change new error : %s", string(errData))))
		}

		// 2.3 If the update succeeds, restore the status of the application chain
		if proposalResult == string(APPROVED) {
			res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "UnPauseAppchain", pb.String(info.AppchainInfo.ID), pb.String(string(info.AppchainInfo.Status)))
			if !res.Ok {
				return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("cross invoke UnPauseAppchain err: %s", res.Result)))
			}
		}
	}

	if err := rm.postAuditRuleEvent(strings.Split(chainRuleID, ":")[0]); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("post audit rule event error: %v", err)))
	}
	return boltvm.Success(nil)
}

// =========== RegisterRule records the rule, and then automatically binds the rule if there is no master validation rule
func (rm *RuleManager) RegisterRule(chainID string, ruleAddress, ruleUrl string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	event := governance.EventRegister

	// 1 check permission: PermissionSelf
	if err := rm.checkPermission([]string{string(PermissionSelf)}, chainID, rm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.RuleNoPermissionCode, fmt.Sprintf(string(boltvm.RuleNoPermissionMsg), rm.CurrentCaller(), err.Error()))
	}

	// 2. check appchain
	res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(chainID))
	if !res.Ok {
		return boltvm.Error(boltvm.RuleNonexistentChainCode, fmt.Sprintf(string(boltvm.RuleNonexistentChainMsg), chainID, string(res.Result)))
	}
	appchainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(res.Result, appchainInfo); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}
	if appchainInfo.Status == governance.GovernanceForbidden {
		return boltvm.Error(boltvm.RuleAppchainForbiddenCode, fmt.Sprintf(string(boltvm.RuleAppchainForbiddenMsg), appchainInfo.ID))
	}

	// 3. check if rule deployed
	if res := CheckRuleAddress(rm.Persister, ruleAddress, appchainInfo.ChainType); !res.Ok {
		return res
	}

	// 4. governance pre: check if exist and status
	if _, be := rm.RuleManager.GovernancePre(ruleAddress, event, []byte(chainID)); be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 5. check built-in
	isDefault := ruleMgr.IsDefault(ruleAddress, appchainInfo.ChainType)
	if isDefault {
		return boltvm.Error(boltvm.RuleRegisterDefaultCode, fmt.Sprintf(string(boltvm.RuleRegisterDefaultMsg), ruleAddress))
	}

	// 6. register
	rm.RuleManager.Register(chainID, ruleAddress, ruleUrl, rm.GetTxTimeStamp(), isDefault)

	if err := rm.postAuditRuleEvent(chainID); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("post audit rule event error: %v", err)))
	}
	return getGovernanceRet("", nil)
}

// =========== RegisterRuleFirst registers the default rule and binds the specified master rule
// Only called after the appchain registration proposal has been voted through
// The master rule is checked(has deployed if not default) before the call
//   and is checked when submitting appchain register proposal
func (rm *RuleManager) RegisterRuleFirst(chainID, chainType, ruleAddress, ruleUrl string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	specificAddrs := []string{constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, chainID, rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.RuleNoPermissionCode, fmt.Sprintf(string(boltvm.RuleNoPermissionMsg), rm.CurrentCaller(), err.Error()))
	}

	// 2. register default rule
	if err := rm.registerDefaultRule(chainID, chainType); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("register default rule error: %v", err)))
	}

	// 3. register master rule
	if !ruleMgr.IsDefault(ruleAddress, chainType) {
		rm.RuleManager.Register(chainID, ruleAddress, ruleUrl, rm.GetTxTimeStamp(), false)
	}

	// 4. bind master rule
	ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventBind), string(governance.GovernanceBindable), []byte(chainID))
	if !ok {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	if err := rm.postAuditRuleEvent(chainID); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("post audit rule event error: %v", err)))
	}

	return boltvm.Success(nil)
}

func (rm *RuleManager) registerDefaultRule(chainID, chainType string) error {
	rm.RuleManager.Register(chainID, validator.HappyRuleAddr, "", rm.GetTxTimeStamp(), true)

	switch chainType {
	case appchainMgr.ChainTypeFabric1_4_3:
		fallthrough
	case appchainMgr.ChainTypeFabric1_4_4:
		rm.RuleManager.Register(chainID, validator.FabricRuleAddr, "", rm.GetTxTimeStamp(), true)
		rm.RuleManager.Register(chainID, validator.SimFabricRuleAddr, "", rm.GetTxTimeStamp(), true)
	case appchainMgr.ChainTypeHyperchain1_8_3:
	case appchainMgr.ChainTypeHyperchain1_8_6:
	case appchainMgr.ChainTypeFlato1_0_0:
	case appchainMgr.ChainTypeFlato1_0_3:
	case appchainMgr.ChainTypeFlato1_0_6:
	case appchainMgr.ChainTypeBCOS2_6_0:
	case appchainMgr.ChainTypeCITA20_2_2:
	case appchainMgr.ChainTypeETH:
		// Todo(fbz): register default rule
	}

	return nil
}

// =========== UpdateMasterRule binds the validation rule address with the chain id and unbinds the master rule
func (rm *RuleManager) UpdateMasterRule(chainID string, newMasterruleAddress, reason string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	event := governance.EventUpdate

	// 1. check permission: PermissionSelf
	if err := rm.checkPermission([]string{string(PermissionSelf)}, chainID, rm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.RuleNoPermissionCode, fmt.Sprintf(string(boltvm.RuleNoPermissionMsg), rm.CurrentCaller(), err.Error()))
	}

	// 2. check appchain
	res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(chainID))
	if !res.Ok {
		return boltvm.Error(boltvm.RuleNonexistentChainCode, fmt.Sprintf(string(boltvm.RuleNonexistentChainMsg), chainID, string(res.Result)))
	}
	appchainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(res.Result, appchainInfo); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}
	if appchainInfo.Status != governance.GovernanceAvailable && appchainInfo.Status != governance.GovernanceFrozen {
		return boltvm.Error(boltvm.RuleAppchainStatusErrorCode, fmt.Sprintf(string(boltvm.RuleAppchainStatusErrorMsg), appchainInfo.ID, string(appchainInfo.Status)))
	}

	// 3. check new rule
	if res := CheckRuleAddress(rm.Persister, newMasterruleAddress, appchainInfo.ChainType); !res.Ok {
		return res
	}

	// 4. new rule governance pre: check if exist and status
	ruleInfo, be := rm.RuleManager.GovernancePre(newMasterruleAddress, governance.EventUpdate, []byte(chainID))
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	newRule := ruleInfo.(*ruleMgr.Rule)

	// 5. pause appchain
	if res = rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "PauseAppchain", pb.String(chainID)); !res.Ok {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("cross invoke PauseAppchain error: %s", string(res.Result))))
	}

	// 6. submit proposal
	masterRule, err := rm.RuleManager.GetMaster(chainID)
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}
	info := &UpdateMasterRuleInfo{
		OldRule:      masterRule,
		NewRule:      newRule,
		AppchainInfo: appchainInfo,
	}
	infoData, err := json.Marshal(info)
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("marshal rule error: %v", err)))
	}
	res = rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(event)),
		pb.String(string(RuleMgr)),
		pb.String(info.NewRule.GetChainRuleID()),
		pb.String(string(info.NewRule.Status)),
		pb.String(reason),
		pb.Bytes(infoData),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("cross invoke SubmitProposal error: %s", string(res.Result))))
	}

	// 7. change new rule status
	if ok, data := rm.RuleManager.ChangeStatus(info.NewRule.Address, string(event), string(info.NewRule.Status), []byte(chainID)); !ok {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	// 8. operate master rule
	if ok, data := rm.RuleManager.ChangeStatus(masterRule.Address, string(governance.EventUnbind), string(masterRule.Status), []byte(chainID)); !ok {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := rm.postAuditRuleEvent(chainID); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("post audit rule event error: %v", err)))
	}
	return getGovernanceRet(string(res.Result), nil)
}

// =========== LogoutRule logout the validation rule address with the chain id
func (rm *RuleManager) LogoutRule(chainID string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	event := governance.EventLogout

	// 1. check permission: PermissionSelf
	if err := rm.checkPermission([]string{string(PermissionSelf)}, chainID, rm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.RuleNoPermissionCode, fmt.Sprintf(string(boltvm.RuleNoPermissionMsg), rm.CurrentCaller(), err.Error()))
	}

	// 2. check appchain
	res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(chainID))
	if !res.Ok {
		return boltvm.Error(boltvm.RuleNonexistentChainCode, fmt.Sprintf(string(boltvm.RuleNonexistentChainMsg), chainID, string(res.Result)))
	}
	appchainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(res.Result, appchainInfo); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}
	if appchainInfo.Status == governance.GovernanceForbidden {
		return boltvm.Error(boltvm.RuleAppchainForbiddenCode, fmt.Sprintf(string(boltvm.RuleAppchainForbiddenMsg), appchainInfo.ID))
	}

	// 3. governance pre: check if exist and status
	ruleInfo, be := rm.RuleManager.GovernancePre(ruleAddress, event, []byte(chainID))
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	rule := ruleInfo.(*ruleMgr.Rule)

	// 4. check built-in
	isDefault := ruleMgr.IsDefault(ruleAddress, appchainInfo.ChainType)
	if isDefault {
		return boltvm.Error(boltvm.RuleLogoutDefaultCode, fmt.Sprintf(string(boltvm.RuleLogoutDefaultMsg), ruleAddress))
	}

	// 5. change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(event), string(rule.Status), []byte(chainID)); !ok {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("change status error: %v", string(data))))
	}

	if err := rm.postAuditRuleEvent(chainID); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("post audit rule event error: %v", err)))
	}
	return getGovernanceRet("", nil)
}

// =========== PauseRule pause the proposals about rule of one chain.
// The rules management module only has updated proposals, which in this case are actually paused proposals.
func (rm *RuleManager) ClearRule(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	// 1. check permission: PermissionSpecific
	specificAddrs := []string{
		constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, "", rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.RuleNoPermissionCode, fmt.Sprintf(string(boltvm.RuleNoPermissionMsg), rm.CurrentCaller(), err.Error()))
	}

	// 2. clear rule
	rules, err := rm.RuleManager.All([]byte(chainID))
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}

	bindingChainRuleID := ""
	for _, r := range rules.([]*ruleMgr.Rule) {
		if r.Status == governance.GovernanceBinding {
			bindingChainRuleID = r.GetChainRuleID()
		}

		if ok, data := rm.RuleManager.ChangeStatus(r.Address, string(governance.EventCLear), string(r.Status), []byte(chainID)); !ok {
			return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("change status error: %v", string(data))))
		}
	}

	// 3. clear rule proposal
	if res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "EndObjProposal",
		pb.String(bindingChainRuleID),
		pb.String(string(ClearReason)),
		pb.Bytes(nil)); !res.Ok {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("cross invoke EndObjProposal error: %s", string(res.Result))))
	}
	//if bindingChainRuleID != "" {
	//	res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetProposalsByObjId", pb.String(bindingChainRuleID))
	//	if !res.Ok {
	//		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("cross invoke GetProposalsByObjId error: %s", string(res.Result))))
	//	}
	//	ps := make([]*Proposal, 0)
	//	if err := json.Unmarshal(res.Result, &ps); err != nil {
	//		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("json unmarshal error: %s", err.Error())))
	//	}
	//
	//	for _, p := range ps {
	//		if res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "EndCurrentProposal",
	//			pb.String(p.Id),
	//			pb.String(string(ClearReason)),
	//			pb.Bytes(nil)); !res.Ok {
	//			return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("cross invoke EndCurrentProposal error: %s", string(res.Result))))
	//		}
	//		rm.Logger().WithFields(logrus.Fields{
	//			"chainRuleID": p.ObjId,
	//			"eventTyp":    p.EventType,
	//			"proposalID":  p.Id,
	//		}).Info("clear rule proposal")
	//	}
	//}

	if err := rm.postAuditRuleEvent(chainID); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("post audit rule event error: %v", err)))
	}

	return boltvm.Success(nil)
}

// ========================== Query interface ========================
func (rm *RuleManager) GetAllRules() *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	rules, err := rm.AllRules()
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("get all rules error: %v", err)))
	}

	rulesData, err := json.Marshal(rules)
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), fmt.Sprintf("marshal rules error: %v", err)))
	}

	return boltvm.Success(rulesData)
}

// CountAvailableRules counts all available rules (should be 0 or 1)
func (rm *RuleManager) CountAvailableRules(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return boltvm.ResponseWrapper(rm.RuleManager.CountAvailable([]byte(chainID)))
}

// CountRules counts all rules of a chain
func (rm *RuleManager) CountRules(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return boltvm.ResponseWrapper(rm.RuleManager.CountAll([]byte(chainID)))
}

// Rules returns all rules of a chain
func (rm *RuleManager) Rules(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	rules, err := rm.RuleManager.All([]byte(chainID))
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}

	if data, err := json.Marshal(rules.([]*ruleMgr.Rule)); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}

}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) GetRuleByAddr(chainID, ruleAddr string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	rule, err := rm.RuleManager.QueryById(ruleAddr, []byte(chainID))
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}
	if data, err := json.Marshal(rule.(*ruleMgr.Rule)); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) GetMasterRule(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	rule, err := rm.RuleManager.GetMaster(chainID)
	if err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	}
	if data, err := json.Marshal(rule); err != nil {
		return boltvm.Error(boltvm.RuleInternalErrCode, fmt.Sprintf(string(boltvm.RuleInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) HasMasterRule(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return boltvm.ResponseWrapper(rm.RuleManager.HasMaster(chainID), nil)
}

func (rm *RuleManager) IsAvailableRule(chainID, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return boltvm.ResponseWrapper(rm.RuleManager.IsAvailable(chainID, ruleAddress), nil)
}

func CheckRuleAddress(persister governance.Persister, addr, chainType string) *boltvm.Response {
	if ruleMgr.IsDefault(addr, chainType) {
		return boltvm.Success(nil)
	}

	if _, err := types.HexDecodeString(addr); err != nil {
		return boltvm.Error(boltvm.RuleIllegalRuleAddrCode, fmt.Sprintf(string(boltvm.RuleIllegalRuleAddrMsg), addr, err.Error()))
	}

	account1 := persister.GetAccount(addr)
	account := account1.(ledger.IAccount)
	if account.CodeHash() == nil || bytes.Equal(account.CodeHash(), crypto.Keccak256(nil)) {
		return boltvm.Error(boltvm.RuleNonexistentRuleCode, fmt.Sprintf(string(boltvm.RuleNonexistentRuleMsg), addr))
	}

	return boltvm.Success(nil)
}

func (rm *RuleManager) postAuditRuleEvent(chainID string) error {
	rm.RuleManager.Persister = rm.Stub
	ok, rulesData := rm.Get(ruleMgr.RuleKey(chainID))
	if !ok {
		return fmt.Errorf("not found rules %s", chainID)
	}

	auditInfo := &pb.AuditRelatedObjInfo{
		AuditObj: rulesData,
		RelatedChainIDList: map[string][]byte{
			chainID: {},
		},
		RelatedNodeIDList: map[string][]byte{},
	}
	rm.PostEvent(pb.Event_AUDIT_RULE, auditInfo)

	return nil
}
