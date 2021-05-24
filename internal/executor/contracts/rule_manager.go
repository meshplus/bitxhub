package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/tidwall/gjson"
)

// RuleManager is the contract manage validation rules
type RuleManager struct {
	boltvm.Stub
	ruleMgr.RuleManager
}

// extra: ruleMgr.rule
func (rm *RuleManager) Manage(eventTyp string, proposalResult string, extra []byte) *boltvm.Response {
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + err.Error())
	}
	res := rm.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSpecific)),
		pb.String(""),
		pb.String(rm.CurrentCaller()),
		pb.Bytes(addrsData))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	rm.RuleManager.Persister = rm.Stub
	rule := &ruleMgr.Rule{}
	if err := json.Unmarshal(extra, &rule); err != nil {
		return boltvm.Error("unmarshal rule error:" + err.Error())
	}

	ok, errData := rm.RuleManager.ChangeStatus(rule.Address, proposalResult, []byte(rule.ChainId))
	if !ok {
		return boltvm.Error(string(errData))
	}

	switch eventTyp {
	case string(governance.EventBind):
		if string(APPOVED) == proposalResult {
			ok, data := rm.RuleManager.SetMaster(rule.Address, []byte(rule.ChainId), true)
			if !ok {
				return boltvm.Error(string(data))
			}
		}
	case string(governance.EventUpdate):
		// get master
		masterRule := &ruleMgr.Rule{}
		hasMaster := rm.RuleManager.HasMaster(rule.ChainId)
		if hasMaster {
			ok, masterData := rm.RuleManager.GetMaster(rule.ChainId)
			if !ok {
				return boltvm.Error("get master error: " + string(masterData))
			}
			if err := json.Unmarshal(masterData, &masterRule); err != nil {
				return boltvm.Error("unmarshal masterRule error:" + err.Error())
			}
		}
		// change master status
		ok, errData := rm.RuleManager.ChangeStatus(masterRule.Address, proposalResult, []byte(masterRule.ChainId))
		if !ok {
			return boltvm.Error(string(errData))
		}

		if string(APPOVED) == proposalResult {
			// new rule
			ok, data := rm.RuleManager.SetMaster(rule.Address, []byte(rule.ChainId), true)
			if !ok {
				return boltvm.Error(string(data))
			}
			// old rule
			ok, data = rm.RuleManager.SetMaster(masterRule.Address, []byte(masterRule.ChainId), false)
			if !ok {
				return boltvm.Error(string(data))
			}
		}
	case string(governance.EventUnbind):
		if string(APPOVED) == proposalResult {
			ok, data := rm.RuleManager.SetMaster(rule.Address, []byte(rule.ChainId), false)
			if !ok {
				return boltvm.Error(string(data))
			}
		}
	case string(governance.EventFreeze):
		if string(REJECTED) == proposalResult {
			if rule.Master {
				res = rm.DefaultRule(rule.ChainId, rule.Address)
				if !res.Ok {
					return boltvm.Error("default rule error:" + string(res.Result))
				}
			}
		} else {
			ok, data := rm.RuleManager.SetMaster(rule.Address, []byte(rule.ChainId), false)
			if !ok {
				return boltvm.Error(string(data))
			}
		}
	case string(governance.EventLogout):
		if string(REJECTED) == proposalResult {
			if rule.Master {
				res = rm.DefaultRule(rule.ChainId, rule.Address)
				if !res.Ok {
					return boltvm.Error("default rule error:" + string(res.Result))
				}
			}
		} else {
			ok, data := rm.RuleManager.SetMaster(rule.Address, []byte(rule.ChainId), false)
			if !ok {
				return boltvm.Error(string(data))
			}
		}
	}

	return boltvm.Success(nil)
}

