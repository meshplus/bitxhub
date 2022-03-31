package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/iancoleman/orderedmap"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

type AppchainManager struct {
	boltvm.Stub
	appchainMgr.AppchainManager
}

type RegisterAppchainInfo struct {
	ChainInfo  *appchainMgr.Appchain `json:"chain_info"`
	MasterRule *ruleMgr.Rule         `json:"master_rule"`
	AdminAddrs string                `json:"admin_addrs"` // comma-separated list of addresses
}

type UpdateAppchainInfo struct {
	Name       UpdateInfo `json:"name"`
	Desc       UpdateInfo `json:"desc"`
	TrustRoot  UpdateInfo `json:"trust_root"`
	AdminAddrs UpdateInfo `json:"admin_addrs"`
}

func (am *AppchainManager) checkPermission(permissions []string, appchainID string, regulatorAddr string, specificAddrsData []byte) error {
	am.AppchainManager.Persister = am.Stub

	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			idTmp, err := am.getChainIdByAdmin(regulatorAddr)
			if err == nil && idTmp == appchainID {
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
				return fmt.Errorf("unmarshal specific addrs error: %w", err)
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

func (am *AppchainManager) occupyChainID(chainID string) {
	am.AppchainManager.Persister = am.Stub
	chainIDMap := orderedmap.New()
	_ = am.GetObject(appchainMgr.ChainOccupyIdPrefix, chainIDMap)
	chainIDMap.Set(chainID, struct{}{})
	am.SetObject(appchainMgr.ChainOccupyIdPrefix, *chainIDMap)
}

func (am *AppchainManager) freeChainID(chainID string) {
	am.AppchainManager.Persister = am.Stub
	chainIDMap := orderedmap.New()
	_ = am.GetObject(appchainMgr.ChainOccupyIdPrefix, chainIDMap)
	chainIDMap.Delete(chainID)
	am.SetObject(appchainMgr.ChainOccupyIdPrefix, *chainIDMap)
}

func (am *AppchainManager) isOccupiedId(chainID string) bool {
	am.AppchainManager.Persister = am.Stub
	chainIdMap := orderedmap.New()
	if ok := am.GetObject(appchainMgr.ChainOccupyIdPrefix, chainIdMap); !ok {
		return false
	}
	if _, ok := chainIdMap.Get(chainID); !ok {
		return false
	}
	return true
}

func (am *AppchainManager) occupyChainName(name string, chainID string) {
	am.AppchainManager.Persister = am.Stub
	am.SetObject(appchainMgr.AppchainOccupyNameKey(name), chainID)
}

func (am *AppchainManager) freeChainName(name string) {
	am.AppchainManager.Persister = am.Stub
	am.Delete(appchainMgr.AppchainOccupyNameKey(name))
}

func (am *AppchainManager) isOccupiedName(name string) (bool, string) {
	am.AppchainManager.Persister = am.Stub
	chainId := ""
	ok := am.GetObject(appchainMgr.AppchainOccupyNameKey(name), &chainId)

	return ok, chainId
}

func (am *AppchainManager) registerRelayChain(chainID string) {
	am.AppchainManager.Persister = am.Stub
	relayChainIdMap := orderedmap.New()
	_ = am.GetObject(appchainMgr.RelaychainType, relayChainIdMap)
	relayChainIdMap.Set(chainID, struct{}{})

	am.SetObject(appchainMgr.RelaychainType, *relayChainIdMap)
}

func (am *AppchainManager) recordChainAdmins(chainID string, addrs []string) {
	am.AppchainManager.Persister = am.Stub
	am.SetObject(appchainMgr.AppAdminsChainKey(chainID), addrs)
	for _, addr := range addrs {
		am.SetObject(appchainMgr.AppchainAdminKey(addr), chainID)
	}
}

func (am *AppchainManager) getChainIdByAdmin(addr string) (string, error) {
	am.AppchainManager.Persister = am.Stub
	id := ""
	ok := am.GetObject(appchainMgr.AppchainAdminKey(addr), &id)
	if !ok {
		return "", fmt.Errorf("not found chain of admin %s", addr)
	}
	return id, nil
}

// =========== Manage does some subsequent operations when the proposal is over
func (am *AppchainManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	am.Logger().WithFields(logrus.Fields{
		"id": objId,
	}).Info("Appchain is managing")

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := am.checkPermission([]string{string(PermissionSpecific)}, objId, am.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.AppchainNoPermissionCode, fmt.Sprintf(string(boltvm.AppchainNoPermissionMsg), am.CurrentCaller(), err.Error()))
	}

	// 2. change status
	if eventTyp != string(governance.EventRegister) {
		if ok, data := am.AppchainManager.ChangeStatus(objId, proposalResult, lastStatus, nil); !ok {
			return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("change status error:%s", string(data))))
		}
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			if err := am.manageRegisterApprove(extra); err != nil {
				return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("manage register approve error: %v", err)))
			}
		case string(governance.EventUpdate):
			if err := am.manageUpdateApprove(objId, extra); err != nil {
				return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("manage update approve error: %v", err)))
			}
		case string(governance.EventFreeze):
			if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", pb.String(objId)); !res.Ok {
				return res
			}
		case string(governance.EventActivate):
			if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", pb.String(objId)); !res.Ok {
				return res
			}
		case string(governance.EventLogout):
			if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "ClearChainService", pb.String(objId)); !res.Ok {
				return res
			}

			if res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "ClearRule", pb.String(objId)); !res.Ok {
				return res
			}
		}
	} else {
		switch eventTyp {
		case string(governance.EventRegister):
			if err := am.manageRegisterReject(extra); err != nil {
				return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("manage register reject error: %v", err)))
			}
		case string(governance.EventUpdate):
			if err := am.manageUpdateReject(objId, extra); err != nil {
				return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("manage update reject error: %v", err)))
			}
		case string(governance.EventLogout):
			if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", pb.String(objId)); !res.Ok {
				return res
			}
		}
	}

	// The appchain ID does not exist when the appchain fails to register.
	// Therefore, you do not need to throw an appchain event.
	// This is the same principle as registering appchains without throwing events.
	if eventTyp != string(governance.EventRegister) || proposalResult != string(REJECTED) {
		if err := am.postAuditAppchainEvent(objId); err != nil {
			return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("post audit appchain event error: %v", err)))
		}
	}

	return boltvm.Success(nil)
}

