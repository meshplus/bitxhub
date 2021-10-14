package contracts

import (
	"encoding/json"
	"fmt"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
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
			role := &Role{}
			if err := json.Unmarshal(res.Result, role); err != nil {
				return err
			}
			if regulatorAddr == role.ID {
				return nil
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
func (rm *RuleManager) Manage(eventTyp, proposalResult, lastStatus, ruleAddr string, extra []byte) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{
		constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, "", rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. other operation
	switch eventTyp {
	case string(governance.EventUpdate):
		info := &UpdateMasterRuleInfo{}
		if err := json.Unmarshal(extra, &info); err != nil {
			return boltvm.Error(fmt.Sprintf("unmarshal rule error: %v", err))
		}

		// 2.1 change old master rule status
		ok, errData := rm.RuleManager.ChangeStatus(info.OldRule.Address, proposalResult, string(info.OldRule.Status), []byte(info.OldRule.ChainID))
		if !ok {
			return boltvm.Error(fmt.Sprintf("change master error : %s", string(errData)))
		}

		// 2.2 change new master status, old master rule status change may influence new master rule, so do change status after other operation
		ok, errData = rm.RuleManager.ChangeStatus(ruleAddr, proposalResult, lastStatus, []byte(info.NewRule.ChainID))
		if !ok {
			return boltvm.Error(fmt.Sprintf("change new error : %s", string(errData)))
		}

		// 2.3 If the update succeeds, restore the status of the application chain
		if proposalResult == string(APPROVED) {
			res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "UnPauseAppchain", pb.String(info.AppchainInfo.ID), pb.String(string(info.AppchainInfo.Status)))
			if !res.Ok {
				return boltvm.Error(fmt.Sprintf("cross invoke UnPauseAppchain err: %s", res.Result))
			}
		}
	}

	return boltvm.Success(nil)
}

// =========== RegisterRule records the rule, and then automatically binds the rule if there is no master validation rule
func (rm *RuleManager) RegisterRule(chainID string, ruleAddress, ruleUrl string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	event := governance.EventRegister

	// 1 check permission: PermissionSelf
	if err := rm.checkPermission([]string{string(PermissionSelf)}, chainID, rm.Caller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error: %v", err))
	}

	// 2. check rule
	res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(chainID))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke GetAppchain error: %s", string(res.Result)))
	}
	appchainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(res.Result, appchainInfo); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal appchain error: %v", err))
	}
	if err := CheckRuleAddress(rm.Persister, ruleAddress, appchainInfo.ChainType); err != nil {
		return boltvm.Error(err.Error())
	}

	// 3. governance pre: check if exist and status
	_, err := rm.RuleManager.GovernancePre(ruleAddress, event, []byte(chainID))
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 4. register
	ok, data := rm.RuleManager.Register(chainID, ruleAddress, ruleUrl)
	if !ok {
		return boltvm.Error(fmt.Sprintf("register error: %s", string(data)))
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
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, chainID, rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. register default rule
	if err := rm.registerDefaultRule(chainID, chainType); err != nil {
		return boltvm.Error(err.Error())
	}

	// 3. register master rule
	if !ruleMgr.IsDefault(ruleAddress, chainType) {
		ok, data := rm.RuleManager.Register(chainID, ruleAddress, ruleUrl)
		if !ok {
			return boltvm.Error(fmt.Sprintf("register master rule error: %s", string(data)))
		}
	}

	// 4. bind master rule
	ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventBind), string(governance.GovernanceBindable), []byte(chainID))
	if !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
	}

	return boltvm.Success(nil)
}

func (rm *RuleManager) registerDefaultRule(chainID, chainType string) error {
	ok, data := rm.RuleManager.Register(chainID, validator.HappyRuleAddr, "")
	if !ok {
		return fmt.Errorf("register error: %v", string(data))
	}

	switch chainType {
	case appchainMgr.ChainTypeFabric1_4_3:
		fallthrough
	case appchainMgr.ChainTypeFabric1_4_4:
		ok, data := rm.RuleManager.Register(chainID, validator.FabricRuleAddr, "")
		if !ok {
			return fmt.Errorf("register error: %v", string(data))
		}
		ok, data = rm.RuleManager.Register(chainID, validator.SimFabricRuleAddr, "")
		if !ok {
			return fmt.Errorf("register error: %v", string(data))
		}
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
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. check appchain
	res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "PauseAppchain", pb.String(chainID))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke PauseAppchain error: %s", string(res.Result)))
	}
	appchain := &appchainMgr.Appchain{}
	if err := json.Unmarshal(res.Result, appchain); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal appchain error: %v", err))
	}

	// 3. check new rule
	if err := CheckRuleAddress(rm.Persister, newMasterruleAddress, appchain.ChainType); err != nil {
		return boltvm.Error(err.Error())
	}

	// 4. new rule governance pre: check if exist and status
	ruleInfo, err := rm.RuleManager.GovernancePre(newMasterruleAddress, governance.EventUpdate, []byte(chainID))
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(governance.EventUpdate), err))
	}
	newRule := ruleInfo.(*ruleMgr.Rule)

	// 5. submit proposal
	masterRule, err := rm.RuleManager.GetMaster(chainID)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	info := &UpdateMasterRuleInfo{
		OldRule:      masterRule,
		NewRule:      newRule,
		AppchainInfo: appchain,
	}
	infoData, err := json.Marshal(info)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal rule error: %v", err))
	}
	res = rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(event)),
		pb.String(string(RuleMgr)),
		pb.String(info.NewRule.Address),
		pb.String(string(info.NewRule.Status)),
		pb.String(reason),
		pb.Bytes(infoData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke SubmitProposal error: %s", string(res.Result)))
	}

	// 6. change new rule status
	if ok, data := rm.RuleManager.ChangeStatus(info.NewRule.Address, string(event), string(info.NewRule.Status), []byte(chainID)); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
	}

	// 7. operate master rule
	if ok, data := rm.RuleManager.ChangeStatus(masterRule.Address, string(governance.EventUnbind), string(masterRule.Status), []byte(chainID)); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// =========== LogoutRule logout the validation rule address with the chain id
