package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

// todo: get this from config file
const relayRootPrefix = "did:bitxhub:relayroot:"

type AppchainManager struct {
	boltvm.Stub
	appchainMgr.AppchainManager
}

// extra: appchainMgr.Appchain
func (am *AppchainManager) Manage(eventTyp string, proposalResult, lastStatus string, extra []byte) *boltvm.Response {
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + err.Error())
	}
	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSpecific)),
		pb.String(""),
		pb.String(am.CurrentCaller()),
		pb.Bytes(addrsData))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	am.AppchainManager.Persister = am.Stub
	chain := &appchainMgr.Appchain{}
	if err := json.Unmarshal(extra, chain); err != nil {
		return boltvm.Error("unmarshal json error:" + err.Error())
	}

	ok, errData := am.AppchainManager.ChangeStatus(chain.ID, proposalResult, lastStatus, nil)
	if !ok {
		return boltvm.Error(string(errData))
	}

	if proposalResult == string(APPROVED) {
		//relaychainAdmin := relayRootPrefix + am.Caller()
		switch eventTyp {
		case string(governance.EventRegister):
			// When applying a new method for appchain is successful
			// 1. notify InterchainContract
			// 2. notify MethodRegistryContract to auditApply this method, then register appchain info
			res = am.CrossInvoke(constant.InterchainContractAddr.String(), "Register", pb.String(chain.ID))
			if !res.Ok {
				return res
			}

			if chain.Rule != "" {
				res = am.CrossInvoke(constant.RuleManagerContractAddr.String(), "BindRule",
					pb.String(chain.ID),
					pb.String(chain.Rule),
					pb.String(chain.RuleUrl),
				)
				if !res.Ok {
					return res
				}
			}
			//res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "AuditApply",
			//	pb.String(relaychainAdmin), pb.String(chain.ID), pb.Int32(1), pb.Bytes(nil))
			//if !res.Ok {
			//	return res
			//}
			//return am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Register",
			//	pb.String(relaychainAdmin), pb.String(chain.ID),
			//	pb.String(chain.DidDocAddr), pb.Bytes([]byte(chain.DidDocHash)), pb.Bytes(nil))
		case string(governance.EventUpdate):
			res := responseWrapper(am.AppchainManager.Update(extra))
			if !res.Ok {
				return res
			}
		}
	} else {
		//relaychainAdmin := relayRootPrefix + am.Caller()
		//switch eventTyp {
		//case string(governance.EventRegister):
		//	res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Audit",
		//		pb.String(relaychainAdmin), pb.String(chain.ID), pb.String(string(bitxid.Initial)), pb.Bytes(nil))
		//	if !res.Ok {
		//		return res
		//	}
		//
		//}
	}

	return boltvm.Success(nil)
}

func (am *AppchainManager) chainDefaultConfig(chain *appchainMgr.Appchain) error {
	if chain.ChainType == appchainMgr.FabricType {
		res := am.CrossInvoke(constant.RuleManagerContractAddr.String(), "DefaultRule", pb.String(chain.ID), pb.String(validator.FabricRuleAddr))
		if !res.Ok {
			return fmt.Errorf(string(res.Result))
		}
		res = am.CrossInvoke(constant.RuleManagerContractAddr.String(), "DefaultRule", pb.String(chain.ID), pb.String(validator.SimFabricRuleAddr))
		if !res.Ok {
			return fmt.Errorf(string(res.Result))
		}
	}
	return nil
}

