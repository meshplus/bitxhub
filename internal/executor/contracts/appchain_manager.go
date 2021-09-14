package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/iancoleman/orderedmap"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

type AppchainManager struct {
	boltvm.Stub
	appchainMgr.AppchainManager
}

func (am *AppchainManager) checkPermission(permissions []string, appchainID string, regulatorAddr string, specificAddrsData []byte) error {
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			res := am.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", pb.String(appchainID))
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
			res := am.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin",
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
// extra: register - rule info
func (am *AppchainManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := am.checkPermission([]string{string(PermissionSpecific)}, objId, am.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. change status
	if ok, data := am.AppchainManager.ChangeStatus(objId, proposalResult, lastStatus, nil); !ok {
		return boltvm.Error(fmt.Sprintf("change status error:%s", string(data)))
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			var ruleInfo ruleMgr.Rule
			if err := json.Unmarshal(extra, &ruleInfo); err != nil {
				return boltvm.Error(fmt.Sprintf("unmarshal extra error: %v", err))
			}
			res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage",
				pb.String(string(governance.EventBind)),
				pb.String(proposalResult),
				pb.String(string(governance.GovernanceBindable)),
				pb.String(ruleInfo.Address),
				pb.Bytes([]byte(objId))) // chain ID
			if !res.Ok {
				return res
			}

			res = am.CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", pb.String(objId))
			if !res.Ok {
				return res
			}
		case string(governance.EventFreeze):
			return am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", pb.String(objId))
		case string(governance.EventActivate):
			return am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", pb.String(objId))
		}
	} else {
		switch eventTyp {
		case string(governance.EventRegister):
			var ruleInfo ruleMgr.Rule
			if err := json.Unmarshal(extra, &ruleInfo); err != nil {
				return boltvm.Error(fmt.Sprintf("unmarshal extra error: %v", err))
			}
			res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "Manage",
				pb.String(string(governance.EventBind)),
				pb.String(proposalResult),
				pb.String(string(governance.GovernanceBindable)),
				pb.String(ruleInfo.Address),
				pb.Bytes([]byte(objId)))
			if !res.Ok {
				return res
			}
		case string(governance.EventLogout):
			if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", pb.String(objId)); !res.Ok {
				return res
			}

			return am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "UnPauseRule", pb.String(objId))
		}
	}

	return boltvm.Success(nil)
}

// =========== RegisterAppchain registers appchain info, returns proposal id and error
func (am *AppchainManager) RegisterAppchain(appchainID string, trustRoot []byte, broker, desc, masterRule, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventRegister

	// 1. check appchain admin
	res := am.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", pb.String(appchainID))
	if !res.Ok {
		// 1.1 register appchain admin
		res := am.CrossInvoke(constant.RoleContractAddr.Address().String(), "RegisterRole",
			pb.String(am.Caller()),
			pb.String(string(AppchainAdmin)),
			pb.String(""),
			pb.String(appchainID),
			pb.String(""),
		)
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("cross invoke role Register error : %s", string(res.Result)))
		}
	} else {
		// 1.2 check permission: self
		role := &Role{}
		if err := json.Unmarshal(res.Result, role); err != nil {
			return boltvm.Error(fmt.Sprintf("unmarshal role error : %v", err))
		}
		if am.CurrentCaller() != role.ID {
			return boltvm.Error(fmt.Sprintf("check permission error: regulatorAddr(%s) does not have the permission", am.Caller()))
		}
	}

	// 2. governancePre: check status
	if _, err := am.AppchainManager.GovernancePre(appchainID, event, nil); err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 3. check rule
	ruleInfo := &ruleMgr.Rule{
		Address: masterRule,
		RuleUrl: "",
		ChainID: appchainID,
		Master:  false,
		Status:  governance.GovernanceBindable,
	}
	if !ruleMgr.IsDefault(masterRule) {
		res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetRuleByAddr",
			pb.String(appchainID), pb.String(masterRule))
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("rule %s is not registered to chain ID %s: %s", masterRule, appchainID, string(res.Result)))
		}

		if err := json.Unmarshal(res.Result, ruleInfo); err != nil {
			return boltvm.Error(fmt.Sprintf("unmarshal rule error: %v", err))
		}
		if ruleInfo.Status != governance.GovernanceBindable {
			return boltvm.Error(fmt.Sprintf("the rule is not bindable: %s", masterRule))
		}
	}
	ruleData, err := json.Marshal(ruleInfo)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal rule error : %v", err))
	}

	// 4. submit proposal
	proposalRes := am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(event)),
		pb.String(string(AppchainMgr)),
		pb.String(appchainID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.String(reason),
		pb.Bytes(ruleData),
	)
	if !proposalRes.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(proposalRes.Result)))
	}

	// 5. register info
	chain := &appchainMgr.Appchain{
		ID:        appchainID,
		TrustRoot: trustRoot,
		Broker:    broker,
		Desc:      desc,
		Version:   0,
		Status:    governance.GovernanceRegisting,
	}

	ok, data := am.AppchainManager.Register(chain)
	if !ok {
		return boltvm.Error(fmt.Sprintf("register error: %s", string(data)))
	}

	if chain.Broker == string(constant.InterBrokerContractAddr) {
		am.registerRelayChain(appchainID)
	}

	// 6. change rule status
	res = am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "BindFirstMasterRule", pb.String(appchainID), pb.String(masterRule))
	if !res.Ok {
		return res
	}

	return getGovernanceRet(string(proposalRes.Result), []byte(appchainID))
}