func (am *AppchainManager) manageRegisterApprove(registerInfoData []byte) error {
	am.AppchainManager.Persister = am.Stub

	registerInfo := &RegisterAppchainInfo{}
	if err := json.Unmarshal(registerInfoData, &registerInfo); err != nil {
		return fmt.Errorf("unmarshal registerInfoData error: %v", err)
	}

	// 1. register appchain
	am.AppchainManager.Register(registerInfo.ChainInfo)

	am.recordChainAdmins(registerInfo.ChainInfo.ID, strings.Split(registerInfo.AdminAddrs, ","))
	// register relay chain
	if registerInfo.ChainInfo.IsBitXHub() {
		am.registerRelayChain(registerInfo.ChainInfo.ID)
	}

	// 2. register appchain admin
	res := am.CrossInvoke(constant.RoleContractAddr.Address().String(),
		"UpdateAppchainAdmin",
		pb.String(registerInfo.ChainInfo.ID),
		pb.String(registerInfo.AdminAddrs))
	if !res.Ok {
		return fmt.Errorf("cross invoke UpdateAppchainAdmin error: %s", string(res.Result))
	}

	// 3. register rule
	res = am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(),
		"RegisterRuleFirst",
		pb.String(registerInfo.ChainInfo.ID),
		pb.String(registerInfo.ChainInfo.ChainType),
		pb.String(registerInfo.MasterRule.Address),
		pb.String(registerInfo.MasterRule.RuleUrl),
	)
	if !res.Ok {
		return fmt.Errorf("cross invoke RegisterRuleFirst error: %s", string(res.Result))
	}

	return nil
}