// Register records the rule, and then automatically binds the rule if there is no master validation rule
func (rm *RuleManager) RegisterRule(chainId string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	if err := rm.checkPermission(chainId, PermissionSelfAdmin, nil); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chainId)); !res.Ok {
		return boltvm.Error("cross invoke IsAvailable error: " + string(res.Result))
	}

	// 3. check rule
	if err := rm.checkRuleAddress(ruleAddress); err != nil {
		return boltvm.Error(err.Error())
	}

	// 4. register
	ok, data := rm.RuleManager.Register(chainId, ruleAddress)
	if !ok {
		return boltvm.Error("register error: " + string(data))
	}

	registerRes := &governance.RegisterResult{}
	if err := json.Unmarshal(data, registerRes); err != nil {
		return boltvm.Error("unmarshal error: " + err.Error())
	}
	if registerRes.IsRegistered {
		return boltvm.Error("rule has registered, chain id: " + chainId + ", rule addr: " + ruleAddress)
	}

	// 5. determine whether to bind, if not bind:
	// -  existing master validation rules
	// -  there is already rule in binding
	if ok, _ := rm.RuleManager.BindPre(chainId, ruleAddress, false); !ok {
		return getGovernanceRet("", nil)
	}

	// 6. submit proposal
	// 7. change status
	return rm.bindRule(chainId, ruleAddress, governance.EventBind)
}

// DefaultRule automatically adds default rules to the appchain after the appchain is registered successfully
// DefaultRule Adds default rules automatically. The rule will automatically bound if there is no master rule currently. All processes do not require vote.
// Possible situations:
// - Default validation rules are automatically added after successful application chain registration
// - It may be necessary to restore the identity of the master rule if it fails to freeze or logout
func (rm *RuleManager) DefaultRule(chainId string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	specificAddrs := []string{constant.AppchainMgrContractAddr.Address().String(), constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + err.Error())
	}
	if err := rm.checkPermission(chainId, PermissionSpecific, addrsData); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. register
	ok, data := rm.RuleManager.Register(chainId, ruleAddress)
	if !ok {
		return boltvm.Error("register error: " + string(data))
	}
	isRegistered := gjson.Get(string(data), "is_registered").Bool()
	if !isRegistered {
		ok, data := rm.RuleManager.SetMaster(ruleAddress, []byte(chainId), true)
		if !ok {
			return boltvm.Error(string(data))
		}
	}

	// 3. default bind
	ok, data = rm.RuleManager.CountAvailable([]byte(chainId))
	if !ok {
		return boltvm.Error("count available error: " + string(data))
	}
	if string(data) == strconv.Itoa(0) {
		ok, data = rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventBind), []byte(chainId))
		if !ok {
			return boltvm.Error("change status error: " + string(data))
		}
		ok, data = rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventApprove), []byte(chainId))
		if !ok {
			return boltvm.Error("change status error: " + string(data))
		}
	}

	return boltvm.Success(nil)
}

// BindRule binds the validation rule address with the chain id
func (rm *RuleManager) UpdateMasterRule(chainId string, newMasterruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	if err := rm.checkPermission(chainId, PermissionSelfAdmin, nil); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chainId)); !res.Ok {
		return boltvm.Error("cross invoke IsAvailable error: " + string(res.Result))
	}

	// 3. check new rule
	if err := rm.checkRuleAddress(newMasterruleAddress); err != nil {
		return boltvm.Error(err.Error())
	}

	// 4. new rule pre bind
	if ok, data := rm.RuleManager.BindPre(chainId, newMasterruleAddress, true); !ok {
		return boltvm.Error("bind prepare error: " + string(data))
	}

	// 5. submit new rule bind proposal
	// 6. change new rule status
	res := rm.bindRule(chainId, newMasterruleAddress, governance.EventUpdate)
	if !res.Ok {
		return res
	}

	// 7. operate master rule
	hasMaster := rm.RuleManager.HasMaster(chainId)
	if hasMaster {
		ok, masterData := rm.RuleManager.GetMaster(chainId)
		if !ok {
			return boltvm.Error("get master error: " + string(masterData))
		}
		masterRuleAddress := gjson.Get(string(masterData), "address").String()
		if ok, data := rm.RuleManager.ChangeStatus(masterRuleAddress, string(governance.EventUnbind), []byte(chainId)); !ok {
			return boltvm.Error("change status error: " + string(data))
		}
	}
	return res
}