// Register registers appchain info
// caller is the appchain manager address
// return appchain id, proposal id and error
func (am *AppchainManager) Register(method string, docAddr, docHash, validators string,
	consensusType, chainType, name, desc, version, pubkey, reason string) *boltvm.Response {
	var (
		addr string
		err  error
	)

	am.AppchainManager.Persister = am.Stub

	if pubkey != "" {
		addr, err = appchainMgr.GetAddressFromPubkey(pubkey)
		if err != nil {
			return boltvm.Error(fmt.Sprintf("get addr from public key: %v", err))
		}
	} else {
		addr = am.Caller()
	}

	res := am.CrossInvoke(constant.RoleContractAddr.String(), "GetRoleByAddr", pb.String(addr))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke IsAnyAdmin error : %s", string(res.Result)))
	} else {
		if string(res.Result) != string(NoRole) {
			return boltvm.Error(fmt.Sprintf("Please do not register appchain with other administrator's public key (address: %s, role: %s)", addr, res.Result))
		}
	}

	res = am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelfAdmin)),
		pb.String(addr),
		pb.String(am.Caller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	//res := am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Apply",
	//	pb.String(appchainAdminDID), pb.String(appchainMethod), pb.Bytes(nil))
	//if !res.Ok {
	//	return res
	//}

	appchainAdminDID := fmt.Sprintf("%s:%s:%s", repo.BitxhubRootPrefix, method, addr)
	appchainDID := fmt.Sprintf("%s:%s:.", repo.BitxhubRootPrefix, method)

	chain := &appchainMgr.Appchain{
		ID:            appchainDID,
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		ChainType:     chainType,
		Status:        governance.GovernanceRegisting,
		Desc:          desc,
		Version:       version,
		PublicKey:     pubkey,
		DidDocAddr:    docAddr,
		DidDocHash:    docHash,
		OwnerDID:      appchainAdminDID,
	}
	chainData, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error("marshal chain error:" + err.Error())
	}

	ok, data := am.AppchainManager.Register(chainData)
	if !ok {
		return boltvm.Error("register error: " + string(data))
	}

	registerRes := &governance.RegisterResult{}
	if err := json.Unmarshal(data, registerRes); err != nil {
		return boltvm.Error("register error: " + string(data))
	}
	if registerRes.IsRegistered {
		return boltvm.Error("appchain has registered, chain id: " + registerRes.ID)
	}

	res = am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventRegister)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(appchainDID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.String(reason),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return res
	}

	return getGovernanceRet(string(res.Result), []byte(appchainDID))
}

// Register registers appchain info
// caller is the appchain manager address
// return appchain id, proposal id and error
func (am *AppchainManager) RegisterV2(method string, docAddr, docHash, validators string,
	consensusType, chainType, name, desc, version, pubkey, reason, rule, ruleUrl string) *boltvm.Response {
	var (
		addr string
		err  error
	)

	am.AppchainManager.Persister = am.Stub

	if pubkey != "" {
		addr, err = appchainMgr.GetAddressFromPubkey(pubkey)
		if err != nil {
			return boltvm.Error(fmt.Sprintf("get addr from public key: %v", err))
		}
	} else {
		addr = am.Caller()
	}

	res := am.CrossInvoke(constant.RoleContractAddr.String(), "GetRoleByAddr", pb.String(addr))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke IsAnyAdmin error : %s", string(res.Result)))
	} else {
		if string(res.Result) != string(NoRole) {
			return boltvm.Error(fmt.Sprintf("Please do not register appchain with other administrator's public key (address: %s, role: %s)", addr, res.Result))
		}
	}

	res = am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelfAdmin)),
		pb.String(addr),
		pb.String(am.Caller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	appchainAdminDID := fmt.Sprintf("%s:%s:%s", repo.BitxhubRootPrefix, method, addr)
	appchainDID := fmt.Sprintf("%s:%s:.", repo.BitxhubRootPrefix, method)

	if rule != validator.FabricRuleAddr && rule != validator.SimFabricRuleAddr && rule != validator.HappyRuleAddr {
		res = am.CrossInvoke(constant.RuleManagerContractAddr.String(), "GetRuleByAddr",
			pb.String(appchainDID), pb.String(rule))
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("rule %s is not registered to chain ID %s: %s", rule, appchainDID, string(res.Result)))
		}

		ruleInfo := &ruleMgr.Rule{}
		if err := json.Unmarshal(res.Result, ruleInfo); err != nil {
			return boltvm.Error(fmt.Sprintf("unmarshal rule error: %v", err))
		}
		if ruleInfo.Status != governance.GovernanceBindable {
			return boltvm.Error(fmt.Sprintf("the rule is not bindable: %s", rule))
		}
	}

	chain := &appchainMgr.Appchain{
		ID:            appchainDID,
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		ChainType:     chainType,
		Status:        governance.GovernanceRegisting,
		Desc:          desc,
		Version:       version,
		PublicKey:     pubkey,
		DidDocAddr:    docAddr,
		DidDocHash:    docHash,
		Rule:          rule,
		RuleUrl:       ruleUrl,
		OwnerDID:      appchainAdminDID,
	}
	chainData, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error("marshal chain error:" + err.Error())
	}

	ok, data := am.AppchainManager.Register(chainData)
	if !ok {
		return boltvm.Error("register error: " + string(data))
	}

	registerRes := &governance.RegisterResult{}
	if err := json.Unmarshal(data, registerRes); err != nil {
		return boltvm.Error("register error: " + string(data))
	}
	if registerRes.IsRegistered {
		return boltvm.Error("appchain has registered, chain id: " + registerRes.ID)
	}

	res = am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventRegister)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(appchainDID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.String(reason),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return res
	}

	return getGovernanceRet(string(res.Result), []byte(appchainDID))
}