func (am *AppchainManager) manageRegisterReject(registerInfoData []byte) error {
	registerInfo := &RegisterAppchainInfo{}
	if err := json.Unmarshal(registerInfoData, &registerInfo); err != nil {
		return fmt.Errorf("unmarshal registerInfoData error: %v", err)
	}

	// free pre-stored registration information
	am.freeChainID(registerInfo.ChainInfo.ID)
	am.freeChainName(registerInfo.ChainInfo.ChainName)
	if res := am.CrossInvoke(constant.RoleContractAddr.String(), "FreeAccount", pb.String(registerInfo.AdminAddrs)); !res.Ok {
		return fmt.Errorf("cross invoke FreeAccount error: %s", string(res.Result))
	}

	return nil
}

func (am *AppchainManager) manageUpdateApprove(appchainId string, updateInfoData []byte) error {
	am.AppchainManager.Persister = am.Stub

	updateInfo := &UpdateAppchainInfo{}
	if err := json.Unmarshal(updateInfoData, &updateInfo); err != nil {
		return fmt.Errorf("unmarshal updateInfoData error: %v", err)
	}

	// 1. update appchain info
	trustroot := []byte("")
	if updateInfo.TrustRoot.NewInfo != nil {
		trustroot = []byte(updateInfo.TrustRoot.NewInfo.(string))
	}

	updateChain := &appchainMgr.Appchain{
		ID:        appchainId,
		ChainName: updateInfo.Name.NewInfo.(string),
		TrustRoot: trustroot,
		Desc:      updateInfo.Desc.NewInfo.(string),
	}
	ok, data := am.AppchainManager.Update(updateChain)
	if !ok {
		return fmt.Errorf("update error: %s", string(data))
	}

	if updateInfo.Name.IsEdit {
		// free old name
		am.freeChainName(updateInfo.Name.OldInfo.(string))
	}

	if updateInfo.AdminAddrs.IsEdit {
		// free old admin addrs
		if res := am.CrossInvoke(constant.RoleContractAddr.String(), "FreeAccount", pb.String(updateInfo.AdminAddrs.OldInfo.(string))); !res.Ok {
			return fmt.Errorf("cross invoke FreeAccount error: %s", string(res.Result))
		}
		// record appchain admins
		am.recordChainAdmins(appchainId, strings.Split(updateInfo.AdminAddrs.NewInfo.(string), ","))
		// update appchain admin
		res := am.CrossInvoke(constant.RoleContractAddr.Address().String(),
			"UpdateAppchainAdmin",
			pb.String(appchainId),
			pb.String(updateInfo.AdminAddrs.NewInfo.(string)),
		)
		if !res.Ok {
			return fmt.Errorf("cross invoke UpdateAppchainAdmin error: %s", string(res.Result))
		}
	}

	// 2. unpause service
	if res := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", pb.String(appchainId)); !res.Ok {
		return fmt.Errorf("cross invoke UnPauseChainService error: %s", string(res.Result))
	}

	return nil
}

func (am *AppchainManager) manageUpdateReject(appchainId string, updateInfoData []byte) error {
	updateInfo := &UpdateAppchainInfo{}
	if err := json.Unmarshal(updateInfoData, &updateInfo); err != nil {
		return fmt.Errorf("unmarshal updateInfoData error: %v", err)
	}

	// free pre-stored update information
	if updateInfo.Name.IsEdit {
		// free new name
		am.freeChainName(updateInfo.Name.NewInfo.(string))
	}

	if updateInfo.AdminAddrs.IsEdit {
		// free new admin addrs
		if res := am.CrossInvoke(constant.RoleContractAddr.String(), "FreeAccount", pb.String(updateInfo.AdminAddrs.NewInfo.(string))); !res.Ok {
			return fmt.Errorf("cross invoke FreeAccount error: %s", string(res.Result))
		}
	}

	return nil
}

