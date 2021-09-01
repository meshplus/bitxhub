package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/eth-kit/ledger"
)

// RuleManager is the contract manage validation rules
type RuleManager struct {
	boltvm.Stub
	ruleMgr.RuleManager
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
// extra: rule info
func (rm *RuleManager) Manage(eventTyp, proposalResult, lastStatus, ruleAddr string, extra []byte) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, "", rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	var ruleInfo ruleMgr.Rule
	if err := json.Unmarshal(extra, &ruleInfo); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal rule error: %v", err))
	}

	// 2. other operation
	switch eventTyp {
	case string(governance.EventUpdate):
		// get master
		masterRule, err := rm.RuleManager.GetMaster(ruleInfo.ChainID)
		if err != nil {
			return boltvm.Error(fmt.Sprintf("get master error: %v", err))
		}
		if masterRule == nil {
			return boltvm.Error("there is no master rule")
		}
		// change master status
		ok, errData := rm.RuleManager.ChangeStatus(masterRule.Address, proposalResult, string(governance.GovernanceAvailable), []byte(masterRule.ChainID))
		if !ok {
			return boltvm.Error(string(errData))
		}
	}

	// 3. change status, old master rule status change may influence new mater rule, so do change status after other operation
	ok, errData := rm.RuleManager.ChangeStatus(ruleAddr, proposalResult, lastStatus, []byte(ruleInfo.ChainID))
	if !ok {
		return boltvm.Error(string(errData))
	}

	return boltvm.Success(nil)
}

// =========== RegisterRule records the rule, and then automatically binds the rule if there is no master validation rule
func (rm *RuleManager) RegisterRule(chainID string, ruleAddress, ruleUrl string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	event := governance.EventRegister

	// 1. check appchain
	res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(chainID))
	if res.Ok {
		// 1.1 check permission: PermissionSelf
		if err := rm.checkPermission([]string{string(PermissionSelf)}, chainID, rm.CurrentCaller(), nil); err != nil {
			return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
		}
	} else {
		// 1.2 register appchain admin
		res := rm.CrossInvoke(constant.RoleContractAddr.Address().String(), "RegisterRole",
			pb.String(rm.Caller()),
			pb.String(string(AppchainAdmin)),
			pb.String(""),
			pb.String(chainID),
			pb.String(""),
		)
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("cross invoke role Register error : %s", string(res.Result)))
		}
	}

	// 2. check rule
	if err := rm.checkRuleAddress(ruleAddress); err != nil {
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

// =========== BindRule adds master rules to the appchain after the appchain is registered successfully
// - Master validation rules are automatically added after successful application chain registration
func (rm *RuleManager) BindFirstMasterRule(chainID, ruleAddress string) *boltvm.Response {
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

	// 2. register
	if ruleMgr.IsDefault(ruleAddress) {
		ok, data := rm.RuleManager.Register(chainID, ruleAddress, "")
		if !ok {
			return boltvm.Error(fmt.Sprintf("register error: %s", string(data)))
		}
	}

	// 3. default bind
	ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventBind), string(governance.GovernanceBindable), []byte(chainID))
	if !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
	}

	return boltvm.Success(nil)
}

// =========== UpdateMasterRule binds the validation rule address with the chain id and unbinds the master rule
func (rm *RuleManager) UpdateMasterRule(chainID string, newMasterruleAddress, reason string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission: PermissionSelf
	if err := rm.checkPermission([]string{string(PermissionSelf)}, chainID, rm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", pb.String(chainID)); !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke IsAvailable error: %s", string(res.Result)))
	}

	// 3. check new rule
	if err := rm.checkRuleAddress(newMasterruleAddress); err != nil {
		return boltvm.Error(err.Error())
	}

	// 4. new rule governance pre: check if exist and status
	ruleInfo, err := rm.RuleManager.GovernancePre(newMasterruleAddress, governance.EventUpdate, []byte(chainID))
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(governance.EventUpdate), err))
	}
	newRule := ruleInfo.(*ruleMgr.Rule)

	// 5. submit new rule bind proposal
	// 6. change new rule status
	res := rm.bindRule(newRule, governance.EventUpdate, reason)
	if !res.Ok {
		return res
	}

	// 7. operate master rule
	masterRule, err := rm.RuleManager.GetMaster(chainID)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("get master error: %v", err))
	}
	if masterRule == nil {
		return boltvm.Error("there is no master rule")
	}
	if ok, data := rm.RuleManager.ChangeStatus(masterRule.Address, string(governance.EventUnbind), string(masterRule.Status), []byte(chainID)); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
	}
	return res
}

func (rm *RuleManager) bindRule(rule *ruleMgr.Rule, event governance.EventType, reason string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	ruleData, err := json.Marshal(rule)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal rule error: %v", err))
	}

	// submit proposal
	res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(event)),
		pb.String(string(RuleMgr)),
		pb.String(rule.Address),
		pb.String(string(rule.Status)),
		pb.String(reason),
		pb.Bytes(ruleData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke SubmitProposal error: %s", string(res.Result)))
	}

	// change status
	if ok, data := rm.RuleManager.ChangeStatus(rule.Address, string(event), string(rule.Status), []byte(rule.ChainID)); !ok {
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

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", pb.String(chainID)); !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke IsAvailable error: %s", string(res.Result)))
	}

	// 3. pre logout
	ruleInfo, err := rm.RuleManager.GovernancePre(ruleAddress, governance.EventLogout, []byte(chainID))
	if err != nil {
		return boltvm.Error(fmt.Sprintf("logout prepare error: %v", err))
	}
	rule := ruleInfo.(*ruleMgr.Rule)

	// 4. change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventLogout), string(rule.Status), []byte(chainID)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet("", nil)
}

// ========================== Query interface ========================

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

func (rm *RuleManager) checkRuleAddress(addr string) error {
	if ruleMgr.IsDefault(addr) {
		return nil
	}

	account1 := rm.Persister.GetAccount(addr)

	account := account1.(ledger.IAccount)
	if account.Code() == nil {
		return fmt.Errorf("the validation rule does not exist")
	}

	return nil
}