// UpdateAppchain updates available appchain
func (am *AppchainManager) UpdateAppchain(id, docAddr, docHash, validators string, consensusType, chainType,
	name, desc, version, pubkey, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, data := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(data))
	}

	oldChainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(data, oldChainInfo); err != nil {
		return boltvm.Error(err.Error())
	}

	if err := am.checkCaller(oldChainInfo, am.CurrentCaller(), PermissionSelf); err != nil {
		return boltvm.Error(err.Error())
	}

	// pre update
	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventUpdate, nil); !ok {
		return boltvm.Error("update prepare error: " + string(data))
	}

	chain := &appchainMgr.Appchain{
		ID:            id,
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		ChainType:     chainType,
		Desc:          desc,
		Version:       version,
		PublicKey:     pubkey,
		DidDocAddr:    docAddr,
		DidDocHash:    docHash,
	}

	if oldChainInfo.PublicKey != "" && chain.PublicKey != "" {
		oldAddr, err := appchainMgr.GetAddressFromPubkey(oldChainInfo.PublicKey)
		if err != nil {
			return boltvm.Error(err.Error())
		}
		newAddr, err := appchainMgr.GetAddressFromPubkey(chain.PublicKey)
		if err != nil {
			return boltvm.Error(err.Error())
		}
		if oldAddr != newAddr {
			return boltvm.Error(fmt.Sprintf("pubkey can not be updated (oldAddr : %s, newAddr: %s)", oldAddr, newAddr))
		}
	} else if oldChainInfo.PublicKey != chain.PublicKey {
		return boltvm.Error(fmt.Sprintf("pubkey can not be updated (old pubkey : %s, new pubkey: %s)", oldChainInfo.PublicKey, chain.PublicKey))
	}

	if oldChainInfo.Validators == chain.Validators {
		chain.Status = oldChainInfo.Status
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
	} else {
		chain.Status = governance.GovernanceAvailable
		data, err := json.Marshal(chain)
		if err != nil {
			return boltvm.Error(err.Error())
		}
		res := am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
			pb.String(am.Caller()),
			pb.String(string(governance.EventUpdate)),
			pb.String(""),
			pb.String(string(AppchainMgr)),
			pb.String(id),
			pb.String(string(oldChainInfo.Status)),
			pb.String(reason),
			pb.Bytes(data),
		)
		if !res.Ok {
			return boltvm.Error("submit proposal error:" + string(res.Result))
		}

		if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventUpdate), string(chain.Status), nil); !ok {
			return boltvm.Error(string(data))
		}

		return getGovernanceRet(string(res.Result), nil)
	}
}

// FreezeAppchain freezes available appchain
func (am *AppchainManager) FreezeAppchain(id, reason string) *boltvm.Response {
	// 1. CheckPermission: PermissionSelfAdmin
	am.AppchainManager.Persister = am.Stub
	ok, chainData := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(chainData))
	}
	chainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(chainData, chainInfo); err != nil {
		return boltvm.Error(err.Error())
	}

	if err := am.checkCaller(chainInfo, am.CurrentCaller(), PermissionAdmin); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. GovernancePre: check if exist and status
	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventFreeze, nil); !ok {
		return boltvm.Error("freeze prepare error: " + string(data))
	}

	// 4. SubmitProposal
	res := am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventFreeze)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.String(reason),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	// 5. ChangeStatus
	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventFreeze), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// ActivateAppchain activate frozen appchain
func (am *AppchainManager) ActivateAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	// 1. CheckPermission: PermissionSelfAdmin
	ok, chainData := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(chainData))
	}
	chainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(chainData, chainInfo); err != nil {
		return boltvm.Error(err.Error())
	}

	if err := am.checkCaller(chainInfo, am.CurrentCaller(), PermissionSelfAdmin); err != nil {
		return boltvm.Error(err.Error())
	}

	// 2. GovernancePre: check if exist and status
	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventActivate, nil); !ok {
		return boltvm.Error("activate prepare error: " + string(data))
	}

	// 3. SubmitProposal
	res := am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventActivate)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.String(reason),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	// 4. ChangeStatus
	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventActivate), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// LogoutAppchain updates available appchain