// =========== RegisterAppchain registers appchain info, returns proposal id and error
func (am *AppchainManager) RegisterAppchain(chainID string, chainName string, chainType string, trustRoot []byte, broker string, desc, masterRuleAddr, masterRuleUrl, adminAddrs, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventRegister

	// 1. check
	// 1.0 check chainID
	if chainID == "" {
		return boltvm.Error(boltvm.AppchainEmptyChainIDCode, string(boltvm.AppchainEmptyChainIDMsg))
	}
	if ok := am.isOccupiedId(chainID); ok {
		return boltvm.Error(boltvm.AppchainDuplicateChainIDCode, fmt.Sprintf(string(boltvm.AppchainDuplicateChainIDMsg), chainID))
	}

	// 1.1 check broker
	if broker == "" {
		return boltvm.Error(boltvm.AppchainNilBrokerCode, string(boltvm.AppchainNilBrokerMsg))
	}
	if strings.Contains(strings.ToLower(chainType), appchainMgr.FabricType) {
		fabBroker := &appchainMgr.FabricBroker{}
		if err := json.Unmarshal([]byte(broker), fabBroker); err != nil {
			return boltvm.Error(boltvm.AppchainIllegalFabricBrokerCode, fmt.Sprintf(string(boltvm.AppchainIllegalFabricBrokerMsg), string(broker), err.Error()))
		}
		if fabBroker.BrokerVersion == "" || fabBroker.ChaincodeID == "" || fabBroker.ChannelID == "" {
			return boltvm.Error(boltvm.AppchainIllegalFabricBrokerCode, fmt.Sprintf(string(boltvm.AppchainIllegalFabricBrokerMsg), string(broker), "fabric broker info can not be nil"))
		}
	}

	// 1.2 check name
	if chainName == "" {
		return boltvm.Error(boltvm.AppchainEmptyChainNameCode, string(boltvm.AppchainEmptyChainNameMsg))
	}
	if ok, chainID := am.isOccupiedName(chainName); ok {
		return boltvm.Error(boltvm.AppchainDuplicateChainNameCode, fmt.Sprintf(string(boltvm.AppchainDuplicateChainNameMsg), chainName, chainID))
	}

	// 1.3 check admin
	if !strings.Contains(adminAddrs, am.Caller()) {
		return boltvm.Error(boltvm.AppchainIncompleteAdminListCode, fmt.Sprintf(string(boltvm.AppchainIncompleteAdminListMsg), am.Caller()))
	}
	adminList := strings.Split(adminAddrs, ",")
	for _, addr := range adminList {
		if _, err := types.HexDecodeString(addr); err != nil {
			return boltvm.Error(boltvm.AppchainIllegalAdminAddrCode, fmt.Sprintf(string(boltvm.AppchainIllegalAdminAddrMsg), addr, err.Error()))
		}
		res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckOccupiedAccount", pb.String(addr))
		if !res.Ok {
			return boltvm.Error(boltvm.AppchainDuplicateAdminCode, fmt.Sprintf(string(boltvm.AppchainDuplicateAdminMsg), addr, string(res.Result)))
		}
	}

	// 1.4 check rule
	if res := CheckRuleAddress(am.Persister, masterRuleAddr, chainType); !res.Ok {
		return res
	}
	if !ruleMgr.IsDefault(masterRuleAddr, chainType) && strings.Trim(masterRuleUrl, " ") == "" {
		return boltvm.Error(boltvm.AppchainEmptyRuleUrlCode, string(boltvm.AppchainEmptyRuleUrlMsg))
	}

	// 2. pre store registration information (name, adminAddrs)
	am.occupyChainID(chainID)
	am.occupyChainName(chainName, chainID)
	if res := am.CrossInvoke(constant.RoleContractAddr.String(), "OccupyAccount", pb.String(adminAddrs), pb.String(string(AppchainAdmin))); !res.Ok {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("cross invoke OccupyAccount error: %s", string(res.Result))))
	}

	// 3. submit proposal
	registerInfo := &RegisterAppchainInfo{
		ChainInfo: &appchainMgr.Appchain{
			ID:        chainID,
			ChainName: chainName,
			ChainType: chainType,
			TrustRoot: trustRoot,
			Broker:    []byte(broker),
			Desc:      desc,
			Version:   0,
			Status:    governance.GovernanceAvailable,
		},
		MasterRule: &ruleMgr.Rule{
			Address: masterRuleAddr,
			RuleUrl: masterRuleUrl,
			Master:  true,
			Status:  governance.GovernanceAvailable,
		},
		AdminAddrs: adminAddrs,
	}
	registerInfoData, err := json.Marshal(registerInfo)
	if err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	}
	proposalRes := am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(event)),
		pb.String(string(AppchainMgr)),
		pb.String(chainID),
		pb.String(""), // no last status
		pb.String(reason),
		pb.Bytes(registerInfoData),
	)
	if !proposalRes.Ok {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(proposalRes.Result))))
	}

	am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(proposalRes.Result)))

	return getGovernanceRet(string(proposalRes.Result), nil)
}