func (am *AppchainManager) registerRelayChain(chainID string) {
	am.AppchainManager.Persister = am.Stub
	relayChainIdMap := orderedmap.New()
	_ = am.GetObject(appchainMgr.RelaychainType, relayChainIdMap)
	relayChainIdMap.Set(chainID, struct{}{})
	am.SetObject(appchainMgr.RelaychainType, *relayChainIdMap)
}

// =========== UpdateAppchain updates appchain info.
// This is currently no need for voting governance.
func (am *AppchainManager) UpdateAppchain(id, desc string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventUpdate

	// 1. check permission: PermissionSelf
	if err := am.checkPermission([]string{string(PermissionSelf)}, id, am.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. governance pre: check if exist and status
	_, err := am.AppchainManager.GovernancePre(id, event, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 3. update info
	updateInfo := &appchainMgr.Appchain{
		ID:   id,
		Desc: desc,
	}
	ok, data := am.AppchainManager.Update(updateInfo)
	if !ok {
		return boltvm.Error(fmt.Sprintf("update appchain error: %s", string(data)))
	}

	return getGovernanceRet("", nil)
}

// =========== FreezeAppchain freezes appchain
func (am *AppchainManager) FreezeAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return am.basicGovernance(id, reason, []string{string(PermissionAdmin)}, governance.EventFreeze)
}

// =========== ActivateAppchain activates frozen appchain
func (am *AppchainManager) ActivateAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	// check rule
	res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", pb.String(id))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke GetMasterRule error: %s", string(res.Result)))
	}
	rule := &ruleMgr.Rule{}
	if err := json.Unmarshal(res.Result, rule); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal rule error: %v", err))
	}
	if rule.Status != governance.GovernanceAvailable {
		return boltvm.Error(fmt.Sprintf("chain master rule(%s:%s) is updating, can not activate appchain(%s).", rule.Address, string(rule.Status), id))
	}

	return am.basicGovernance(id, reason, []string{string(PermissionSelf), string(PermissionAdmin)}, governance.EventActivate)
}

// =========== LogoutAppchain logouts appchain
func (am *AppchainManager) LogoutAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	governanceRes := am.basicGovernance(id, reason, []string{string(PermissionSelf)}, governance.EventLogout)
	if !governanceRes.Ok {
		return governanceRes
	}

	// pause service
	if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", pb.String(id)); !res.Ok {
		return res
	}

	// pause rule proposal
	if res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "PauseRule", pb.String(id)); !res.Ok {
		return res
	}

	return governanceRes
}

