package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
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
// extra: register - if bind default rule, update - chain info
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
	if proposalResult == string(APPOVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			if string(extra) == TRUE {
				if err = am.bindDefaultRule(objId); err != nil {
					return boltvm.Error(fmt.Sprintf("bind default rule error:%v", err))
				}
			}

			res := am.CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", pb.String(objId))
			if !res.Ok {
				return res
			}
		case string(governance.EventUpdate):
			res := responseWrapper(am.AppchainManager.Update(extra))
			if !res.Ok {
				return res
			}
		}
	}

	return boltvm.Success(nil)
}

func (am *AppchainManager) bindDefaultRule(chainID string) error {
	res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "DefaultRule", pb.String(chainID), pb.String(validator.FabricRuleAddr))
	if !res.Ok {
		return fmt.Errorf(string(res.Result))
	}
	res = am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "DefaultRule", pb.String(chainID), pb.String(validator.SimFabricRuleAddr))
	if !res.Ok {
		return fmt.Errorf(string(res.Result))
	}
	return nil
}

// =========== RegisterAppchain registers appchain info, returns proposal id and error
func (am *AppchainManager) RegisterAppchain(appchainID string, trustRoot []byte, broker, desc, version, bindDefaultRule, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventRegister

	// 1. register appchain admin
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

	// 2. governancePre: check status
	if _, _, err := am.AppchainManager.GovernancePre(appchainID, event, nil); err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 3. submit proposal
	res = am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(event)),
		pb.String(string(AppchainMgr)),
		pb.String(appchainID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.String(reason),
		pb.Bytes([]byte(bindDefaultRule)),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 4. register info
	chain := &appchainMgr.Appchain{
		ID:        appchainID,
		TrustRoot: trustRoot,
		Broker:    broker,
		Desc:      desc,
		Version:   version,
		Status:    governance.GovernanceRegisting,
	}
	chainData, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal chain error: %v", err))
	}

	ok, data := am.AppchainManager.Register(chainData)
	if !ok {
		return boltvm.Error(fmt.Sprintf("register error: %s", string(data)))
	}

	return getGovernanceRet(string(res.Result), []byte(appchainID))
}

// =========== UpdateAppchain updates appchain info.
// This is currently no need for voting governance.
func (am *AppchainManager) UpdateAppchain(id, desc, version string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventUpdate

	// 1. check permission: PermissionSelf
	if err := am.checkPermission([]string{string(PermissionSelf)}, id, am.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. governance pre: check if exist and status
	oldChain, _, err := am.AppchainManager.GovernancePre(id, event, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}
	oldChainInfo := oldChain.(*appchainMgr.Appchain)

	// 3. update info
	chain := &appchainMgr.Appchain{
		ID:        id,
		TrustRoot: oldChainInfo.TrustRoot,
		Broker:    oldChainInfo.Broker,
		Desc:      desc,
		Version:   version,
		Status:    oldChainInfo.Status,
	}
	data, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	res := responseWrapper(am.AppchainManager.Update(data))
	if !res.Ok {
		return res
	} else {
		return getGovernanceRet("", nil)
	}
}

// =========== FreezeAppchain freezes appchain
func (am *AppchainManager) FreezeAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventFreeze
	return am.basicGovernance(id, reason, []string{string(PermissionAdmin)}, event)
}

// =========== ActivateAppchain activates frozen appchain
func (am *AppchainManager) ActivateAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventActivate
	return am.basicGovernance(id, reason, []string{string(PermissionSelf), string(PermissionAdmin)}, event)
}

// =========== LogoutAppchain logouts appchain
func (am *AppchainManager) LogoutAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventLogout
	return am.basicGovernance(id, reason, []string{string(PermissionSelf)}, event)
}

func (am *AppchainManager) basicGovernance(id, reason string, permissions []string, event governance.EventType) *boltvm.Response {
	// 1. check permission
	if err := am.checkPermission(permissions, id, am.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. governance pre: check if exist and status
	chain, _, err := am.AppchainManager.GovernancePre(id, event, nil)
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

	return getGovernanceRet(string(res.Result), nil)
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
	if chains == nil {
		return boltvm.Success(nil)
	} else {
		if data, err := json.Marshal(chains.([]*appchainMgr.Appchain)); err != nil {
			return boltvm.Error(err.Error())
		} else {
			return boltvm.Success(data)
		}
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

func responseWrapper(ok bool, data []byte) *boltvm.Response {
	if ok {
		return boltvm.Success(data)
	}
	return boltvm.Error(string(data))
}

func AppchainKey(id string) string {
	return fmt.Sprintf("%s-%s", appchainMgr.PREFIX, id)
}