// =========== UpdateAppchain updates appchain info.
func (am *AppchainManager) UpdateAppchain(id, name, desc string, trustRoot []byte, adminAddrs, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventUpdate

	// 1. check permission: PermissionSelf
	if err := am.checkPermission([]string{string(PermissionSelf)}, id, am.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.AppchainNoPermissionCode, fmt.Sprintf(string(boltvm.AppchainNoPermissionMsg), am.CurrentCaller(), err.Error()))
	}

	// 2. governance pre: check if exist and status
	chainInfoTmp, be := am.AppchainManager.GovernancePre(id, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	chainInfo := chainInfoTmp.(*appchainMgr.Appchain)

	// 3. check info
	// 3.1 check name
	if name == "" {
		return boltvm.Error(boltvm.AppchainEmptyChainNameCode, string(boltvm.AppchainEmptyChainNameMsg))
	}
	updateName := false
	if name != chainInfo.ChainName {
		if ok, chainID := am.isOccupiedName(name); ok {
			return boltvm.Error(boltvm.AppchainDuplicateChainNameCode, fmt.Sprintf(string(boltvm.AppchainDuplicateChainNameMsg), name, chainID))
		}
		updateName = true
	}

	// 3.2 check admins
	if !strings.Contains(adminAddrs, am.Caller()) {
		return boltvm.Error(boltvm.AppchainIncompleteAdminListCode, fmt.Sprintf(string(boltvm.AppchainIncompleteAdminListMsg), am.Caller()))
	}
	updateAdmin := false
	oldAdminList := am.getAdminAddrByChainId(id)
	oldAdminMap := make(map[string]struct{})
	for _, addr := range oldAdminList {
		oldAdminMap[addr] = struct{}{}
	}
	newAdminList := strings.Split(adminAddrs, ",")
	newAdminMap := make(map[string]struct{})
	for _, addr := range newAdminList {
		newAdminMap[addr] = struct{}{}
	}
	if len(oldAdminList) != len(newAdminMap) {
		updateAdmin = true
	} else {
		for _, oldAdmin := range oldAdminList {
			if _, ok := newAdminMap[oldAdmin]; !ok {
				updateAdmin = true
				break
			}
		}
	}
	if updateAdmin {
		for addr, _ := range newAdminMap {
			if _, ok := oldAdminMap[addr]; !ok {
				if _, err := types.HexDecodeString(addr); err != nil {
					return boltvm.Error(boltvm.AppchainIllegalAdminAddrCode, fmt.Sprintf(string(boltvm.AppchainIllegalAdminAddrMsg), addr, err.Error()))
				}
				res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckOccupiedAccount", pb.String(addr))
				if !res.Ok {
					return boltvm.Error(boltvm.AppchainDuplicateAdminCode, fmt.Sprintf(string(boltvm.AppchainDuplicateAdminMsg), addr, string(res.Result)))
				}
			}
		}
	}

	// 3.3 check trustroot
	updateTrustroot := false
	if !bytes.Equal(trustRoot, chainInfo.TrustRoot) {
		updateTrustroot = true
	}

	// 4. update info without proposal
	if !updateName && !updateAdmin && !updateTrustroot {
		updateInfo := &appchainMgr.Appchain{
			ID:        id,
			ChainName: name,
			TrustRoot: trustRoot,
			Desc:      desc,
		}
		ok, data := am.AppchainManager.Update(updateInfo)
		if !ok {
			return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("update appchain error: %s", string(data))))
		}

		if err := am.postAuditAppchainEvent(id); err != nil {
			return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("post audit appchain event error: %v", err)))
		}

		return getGovernanceRet("", nil)
	}

	// 5. check rule
	res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", pb.String(id))
	if !res.Ok {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("cross invoke GetMasterRule error: %s", string(res.Result))))
	}
	rule := &ruleMgr.Rule{}
	if err := json.Unmarshal(res.Result, rule); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("unmarshal rule error: %v", err)))
	}
	if rule.Status != governance.GovernanceAvailable {
		return boltvm.Error(boltvm.AppchainRuleUpdatingCode, fmt.Sprintf(string(boltvm.AppchainRuleUpdatingMsg), rule.Address, string(event), id))
	}

	// 6. pre store update information (name, adminAddrs)
	if updateName {
		am.occupyChainName(name, id)
	}
	if updateAdmin {
		if res := am.CrossInvoke(constant.RoleContractAddr.String(), "OccupyAccount", pb.String(adminAddrs), pb.String(string(AppchainAdmin))); !res.Ok {
			return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("cross invoke OccupyAccount error: %s", string(res.Result))))
		}
	}

	// 7. submit proposal
	updateAppchainInfo := UpdateAppchainInfo{
		Name: UpdateInfo{
			OldInfo: chainInfo.ChainName,
			NewInfo: name,
			IsEdit:  updateName,
		},
		Desc: UpdateInfo{
			OldInfo: chainInfo.Desc,
			NewInfo: desc,
			IsEdit:  !(desc == chainInfo.Desc),
		},
		TrustRoot: UpdateInfo{
			OldInfo: chainInfo.TrustRoot,
			NewInfo: trustRoot,
			IsEdit:  updateTrustroot,
		},
		AdminAddrs: UpdateInfo{
			OldInfo: strings.Join(oldAdminList, ","),
			NewInfo: strings.Join(newAdminList, ","),
			IsEdit:  updateAdmin,
		},
	}
	updateAppchainInfoData, err := json.Marshal(updateAppchainInfo)
	if err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	}
	res = am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(event)),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.String(reason),
		pb.Bytes(updateAppchainInfoData),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 8. change status
	if ok, data := am.AppchainManager.ChangeStatus(id, string(event), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), string(data)))
	}

	// 9. pause service
	if res1 := am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", pb.String(id)); !res1.Ok {
		return res1
	}

	am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	am.Logger().WithFields(logrus.Fields{
		"id": chainInfo.ID,
	}).Info(fmt.Sprintf("Appchain is doing event %s", event))

	if err := am.postAuditAppchainEvent(id); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("post audit appchain event error: %v", err)))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// =========== FreezeAppchain freezes appchain