// BindRule binds the validation rule address with the chain id
func (rm *RuleManager) BindRule(chainId string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	if err := rm.checkPermission(chainId, PermissionSelfAdmin, nil); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chainId)); !res.Ok {
		return boltvm.Error("cross invoke IsAvailable error: " + string(res.Result))
	}

	// 3. check rule
	if err := rm.checkRuleAddress(ruleAddress); err != nil {
		return boltvm.Error(err.Error())
	}

	// 4. pre bind
	if ok, data := rm.RuleManager.BindPre(chainId, ruleAddress, false); !ok {
		return boltvm.Error("bind prepare error: " + string(data))
	}

	// 5. submit proposal
	// 6. change status
	return rm.bindRule(chainId, ruleAddress, governance.EventBind)
}

func (rm *RuleManager) bindRule(chainId string, ruleAddr string, event governance.EventType) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// submit proposal
	ruleRes := rm.GetRuleByAddr(chainId, ruleAddr)
	if !ruleRes.Ok {
		return boltvm.Error("get rule by addr error: " + string(ruleRes.Result))
	}

	res := rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(event)),
		pb.String(""),
		pb.String(string(RuleMgr)),
		pb.String(ruleAddr),
		pb.Bytes(ruleRes.Result),
	)
	if !res.Ok {
		return boltvm.Error("cross invoke SubmitProposal error: " + string(res.Result))
	}

	// change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddr, string(governance.EventBind), []byte(chainId)); !ok {
		return boltvm.Error("change status error: " + string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// UnbindRule unbinds the validation rule address with the chain id
func (rm *RuleManager) UnbindRule(chainId string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	if err := rm.checkPermission(chainId, PermissionSelfAdmin, nil); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chainId)); !res.Ok {
		return boltvm.Error("cross invoke IsAvailable error: " + string(res.Result))
	}

	// 3. submit proposal
	ruleRes := rm.GetRuleByAddr(chainId, ruleAddress)
	if !ruleRes.Ok {
		return boltvm.Error("get rule by addr error: " + string(ruleRes.Result))
	}

	res := rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventUnbind)),
		pb.String(""),
		pb.String(string(RuleMgr)),
		pb.String(ruleAddress),
		pb.Bytes(ruleRes.Result),
	)
	if !res.Ok {
		return boltvm.Error("cross invoke SubmitProposal error: " + string(res.Result))
	}

	// 4. change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventUnbind), []byte(chainId)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// FreezeRule freezes the validation rule address with the chain id
func (rm *RuleManager) FreezeRule(chainId string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	if err := rm.checkPermission(chainId, PermissionSelfAdmin, nil); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chainId)); !res.Ok {
		return boltvm.Error("cross invoke IsAvailable error: " + string(res.Result))
	}

	// 3. submit proposal
	ruleRes := rm.GetRuleByAddr(chainId, ruleAddress)
	if !ruleRes.Ok {
		return boltvm.Error("get rule by addr error: " + string(ruleRes.Result))
	}

	res := rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventFreeze)),
		pb.String(""),
		pb.String(string(RuleMgr)),
		pb.String(ruleAddress),
		pb.Bytes(ruleRes.Result),
	)
	if !res.Ok {
		return boltvm.Error("cross invoke SubmitProposal error: " + string(res.Result))
	}

	// 4. change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventFreeze), []byte(chainId)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// ActivateRule activate the validation rule address with the chain id