func (am *AppchainManager) LogoutAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, chainData := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(chainData))
	}
	chainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(chainData, chainInfo); err != nil {
		return boltvm.Error(err.Error())
	}

	if err := am.checkCaller(chainInfo, am.CurrentCaller(), PermissionSelf); err != nil {
		return boltvm.Error(err.Error())
	}

	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventLogout, nil); !ok {
		return boltvm.Error("logout prepare error: " + string(data))
	}

	res := am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventLogout)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.String(reason),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventLogout), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
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
	ok, data := am.AppchainManager.GovernancePre(id, event, nil)
	if !ok {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %s", string(event), string(data)))
	}
	chainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(data, chainInfo); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal chain error: %v", err))
	}

	// 3. change status
	if chainInfo.Status == governance.GovernanceAvailable {
		if ok, data1 := am.AppchainManager.ChangeStatus(id, string(event), string(chainInfo.Status), nil); !ok {
			return boltvm.Error(fmt.Sprintf("change status error: %s", string(data1)))
		}
	}

	am.Logger().WithFields(logrus.Fields{
		"chainID": id,
	}).Info("appchain pause")

	return boltvm.Success(data)
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
	ok, data := am.AppchainManager.GovernancePre(id, event, nil)
	if !ok {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %s", string(event), string(data)))
	}

	// 3. change status
	if governance.GovernanceFrozen != governance.GovernanceStatus(lastStatus) {
		if ok, data := am.AppchainManager.ChangeStatus(id, string(event), lastStatus, nil); !ok {
			return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
		}
	}

	am.Logger().WithFields(logrus.Fields{
		"chainID": id,
	}).Info("appchain unpause")

	return boltvm.Success(nil)
}

// CountAvailableAppchains counts all available appchains
func (am *AppchainManager) CountAvailableAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAvailable(nil))
}

// CountAppchains counts all appchains including approved, rejected or registered
func (am *AppchainManager) CountAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAll(nil))
}

// Appchains returns all appchains
func (am *AppchainManager) Appchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.All(nil))
}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) IsAppchainAdmin() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	addr := am.Caller()

	ok, data := am.AppchainManager.All(nil)
	if !ok {
		return boltvm.Error(string(data))
	}
	chains := make([]*appchainMgr.Appchain, 0)
	err := json.Unmarshal(data, &chains)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	for _, chain := range chains {
		if chain.Status == governance.GovernanceUnavailable {
			continue
		}
		tmpAddr, err := chain.GetAdminAddress()
		if err != nil {
			return boltvm.Error("get addr error: " + err.Error())
		}
		if tmpAddr == addr {
			return boltvm.Success(nil)
		}
	}

	return boltvm.Error("not found the appchain admin")
}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) GetAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.QueryById(id, nil))
}

func (am *AppchainManager) GetIdByAddr(addr string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.GetIdByAddr(addr))
}

// GetPubKeyByChainID can get aim chain's public key using aim chain ID
func (am *AppchainManager) GetPubKeyByChainID(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.GetPubKeyByChainID(id))
}

func (am *AppchainManager) DeleteAppchain(toDeleteMethod string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	if res := am.IsAdmin(); !res.Ok {
		return res
	}
	res := am.CrossInvoke(constant.InterchainContractAddr.String(), "DeleteInterchain", pb.String(toDeleteMethod))
	if !res.Ok {
		return res
	}
	//relayAdminDID := relayRootPrefix + am.Caller()
	//res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Delete", pb.String(relayAdminDID), pb.String(toDeleteMethod), nil)
	//if !res.Ok {
	//	return res
	//}
	return responseWrapper(am.AppchainManager.DeleteAppchain(toDeleteMethod))
}

func (am *AppchainManager) IsAdmin() *boltvm.Response {
	ret := am.CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", pb.String(am.Caller()))
	is, err := strconv.ParseBool(string(ret.Result))
	if err != nil {
		return boltvm.Error(fmt.Errorf("judge caller type: %w", err).Error())
	}

	if !is {
		return boltvm.Error("caller is not an admin account")
	}
	return boltvm.Success([]byte("1"))
}

func responseWrapper(ok bool, data []byte) *boltvm.Response {
	if ok {
		return boltvm.Success(data)
	}
	return boltvm.Error(string(data))
}

func (am *AppchainManager) IsAvailable(chainId string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	is, data := am.AppchainManager.QueryById(chainId, nil)

	if !is {
		return boltvm.Error("get appchain info error: " + string(data))
	}

	app := &appchainMgr.Appchain{}
	if err := json.Unmarshal(data, app); err != nil {
		return boltvm.Error("unmarshal error: " + err.Error())
	}

	for _, s := range appchainMgr.AppchainAvailableState {
		if app.Status == s {
			return boltvm.Success([]byte("true"))
		}
	}
	return boltvm.Success([]byte("false"))
}

func (am *AppchainManager) checkCaller(appchain *appchainMgr.Appchain, caller string, permission Permission) error {
	addr, err := appchain.GetAdminAddress()
	if err != nil {
		return fmt.Errorf("get appchain admin error: %v", err)
	}

	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(permission)),
		pb.String(addr),
		pb.String(caller),
		pb.Bytes(nil))
	if !res.Ok {
		return fmt.Errorf("check permission error: %v", string(res.Result))
	}
	return nil
}