func (am *AppchainManager) FreezeAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	res := am.basicGovernance(id, reason, []string{string(PermissionAdmin)}, governance.EventFreeze)
	if !res.Ok {
		return res
	}
	var gr *governance.GovernanceResult
	if err := json.Unmarshal(res.Result, &gr); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	}
	am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(gr.ProposalID))
	return res
}

// =========== ActivateAppchain activates frozen appchain
func (am *AppchainManager) ActivateAppchain(id, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventActivate

	// check rule
	res := am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(), "GetMasterRule", pb.String(id))
	if !res.Ok {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("cross invoke GetMasterRule error: %s", string(res.Result))))
	}
	rule := &ruleMgr.Rule{}
	if err := json.Unmarshal(res.Result, rule); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("unmarshal rule error: %v", err)))
	}
	if rule.Status != governance.GovernanceAvailable {
		return boltvm.Error(boltvm.AppchainRuleUpdatingCode, fmt.Sprintf(string(boltvm.AppchainRuleUpdatingMsg), rule.Address, string(event), id))
	}

	res = am.basicGovernance(id, reason, []string{string(PermissionSelf), string(PermissionAdmin)}, event)
	if !res.Ok {
		return res
	}
	var gr *governance.GovernanceResult
	if err := json.Unmarshal(res.Result, &gr); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	}
	am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(gr.ProposalID))
	return res
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
	var gr *governance.GovernanceResult
	if err := json.Unmarshal(governanceRes.Result, &gr); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	}

	am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(gr.ProposalID))

	return governanceRes
}