func (rm *RuleManager) ActivateRule(chainId string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	if err := rm.checkPermission(chainId, PermissionSelfAdmin, nil); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chainId)); !res.Ok {
		return boltvm.Error("cross invoke IsAvailable error: " + string(res.Result))
	}

	// 3. submit proposal
	ruleRes := rm.GetRuleByAddr(chainId, ruleAddress)
	if !ruleRes.Ok {
		return boltvm.Error("get rule by addr error: " + string(ruleRes.Result))
	}

	res := rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventActivate)),
		pb.String(""),
		pb.String(string(RuleMgr)),
		pb.String(ruleAddress),
		pb.Bytes(ruleRes.Result),
	)
	if !res.Ok {
		return boltvm.Error("cross invoke SubmitProposal error: " + string(res.Result))
	}

	// 4. change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventActivate), []byte(chainId)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// LogoutRule logout the validation rule address with the chain id
func (rm *RuleManager) LogoutRule(chainId string, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub

	// 1. check permission
	if err := rm.checkPermission(chainId, PermissionSelf, nil); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. check appchain
	if res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(chainId)); !res.Ok {
		return boltvm.Error("cross invoke IsAvailable error: " + string(res.Result))
	}

	// 3. submit proposal
	ruleRes := rm.GetRuleByAddr(chainId, ruleAddress)
	if !ruleRes.Ok {
		return boltvm.Error("get rule by addr error: " + string(ruleRes.Result))
	}

	res := rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventLogout)),
		pb.String(""),
		pb.String(string(RuleMgr)),
		pb.String(ruleAddress),
		pb.Bytes(ruleRes.Result),
	)
	if !res.Ok {
		return boltvm.Error("cross invoke SubmitProposal error: " + string(res.Result))
	}

	// 4. change status
	if ok, data := rm.RuleManager.ChangeStatus(ruleAddress, string(governance.EventLogout), []byte(chainId)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// CountAvailableRules counts all available rules (should be 0 or 1)
func (rm *RuleManager) CountAvailableRules(chainId string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.CountAvailable([]byte(chainId)))
}

// CountRules counts all rules of a chain
func (rm *RuleManager) CountRules(chainId string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.CountAll([]byte(chainId)))
}

// Rules returns all rules of a chain
func (rm *RuleManager) Rules(chainId string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.All([]byte(chainId)))
}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) GetRuleByAddr(chainId, ruleAddr string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.QueryById(ruleAddr, []byte(chainId)))
}

// GetRuleByAddr returns rule by appchain id and rule address
func (rm *RuleManager) GetMasterRule(chainId string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.GetMaster(chainId))
}

// GetAvailableRuleAddr returns available rule address by appchain id and rule address
func (rm *RuleManager) GetAvailableRuleAddr(chainId string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.GetAvailableRuleAddress(chainId))
}

func (rm *RuleManager) IsAvailableRule(chainId, ruleAddress string) *boltvm.Response {
	rm.RuleManager.Persister = rm.Stub
	return responseWrapper(rm.RuleManager.IsAvailable(chainId, ruleAddress))
}

func (rm *RuleManager) checkRuleAddress(addr string) error {
	if addr == validator.FabricRuleAddr || addr == validator.SimFabricRuleAddr {
		return nil
	}

	ok, account1 := rm.Persister.GetAccount(addr)
	if !ok {
		return fmt.Errorf("get account error")
	}

	account := account1.(*ledger.Account)
	if account.Code() == nil {
		return fmt.Errorf("the validation rule does not exist")
	}

	return nil
}

func (rm *RuleManager) checkPermission(chainId string, p Permission, specificAddrsData []byte) error {
	res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(chainId))
	if !res.Ok {
		return fmt.Errorf("cross invoke GetAppchain error: %s", string(res.Result))
	}

	pubKeyStr := gjson.Get(string(res.Result), "public_key").String()
	addr, err := getAddr(pubKeyStr)
	if err != nil {
		return fmt.Errorf("get addr error: %s", err.Error())
	}

	res = rm.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(p)),
		pb.String(addr),
		pb.String(rm.CurrentCaller()),
		pb.Bytes(specificAddrsData))
	if !res.Ok {
		return fmt.Errorf("check permission error: %s", string(res.Result))
	}

	return nil
}
func RuleKey(id string) string {
	return ruleMgr.RULEPREFIX + id
}
