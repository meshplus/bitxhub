package contracts

import (
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

const (
	AppchainIDTmp = "appchainTmp"
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
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			idTmp, err := am.getChainIdByAdminAddr(regulatorAddr)
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
func (am *AppchainManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	am.Logger().WithFields(logrus.Fields{
		"id": objId,
	}).Info("Appchain is manageing")
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
	if eventTyp != string(governance.EventRegister) {
		if ok, data := am.AppchainManager.ChangeStatus(objId, proposalResult, lastStatus, nil); !ok {
			return boltvm.Error(fmt.Sprintf("change status error:%s", string(data)))
		}
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			id, err := am.manageRegister(extra)
			if err != nil {
				return boltvm.Error(fmt.Sprintf("manage register error: %v", err))
			}
			return boltvm.Success([]byte(id))
		case string(governance.EventUpdate):
			if err := am.manageUpdate(objId, extra); err != nil {
				return boltvm.Error(fmt.Sprintf("manage update error: %v", err))
			}
		case string(governance.EventFreeze):
			return am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "PauseChainService", pb.String(objId))
		case string(governance.EventActivate):
			return am.CrossInvoke(constant.ServiceMgrContractAddr.Address().String(), "UnPauseChainService", pb.String(objId))
		}
	} else {
		switch eventTyp {
		case string(governance.EventRegister):
			registerInfo := &RegisterAppchainInfo{}
			if err := json.Unmarshal(extra, &registerInfo); err != nil {
				return boltvm.Error(fmt.Sprintf("unmarshal registerInfoData error: %v", err))
			}

			// free pre-stored registration information
			am.freeChainName(registerInfo.ChainInfo.ChainName)
			am.freeChainAdmin(strings.Split(registerInfo.AdminAddrs, ","))
		case string(governance.EventUpdate):
			updateInfo := &UpdateAppchainInfo{}
			if err := json.Unmarshal(extra, &updateInfo); err != nil {
				return boltvm.Error(fmt.Sprintf("unmarshal updateInfoData error: %v", err))
			}

			// free pre-stored update information
			if updateInfo.Name.IsEdit {
				// free new name
				am.freeChainName(updateInfo.Name.NewInfo.(string))
			}

			if updateInfo.AdminAddrs.IsEdit {
				// free new admin addrs
				am.freeChainAdmin(strings.Split(updateInfo.AdminAddrs.NewInfo.(string), ","))
				// occupy old admin addrs
				am.occupyChainAdmin(strings.Split(updateInfo.AdminAddrs.OldInfo.(string), ","), objId)
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

func (am *AppchainManager) manageRegister(registerInfoData []byte) (string, error) {
	am.AppchainManager.Persister = am.Stub

	registerInfo := &RegisterAppchainInfo{}
	if err := json.Unmarshal(registerInfoData, &registerInfo); err != nil {
		return "", fmt.Errorf("unmarshal registerInfoData error: %v", err)
	}

	// 1. register appchain
	ok, data := am.AppchainManager.Register(registerInfo.ChainInfo)
	if !ok {
		return "", fmt.Errorf("register error: %s", string(data))
	}
	appchainID := string(data)
	am.recordChainAdmins(appchainID, strings.Split(registerInfo.AdminAddrs, ","))
	// register relay chain
	if registerInfo.ChainInfo.IsBitXHub() {
		am.registerRelayChain(appchainID)
	}
	// record occupied name
	am.occupyChainName(registerInfo.ChainInfo.ChainName, appchainID)
	// record occupied admin addrs
	am.occupyChainAdmin(strings.Split(registerInfo.AdminAddrs, ","), appchainID)

	// 2. register appchain admin
	res := am.CrossInvoke(constant.RoleContractAddr.Address().String(),
		"UpdateAppchainAdmin",
		pb.String(appchainID),
		pb.String(registerInfo.AdminAddrs))
	if !res.Ok {
		return "", fmt.Errorf("cross invoke UpdateAppchainAdmin error: %s", string(res.Result))
	}

	// 3. register rule
	res = am.CrossInvoke(constant.RuleManagerContractAddr.Address().String(),
		"RegisterRuleFirst",
		pb.String(appchainID),
		pb.String(registerInfo.ChainInfo.ChainType),
		pb.String(registerInfo.MasterRule.Address),
		pb.String(registerInfo.MasterRule.RuleUrl),
	)
	if !res.Ok {
		return "", fmt.Errorf("cross invoke RegisterRuleFirst error: %s", string(res.Result))
	}

	// 4. register interchain
	res = am.CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", pb.String(appchainID))
	if !res.Ok {
		return "", fmt.Errorf("cross invoke interchain Register error: %s", string(res.Result))
	}

	return appchainID, nil
}

func (am *AppchainManager) manageUpdate(appchainId string, updateInfoData []byte) error {
	am.AppchainManager.Persister = am.Stub

	updateInfo := &UpdateAppchainInfo{}
	if err := json.Unmarshal(updateInfoData, &updateInfo); err != nil {
		return fmt.Errorf("unmarshal updateInfoData error: %v", err)
	}

	// update appchain
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
		am.freeChainAdmin(strings.Split(updateInfo.AdminAddrs.OldInfo.(string), ","))
		// occupy new admin addrs
		am.occupyChainAdmin(strings.Split(updateInfo.AdminAddrs.NewInfo.(string), ","), appchainId)
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

	return nil
}

func (am *AppchainManager) occupyChainName(name string, chainID string) {
	am.AppchainManager.Persister = am.Stub
	am.SetObject(appchainMgr.AppchainOccupyNameKey(name), chainID)
}

func (am *AppchainManager) freeChainName(name string) {
	am.AppchainManager.Persister = am.Stub
	am.Delete(appchainMgr.AppchainOccupyNameKey(name))
}

func (am *AppchainManager) occupyChainAdmin(addrs []string, chainID string) {
	am.AppchainManager.Persister = am.Stub
	for _, addr := range addrs {
		am.SetObject(appchainMgr.AppchainOccupyAdminKey(addr), chainID)
	}
}

func (am *AppchainManager) freeChainAdmin(addrs []string) {
	am.AppchainManager.Persister = am.Stub
	for _, addr := range addrs {
		am.Delete(appchainMgr.AppchainOccupyAdminKey(addr))
	}
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
	am.SetObject(appchainMgr.AppchainAdminKey(chainID), addrs)
}

// =========== RegisterAppchain registers appchain info, returns proposal id and error
func (am *AppchainManager) RegisterAppchain(chainName string, chainType string, trustRoot []byte, broker []byte, desc, masterRuleAddr, masterRuleUrl, adminAddrs, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventRegister

	// 1. check
	// 1.1 check broker
	if broker == nil || string(broker) == "" {
		return boltvm.Error("broker can not be nil")
	}
	if strings.Contains(strings.ToLower(chainType), appchainMgr.FabricType) {
		fabBroker := &appchainMgr.FabricBroker{}
		if err := json.Unmarshal(broker, fabBroker); err != nil {
			return boltvm.Error(fmt.Sprintf("unmarshal fabric broker error: %v", err))
		}
		if fabBroker.BrokerVersion == "" || fabBroker.ChaincodeID == "" || fabBroker.ChannelID == "" {
			return boltvm.Error("fabric broker info can not be nil")
		}
	}

	// 1.2 check name
	if chainName == "" {
		return boltvm.Error("chain name can not be an empty string")
	}
	if chainID, err := am.getChainIdByName(chainName); err == nil {
		return boltvm.Error(fmt.Sprintf("the name is already occupied by appchain %s", chainID))
	}

	// 1.3 check admin
	if !strings.Contains(adminAddrs, am.Caller()) {
		return boltvm.Error("the admin address list does not contain the current admin")
	}
	adminList := strings.Split(adminAddrs, ",")
	for _, addr := range adminList {
		if _, err := types.HexDecodeString(addr); err != nil {
			return boltvm.Error(fmt.Sprintf("illegal admin addr: %s", addr))
		}
		res := am.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleByAddr", pb.String(addr))
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("cross invoke GetRoleByAddr error: %s", string(res.Result)))
		}
		if string(NoRole) != string(res.Result) {
			return boltvm.Error(fmt.Sprintf("the admin add is already occupied by %s", string(res.Result)))
		}
		if chainID, err := am.getChainIdByAdminAddr(addr); err == nil {
			return boltvm.Error(fmt.Sprintf("the admin add is already occupied by appchain %s", chainID))
		}
	}

	// 1.4 check rule
	if err := CheckRuleAddress(am.Persister, masterRuleAddr, chainType); err != nil {
		return boltvm.Error(fmt.Sprintf("check rule address error: %v", err))
	}
	if !ruleMgr.IsDefault(masterRuleAddr, chainType) && strings.Trim(masterRuleUrl, " ") == "" {
		return boltvm.Error(fmt.Sprintf("Urls for custom rules cannot be empty strings"))
	}

	// 2. pre store registration information (name, adminAddrs)
	chainIDTmp := fmt.Sprintf("%s%s", AppchainIDTmp, am.GetTxHash().String())
	am.occupyChainName(chainName, chainIDTmp)
	am.occupyChainAdmin(adminList, chainIDTmp)

	// 3. submit proposal
	registerInfo := &RegisterAppchainInfo{
		ChainInfo: &appchainMgr.Appchain{
			ChainName: chainName,
			ChainType: chainType,
			TrustRoot: trustRoot,
			Broker:    broker,
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
		return boltvm.Error(err.Error())
	}
	proposalRes := am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(event)),
		pb.String(string(AppchainMgr)),
		pb.String(chainIDTmp), // chainID has not been generated
		pb.String(""),         // no last status
		pb.String(reason),
		pb.Bytes(registerInfoData),
	)
	if !proposalRes.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(proposalRes.Result)))
	}

	return getGovernanceRet(string(proposalRes.Result), nil)
}