func (am *AppchainManager) basicGovernance(id, reason string, permissions []string, event governance.EventType) *boltvm.Response {
	// 1. check permission
	if err := am.checkPermission(permissions, id, am.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.AppchainNoPermissionCode, fmt.Sprintf(string(boltvm.AppchainNoPermissionMsg), am.CurrentCaller(), err.Error()))
	}

	// 2. governance pre: check if exist and status
	chain, be := am.AppchainManager.GovernancePre(id, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
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
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 4. change status
	if ok, data := am.AppchainManager.ChangeStatus(id, string(event), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	am.Logger().WithFields(logrus.Fields{
		"id": chainInfo.ID,
	}).Info(fmt.Sprintf("Appchain is doing event %s", event))

	if err := am.postAuditAppchainEvent(id); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("post audit appchain event error: %v", err)))
	}

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
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := am.checkPermission([]string{string(PermissionSpecific)}, id, am.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.AppchainNoPermissionCode, fmt.Sprintf(string(boltvm.AppchainNoPermissionMsg), am.CurrentCaller(), err.Error()))
	}

	// 2. governance pre: check if exist and status
	chain, be := am.AppchainManager.GovernancePre(id, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	chainInfo := chain.(*appchainMgr.Appchain)

	// 3. change status
	if chainInfo.Status == governance.GovernanceAvailable {
		if ok, data := am.AppchainManager.ChangeStatus(id, string(event), string(chainInfo.Status), nil); !ok {
			return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
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
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("marshal chain error: %v", err)))
	}

	if err := am.postAuditAppchainEvent(id); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("post audit appchain event error: %v", err)))
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
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := am.checkPermission([]string{string(PermissionSpecific)}, id, am.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.AppchainNoPermissionCode, fmt.Sprintf(string(boltvm.AppchainNoPermissionMsg), am.CurrentCaller(), err.Error()))
	}

	// 2. governance pre: check if exist and status
	_, be := am.AppchainManager.GovernancePre(id, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 3. change status
	if governance.GovernanceFrozen != governance.GovernanceStatus(lastStatus) {
		if ok, data := am.AppchainManager.ChangeStatus(id, string(event), lastStatus, nil); !ok {
			return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
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

	if err := am.postAuditAppchainEvent(id); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("post audit appchain event error: %v", err)))
	}

	return boltvm.Success(nil)
}

// ========================== Query interface ========================

// CountAvailableAppchains counts all available appchains
func (am *AppchainManager) CountAvailableAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return boltvm.ResponseWrapper(am.AppchainManager.CountAvailable(nil))
}