func (am *AppchainManager) basicGovernance(id, reason string, permissions []string, event governance.EventType) *boltvm.Response {
	// 1. check permission
	if err := am.checkPermission(permissions, id, am.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. governance pre: check if exist and status
	chain, err := am.AppchainManager.GovernancePre(id, event, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}
	chainInfo := chain.(*appchainMgr.Appchain)

	// 3. submit proposal
	res := am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(event)),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 4. change status
	if ok, data := am.AppchainManager.ChangeStatus(id, string(event), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	am.Logger().WithFields(logrus.Fields{
		"id": chainInfo.ID,
	}).Info(fmt.Sprintf("Appchain is doing event %s", event))

	return getGovernanceRet(string(res.Result), nil)
}

// =========== PauseAppchain freezes appchain without governance
// This function is triggered when the master rule is updating.
// Information about the appchain before the pause is returned
//  so that the appchain can be restored when unpause is invoked.
func (am *AppchainManager) PauseAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventPause

	// 1. check permission: PermissionSpecific
	specificAddrs := []string{constant.RuleManagerContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := am.checkPermission([]string{string(PermissionSpecific)}, id, am.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. governance pre: check if exist and status
	chain, err := am.AppchainManager.GovernancePre(id, event, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}
	chainInfo := chain.(*appchainMgr.Appchain)

	// 3. change status
	if chainInfo.Status == governance.GovernanceAvailable {
		if ok, data := am.AppchainManager.ChangeStatus(id, string(event), string(chainInfo.Status), nil); !ok {
			return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
		}
		// 4. pause service
		if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", pb.String(id)); !res.Ok {
			return res
		}
	}

	am.Logger().WithFields(logrus.Fields{
		"chainID": id,
	}).Info("appchain pause")

	chainData, err := json.Marshal(chainInfo)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal chain error: %v", err))
	}
	return boltvm.Success(chainData)
}

// =========== UnPauseAppchain restores to the state before the appchain was suspended
// This exist when the rule is update successsfully
func (am *AppchainManager) UnPauseAppchain(id, lastStatus string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventUnpause

	// 1. check permission: PermissionSpecific
	specificAddrs := []string{constant.RuleManagerContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := am.checkPermission([]string{string(PermissionSpecific)}, id, am.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. governance pre: check if exist and status
	_, err = am.AppchainManager.GovernancePre(id, event, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 3. change status
	if governance.GovernanceFrozen != governance.GovernanceStatus(lastStatus) {
		if ok, data := am.AppchainManager.ChangeStatus(id, string(event), lastStatus, nil); !ok {
			return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
		}
		// 4. unpause services
		if string(governance.GovernanceAvailable) == lastStatus {
			if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", pb.String(id)); !res.Ok {
				return res
			}
		}
	}

	am.Logger().WithFields(logrus.Fields{
		"chainID": id,
	}).Info("appchain unpause")

	return boltvm.Success(nil)
}

// ========================== Query interface ========================

// CountAvailableAppchains counts all available appchains
func (am *AppchainManager) CountAvailableAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAvailable(nil))
}

// CountAppchains counts all appchains including all status
func (am *AppchainManager) CountAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAll(nil))
}

// Appchains returns all appchains
func (am *AppchainManager) Appchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	chains, err := am.AppchainManager.All(nil)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	if data, err := json.Marshal(chains.([]*appchainMgr.Appchain)); err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success(data)
	}

}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) GetAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	chain, err := am.AppchainManager.QueryById(id, nil)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	if data, err := json.Marshal(chain.(*appchainMgr.Appchain)); err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success(data)
	}
}

func (am *AppchainManager) IsAvailable(chainID string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	chain, err := am.AppchainManager.QueryById(chainID, nil)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success([]byte(strconv.FormatBool(chain.(*appchainMgr.Appchain).IsAvailable())))
}

func (am *AppchainManager) GetBitXHubChainIDs() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	relayChainIdMap := orderedmap.New()
	_ = am.GetObject(appchainMgr.RelaychainType, relayChainIdMap)

	if data, err := json.Marshal(relayChainIdMap.Keys()); err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success(data)
	}
}

func responseWrapper(ok bool, data []byte) *boltvm.Response {
	if ok {
		return boltvm.Success(data)
	}
	return boltvm.Error(string(data))
}

func AppchainKey(id string) string {
	return fmt.Sprintf("%s-%s", appchainMgr.PREFIX, id)
}