func (rm *RuleManager) LogoutRule(chainID string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission: PermissionSelf
	if err := rm.checkPermission([]string{string(PermissionSelf)}, chainID, rm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. pre logout
	ruleInfo, err := rm.RuleManager.GovernancePre(ruleAddress, governance.EventLogout, []byte(chainID))
	if err != nil {
		return boltvm.Error(fmt.Sprintf("logout prepare error: %v", err))
	}
	rule := ruleInfo.(*ruleMgr.Rule)

	// 3. change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventLogout), string(rule.Status), []byte(chainID)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet("", nil)
}

// =========== PauseRule pause the proposals about rule of one chain.
// The rules management module only has updated proposals, which in this case are actually paused proposals.
func (rm *RuleManager) PauseRule(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	// 1. check permission: PermissionSpecific
	specificAddrs := []string{
		constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, "", rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. pause rule proposal
	rule, err := rm.RuleManager.GetMaster(chainID)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	if rule.Status == governance.GovernanceUnbinding {
		var ruleID string
		rules, err := rm.RuleManager.All([]byte(chainID))
		if err != nil {
			return boltvm.Error(err.Error())
		}
		for _, r := range rules.([]*ruleMgr.Rule) {
			if r.Status == governance.GovernanceBinding {
				ruleID = r.Address
				break
			}
		}
		res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal",
			pb.String(ruleID),
			pb.String(string(governance.EventPause)))
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("cross invoke LockLowPriorityProposal error: %s", string(res.Result)))
		}
		rm.Logger().WithFields(logrus.Fields{
			"chainID":    chainID,
			"MasterRule": rule,
			"proposalID": string(res.Result),
		}).Info("pause rule proposal")
	}

	return boltvm.Success(nil)
}

// =========== UnPauseRule unpause the proposals about rule of one chain.
func (rm *RuleManager) UnPauseRule(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	// 1. check permission: PermissionSpecific
	specificAddrs := []string{
		constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, "", rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. unpause rule proposal
	rule, err := rm.RuleManager.GetMaster(chainID)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	if rule.Status == governance.GovernanceUnbinding {
		var ruleID string
		rules, err := rm.RuleManager.All([]byte(chainID))
		if err != nil {
			return boltvm.Error(err.Error())
		}
		for _, r := range rules.([]*ruleMgr.Rule) {
			if r.Status == governance.GovernanceBinding {
				ruleID = r.Address
				break
			}
		}
		res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UnLockLowPriorityProposal",
			pb.String(ruleID),
			pb.String(string(governance.EventUnpause)))
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("cross invoke LockLowPriorityProposal error: %s", string(res.Result)))
		}
		rm.Logger().WithFields(logrus.Fields{
			"chainID":    chainID,
			"MasterRule": rule,
			"proposalID": string(res.Result),
			"ruleID":     ruleID,
		}).Info("unpause rule proposal")
	}

	return boltvm.Success(nil)
}

// ========================== Query interface ========================
func (rm *RuleManager) GetAllRules() *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	rules, err := rm.AllRules()
	if err != nil {
		return boltvm.Error(fmt.Sprintf("get all rules error: %v", err))
	}

	rulesData, err := json.Marshal(rules)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal rules error: %v", err))
	}

	return boltvm.Success(rulesData)
}

// CountAvailableRules counts all available rules (should be 0 or 1)
func (rm *RuleManager) CountAvailableRules(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.CountAvailable([]byte(chainID)))
}

// CountRules counts all rules of a chain
func (rm *RuleManager) CountRules(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.CountAll([]byte(chainID)))
}

// Rules returns all rules of a chain
func (rm *RuleManager) Rules(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	rules, err := rm.RuleManager.All([]byte(chainID))
	if err != nil {
		return boltvm.Error(err.Error())
	}

	if data, err := json.Marshal(rules.([]*ruleMgr.Rule)); err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success(data)
	}

}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) GetRuleByAddr(chainID, ruleAddr string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	rule, err := rm.RuleManager.QueryById(ruleAddr, []byte(chainID))
	if err != nil {
		return boltvm.Error(err.Error())
	}
	if data, err := json.Marshal(rule.(*ruleMgr.Rule)); err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success(data)
	}
}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) GetMasterRule(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	rule, err := rm.RuleManager.GetMaster(chainID)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	if data, err := json.Marshal(rule); err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success(data)
	}
}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) HasMasterRule(chainID string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.HasMaster(chainID), nil)
}

func (rm *RuleManager) IsAvailableRule(chainID, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.IsAvailable(chainID, ruleAddress), nil)
}

func CheckRuleAddress(persister governance.Persister, addr, chainType string) error {
	if ruleMgr.IsDefault(addr, chainType) {
		return nil
	}

	if _, err := types.HexDecodeString(addr); err != nil {
		return fmt.Errorf("illegal rule addr: %s", addr)
	}

	account1 := persister.GetAccount(addr)
	account := account1.(ledger.IAccount)
	if account.Code() == nil {
		return fmt.Errorf("the validation rule does not exist")
	}

	return nil
}