// =========== UpdateAppchain updates appchain info.
// This is currently no need for voting governance.
func (am *AppchainManager) UpdateAppchain(id, name, desc string, trustRoot []byte, adminAddrs, reason string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	event := governance.EventUpdate

	// 1. check permission: PermissionSelf
	if err := am.checkPermission([]string{string(PermissionSelf)}, id, am.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. governance pre: check if exist and status
	chainInfoTmp, err := am.AppchainManager.GovernancePre(id, event, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}
	chainInfo := chainInfoTmp.(*appchainMgr.Appchain)

	// 3. check info
	// 3.1 check name
	if name == "" {
		return boltvm.Error("chain name can not be an empty string")
	}
	updateName := false
	if name != chainInfo.ChainName {
		if chainID, err := am.getChainIdByName(name); err == nil {
			return boltvm.Error(fmt.Sprintf("the name is already occupied by appchain %s", chainID))
		}
		updateName = true
	}

	// 3.2 check admins
	if !strings.Contains(adminAddrs, am.Caller()) {
		return boltvm.Error("the admin address list does not contain the current admin")
	}
	updateAdmin := false
	oldAdminList := am.getAdminAddrByChainId(id)
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
		for addr := range newAdminMap {
			if _, err := types.HexDecodeString(addr); err != nil {
				return boltvm.Error(fmt.Sprintf("illegal admin addr: %s", addr))
			}
			if chainID, err := am.getChainIdByAdminAddr(addr); err == nil {
				return boltvm.Error(fmt.Sprintf("the admin addr is already occupied by appchain %s", chainID))
			}
		}
	}

	// 3.3 check trustroot
	updateTrustroot := false
	if string(trustRoot) != string(chainInfo.TrustRoot) {
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
			return boltvm.Error(fmt.Sprintf("update appchain error: %s", string(data)))
		}

		return getGovernanceRet("", nil)
	}

	// 5. pre store update information (name, adminAddrs)
	if updateName {
		am.occupyChainName(name, id)
	}
	if updateAdmin {
		am.occupyChainAdmin(newAdminList, id)
	}

	// 6. submit proposal
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
		return boltvm.Error(err.Error())
	}
	res := am.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(event)),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.String(reason),
		pb.Bytes(updateAppchainInfoData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 7. change status
	if ok, data := am.AppchainManager.ChangeStatus(id, string(event), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	am.Logger().WithFields(logrus.Fields{
		"id": chainInfo.ID,
	}).Info(fmt.Sprintf("Appchain is doing event %s", event))

	return getGovernanceRet(string(res.Result), nil)
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

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) GetAppchainByName(name string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	id, err := am.getChainIdByName(name)
	if err != nil {
		return boltvm.Error(err.Error())
	}
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

func (am *AppchainManager) getChainIdByName(name string) (string, error) {
	am.AppchainManager.Persister = am.Stub
	chainId := ""
	ok := am.GetObject(appchainMgr.AppchainOccupyNameKey(name), &chainId)
	if !ok {
		return "", fmt.Errorf("the appchain of this name(%s) does not exist", name)
	}
	return chainId, nil
}

func (am *AppchainManager) getChainIdByAdminAddr(adminAddr string) (string, error) {
	am.AppchainManager.Persister = am.Stub
	chainId := ""
	ok := am.GetObject(appchainMgr.AppchainOccupyAdminKey(adminAddr), &chainId)
	if !ok {
		return "", fmt.Errorf("the appchain of this admin addr(%s) does not exist", adminAddr)
	}
	return chainId, nil
}

func (am *AppchainManager) getAdminAddrByChainId(chainId string) []string {
	am.AppchainManager.Persister = am.Stub

	addrs := []string{}
	_ = am.GetObject(appchainMgr.AppchainAdminKey(chainId), &addrs)
	return addrs
}

func responseWrapper(ok bool, data []byte) *boltvm.Response {
	if ok {
		return boltvm.Success(data)
	}
	return boltvm.Error(string(data))
}