// CountAppchains counts all appchains including all status
func (am *AppchainManager) CountAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return boltvm.ResponseWrapper(am.AppchainManager.CountAll(nil))
}

// Appchains returns all appchains
func (am *AppchainManager) Appchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	chains, err := am.AppchainManager.All(nil)
	if err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	}

	if data, err := json.Marshal(chains.([]*appchainMgr.Appchain)); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}

}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) GetAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	chain, err := am.AppchainManager.QueryById(id, nil)
	if err != nil {
		return boltvm.Error(boltvm.AppchainNonexistentChainCode, fmt.Sprintf(string(boltvm.AppchainNonexistentChainMsg), id, err.Error()))
	}
	if data, err := json.Marshal(chain.(*appchainMgr.Appchain)); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) GetAppchainByName(name string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	id, err := am.AppchainManager.GetChainIdByName(name)
	if err != nil {
		return boltvm.Error(boltvm.AppchainNonexistentChainCode, fmt.Sprintf(string(boltvm.AppchainNonexistentChainMsg), name, err.Error()))
	}
	chain, err := am.AppchainManager.QueryById(id, nil)
	if err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), fmt.Sprintf("appchain name %s exist but appchain %s not exist: %v", name, id, err)))
	}
	if data, err := json.Marshal(chain.(*appchainMgr.Appchain)); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

func (am *AppchainManager) getAdminAddrByChainId(chainId string) []string {
	am.AppchainManager.Persister = am.Stub

	addrs := []string{}
	_ = am.GetObject(appchainMgr.AppAdminsChainKey(chainId), &addrs)
	return addrs
}

func (am *AppchainManager) GetAdminByChainId(chainId string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	addrs := am.getAdminAddrByChainId(chainId)
	addrsData, err := json.Marshal(addrs)
	if err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	}
	return boltvm.Success(addrsData)
}

func (am *AppchainManager) IsAvailable(chainID string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	chain, err := am.AppchainManager.QueryById(chainID, nil)
	if err != nil {
		return boltvm.Error(boltvm.AppchainNonexistentChainCode, fmt.Sprintf(string(boltvm.AppchainNonexistentChainMsg), chainID, err.Error()))
	}

	return boltvm.Success([]byte(strconv.FormatBool(chain.(*appchainMgr.Appchain).IsAvailable())))
}

func (am *AppchainManager) IsAvailableBitxhub(chainID string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	chain, err := am.AppchainManager.QueryById(chainID, nil)
	if err != nil {
		return boltvm.Error(boltvm.AppchainNonexistentChainCode, fmt.Sprintf(string(boltvm.AppchainNonexistentChainMsg), chainID, err.Error()))
	}

	if chain.(*appchainMgr.Appchain).IsAvailable() && chain.(*appchainMgr.Appchain).IsBitXHub() {
		return boltvm.Success([]byte(TRUE))
	} else {
		return boltvm.Success([]byte(FALSE))
	}
}

func (am *AppchainManager) GetBitXHubChainIDs() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	relayChainIdMap := orderedmap.New()
	_ = am.GetObject(appchainMgr.RelaychainType, relayChainIdMap)

	if data, err := json.Marshal(relayChainIdMap.Keys()); err != nil {
		return boltvm.Error(boltvm.AppchainInternalErrCode, fmt.Sprintf(string(boltvm.AppchainInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

func (am *AppchainManager) postAuditAppchainEvent(appchainID string) error {
	am.AppchainManager.Persister = am.Stub
	ok, chainData := am.Get(appchainMgr.AppchainKey(appchainID))
	if !ok {
		return fmt.Errorf("not found appchain %s", appchainID)
	}

	auditInfo := &pb.AuditRelatedObjInfo{
		AuditObj: chainData,
		RelatedChainIDList: map[string][]byte{
			appchainID: {},
		},
		RelatedNodeIDList: map[string][]byte{},
	}
	am.PostEvent(pb.Event_AUDIT_APPCHAIN, auditInfo)

	return nil
}
