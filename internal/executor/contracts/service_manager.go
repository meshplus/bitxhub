package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	servicemgr "github.com/meshplus/bitxhub-core/service-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

type ServiceManager struct {
	boltvm.Stub
	servicemgr.ServiceManager
}

type UpdateServiceInfo struct {
	ServiceName UpdateInfo    `json:"service_name"`
	Intro       UpdateInfo    `json:"intro"`
	Details     UpdateInfo    `json:"details"`
	Permission  UpdateMapInfo `json:"permission"`
}

func (sm *ServiceManager) checkPermission(permissions []string, chainID string, regulatorAddr string, specificAddrsData []byte) error {
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			res := sm.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetAppchainAdmin", pb.String(chainID))
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
			res := sm.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin",
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

func (sm *ServiceManager) occupyServiceName(name string, chainServiceID string) {
	sm.ServiceManager.Persister = sm.Stub
	sm.SetObject(servicemgr.ServiceOccupyNameKey(name), chainServiceID)
}

func (sm *ServiceManager) freeServiceName(name string) {
	sm.ServiceManager.Persister = sm.Stub
	sm.Delete(servicemgr.ServiceOccupyNameKey(name))
}

func (sm *ServiceManager) isOccupiedName(name string) (bool, string) {
	sm.ServiceManager.Persister = sm.Stub
	chainServiceId := ""
	ok := sm.GetObject(servicemgr.ServiceOccupyNameKey(name), &chainServiceId)
	return ok, chainServiceId
}

// ========================== Governance interface ========================
// =========== Manage does some subsequent operations when the proposal is over
// extra: update - service info,
func (sm *ServiceManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := sm.checkPermission([]string{string(PermissionSpecific)}, objId, sm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 2. change status
	if ok, data := sm.ChangeStatus(objId, proposalResult, lastStatus, nil); !ok {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("change status error:%s", string(data))))
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			service, err := sm.ServiceManager.QueryById(objId, nil)
			if err != nil {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cannot get service by id %s", objId)))
			}
			serviceInfo := service.(*servicemgr.Service)
			sm.ServiceManager.Register(serviceInfo)

			res := sm.CrossInvoke(constant.InterchainContractAddr.Address().String(), "Register", pb.String(objId))
			if !res.Ok {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cross invoke register: %s", string(res.Result))))
			}

			chainID := strings.Split(objId, ":")[0]
			res = sm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", pb.String(chainID))
			if !res.Ok {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cross invoke is available error: %s", string(res.Result))))
			}
			if FALSE == string(res.Result) {
				if err := sm.pauseService(objId); err != nil {
					return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("chain is not available, pause service %s err: %v", objId, err)))
				}
			}
		case string(governance.EventUpdate):
			updateInfo := &UpdateServiceInfo{}
			if err := json.Unmarshal(extra, updateInfo); err != nil {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("unmarshal update data error:%v", err)))
			}

			updateService := &servicemgr.Service{
				ChainID:    strings.Split(objId, ":")[0],
				ServiceID:  strings.Split(objId, ":")[1],
				Name:       updateInfo.ServiceName.NewInfo.(string),
				Intro:      updateInfo.Intro.NewInfo.(string),
				Permission: updateInfo.Permission.NewInfo,
				Details:    updateInfo.Details.NewInfo.(string),
			}

			ok, data := sm.ServiceManager.Update(updateService)
			if !ok {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("update service error: %s", string(data))))
			}

			if updateInfo.ServiceName.IsEdit {
				sm.freeServiceName(updateInfo.ServiceName.OldInfo.(string))
			}
		case string(governance.EventLogout):
			if err := sm.clearService(objId); err != nil {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("clear service %s err: %v", objId, err)))
			}
		}
	} else {
		switch eventTyp {
		case string(governance.EventRegister):
			service, err := sm.ServiceManager.QueryById(objId, nil)
			if err != nil {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cannot get service by id %s", objId)))
			}
			serviceInfo := service.(*servicemgr.Service)
			sm.freeServiceName(serviceInfo.Name)
		case string(governance.EventUpdate):
			serviceUpdateInfo := &UpdateServiceInfo{}
			if err := json.Unmarshal(extra, serviceUpdateInfo); err != nil {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("unmarshal service error: %v", err)))
			}
			if serviceUpdateInfo.ServiceName.IsEdit {
				sm.freeServiceName(serviceUpdateInfo.ServiceName.NewInfo.(string))
			}
		case string(governance.EventLogout):
			chainID := strings.Split(objId, ":")[0]
			res := sm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", pb.String(chainID))
			if !res.Ok {
				return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cross invoke is available error: %s", string(res.Result))))
			}
			if FALSE == string(res.Result) {
				if err := sm.pauseService(objId); err != nil {
					return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("chain is not available, pause service %s err: %v", objId, err)))
				}
			}
		}
	}

	if err = sm.postAuditServiceEvent(objId); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
	}

	// record updated service in interchain contract cache
	if err = sm.postServiceEvent(objId); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
	}

	return boltvm.Success(nil)
}

// =========== RegisterService registers service info, returns proposal id and error
func (sm *ServiceManager) RegisterService(chainID, serviceID, name, typ, intro string, ordered uint64, permits, details, reason string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	event := governance.EventRegister

	// 1. check permission: PermissionSelf
	if err := sm.checkPermission([]string{string(PermissionSelf)}, chainID, sm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 2. governancePre: check status
	chainServiceID := fmt.Sprintf("%s:%s", chainID, serviceID)
	if _, be := sm.ServiceManager.GovernancePre(chainServiceID, event, nil); be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 3. check appchain
	if err := sm.checkAppchain(chainID); err != nil {
		return boltvm.Error(boltvm.ServiceUnavailableChainCode, fmt.Sprintf(string(boltvm.ServiceUnavailableChainMsg), chainID, chainServiceID, err.Error()))
	}

	// 4. check service info
	service, err := sm.ServiceManager.PackageServiceInfo(chainID, serviceID, name, typ, intro, ordered == 1, permits, details, sm.GetTxTimeStamp(), governance.GovernanceRegisting)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("get service info error: %v", err)))
	}
	if res := sm.checkServiceInfo(service, true); !res.Ok {
		return res
	}

	// 5. pre store registration information (name,)
	sm.occupyServiceName(name, chainServiceID)

	// 6. submit proposal
	res := sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(sm.Caller()),
		pb.String(string(event)),
		pb.String(string(ServiceMgr)),
		pb.String(chainServiceID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 7. register info
	sm.ServiceManager.RegisterPre(service)

	sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := sm.postAuditServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
	}

	// record updated service in interchain contract cache
	if err = sm.postServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post excutor serviceCache event error: %v", err)))
	}

	return getGovernanceRet(string(res.Result), []byte(chainServiceID))
}

// =========== UpdateService updates service info.
// updata permits does not need proposal
func (sm *ServiceManager) UpdateService(chainServiceID, name, intro, permits, details, reason string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	event := governance.EventUpdate

	// 1. governance pre: check if exist and status
	oldServiceInfo, be := sm.ServiceManager.GovernancePre(chainServiceID, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	oldService := oldServiceInfo.(*servicemgr.Service)

	// 2. check permission: PermissionSelf
	if err := sm.checkPermission([]string{string(PermissionSelf)}, oldService.ChainID, sm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 3. check service info
	newService, err := sm.ServiceManager.PackageServiceInfo(oldService.ChainID, oldService.ServiceID, name, string(oldService.Type), intro, oldService.Ordered, permits, details, oldService.CreateTime, oldService.Status)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("get service info error: %v", err)))
	}

	if res := sm.checkServiceInfo(newService, false); !res.Ok {
		return res
	}

	// 4. update permit or intro do not need proposal
	if newService.Name == oldService.Name &&
		newService.Details == oldService.Details {
		ok, data := sm.ServiceManager.Update(newService)
		if !ok {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("update service error: %s", string(data))))
		}

		if err := sm.postAuditServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
		}

		// record updated service in interchain contract cache
		if err = sm.postServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
		}

		return getGovernanceRet("", nil)
	}

	// 5. pre store registration information (name)
	if newService.Name != oldService.Name {
		sm.occupyServiceName(name, chainServiceID)
	}

	// 6. submit proposal
	updatePermission := false
	if len(oldService.Permission) != len(newService.Permission) {
		updatePermission = true
	} else {
		for permit, _ := range newService.Permission {
			if _, ok := oldService.Permission[permit]; !ok {
				updatePermission = true
				break
			}
		}
	}
	updateServiceInfo := &UpdateServiceInfo{
		ServiceName: UpdateInfo{
			OldInfo: oldService.Name,
			NewInfo: newService.Name,
			IsEdit:  oldService.Name != newService.Name,
		},
		Intro: UpdateInfo{
			OldInfo: oldService.Intro,
			NewInfo: newService.Intro,
			IsEdit:  oldService.Intro != newService.Intro,
		},
		Details: UpdateInfo{
			OldInfo: oldService.Details,
			NewInfo: newService.Details,
			IsEdit:  oldService.Details != newService.Details,
		},
		Permission: UpdateMapInfo{
			OldInfo: oldService.Permission,
			NewInfo: newService.Permission,
			IsEdit:  updatePermission,
		},
	}
	updateServiceInfoData, err := json.Marshal(updateServiceInfo)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal updateServiceInfo error: %v", err)))
	}
	res := sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(sm.Caller()),
		pb.String(string(event)),
		pb.String(string(ServiceMgr)),
		pb.String(chainServiceID),
		pb.String(string(oldService.Status)),
		pb.String(reason),
		pb.Bytes(updateServiceInfoData),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 7. change status
	if ok, data := sm.ServiceManager.ChangeStatus(chainServiceID, string(event), string(oldService.Status), nil); !ok {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := sm.postAuditServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
	}

	// record updated service in interchain contract cache
	if err = sm.postServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
	}
	return getGovernanceRet(string(res.Result), nil)
}

// =========== FreezeService freezes service
func (sm *ServiceManager) FreezeService(chainServiceID, reason string) *boltvm.Response {
	return sm.basicGovernance(chainServiceID, reason, []string{string(PermissionAdmin)}, governance.EventFreeze, nil)
}

// =========== ActivateService activates frozen service
func (sm *ServiceManager) ActivateService(chainServiceID, reason string) *boltvm.Response {
	return sm.basicGovernance(chainServiceID, reason, []string{string(PermissionSelf), string(PermissionAdmin)}, governance.EventActivate, nil)
}

// =========== LogoutService logouts service
func (sm *ServiceManager) LogoutService(chainServiceID, reason string) *boltvm.Response {
	return sm.basicGovernance(chainServiceID, reason, []string{string(PermissionSelf)}, governance.EventLogout, nil)
}

func (sm *ServiceManager) basicGovernance(chainServiceID, reason string, permissions []string, event governance.EventType, extra []byte) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	// 1. governance pre: check if exist and status
	serviceInfo, be := sm.ServiceManager.GovernancePre(chainServiceID, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	service := serviceInfo.(*servicemgr.Service)

	// 2. check permission
	if err := sm.checkPermission(permissions, service.ChainID, sm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 3. submit proposal
	res := sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(sm.Caller()),
		pb.String(string(event)),
		pb.String(string(ServiceMgr)),
		pb.String(chainServiceID),
		pb.String(string(service.Status)),
		pb.String(reason),
		pb.Bytes(extra),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 4. change status
	if ok, data := sm.ServiceManager.ChangeStatus(chainServiceID, string(event), string(service.Status), nil); !ok {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := sm.postAuditServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
	}

	// record updated service in interchain contract cache
	if err := sm.postServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
	}
	return getGovernanceRet(string(res.Result), nil)
}

// =========== PauseChainService pauses services by chainID
func (sm *ServiceManager) PauseChainService(chainID string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub

	// 1. check permission
	specificAddrs := []string{constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := sm.checkPermission([]string{string(PermissionSpecific)}, chainID, sm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 2. get services id
	idList, err := sm.ServiceManager.GetIDListByChainID(chainID)
	if err != nil {
		return getGovernanceRet("", nil)
	}

	sm.Logger().WithFields(logrus.Fields{
		"chainID":        chainID,
		"servicesIdList": idList,
	}).Info("pause chain services")

	// 3. pause services
	for _, chainServiceID := range idList {
		if err = sm.pauseService(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("pause service %s err: %v", chainServiceID, err)))
		}

		if err = sm.postAuditServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
		}
		// record updated service in interchain contract cache
		if err = sm.postServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
		}
	}

	return getGovernanceRet("", nil)
}

func (sm *ServiceManager) pauseService(chainServiceID string) error {
	event := governance.EventPause

	// 1. governance pre: check if exist and status
	if _, be := sm.ServiceManager.GovernancePre(chainServiceID, event, nil); be == nil {
		// 2. change status
		if ok, data := sm.ServiceManager.ChangeStatus(chainServiceID, string(event), "", nil); !ok {
			return fmt.Errorf("change status error: %s", string(data))
		}
	}

	// 3. lockProposal
	if res := sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "LockLowPriorityProposal",
		pb.String(chainServiceID),
		pb.String(string(governance.EventPause))); !res.Ok {
		return fmt.Errorf("cross invoke LockLowPriorityProposal error: %s", string(res.Result))
	}

	sm.Logger().WithFields(logrus.Fields{
		"chainServiceID": chainServiceID,
	}).Info("service pause")

	return nil
}

// =========== UnPauseChainService resumes suspended services by chain id
func (sm *ServiceManager) UnPauseChainService(chainID string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub

	// 1. check permission
	specificAddrs := []string{constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := sm.checkPermission([]string{string(PermissionSpecific)}, chainID, sm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 2. get services id
	idList, err := sm.ServiceManager.GetIDListByChainID(chainID)
	if err != nil {
		return getGovernanceRet("", nil)
	}

	// 3. unpause services
	for _, chainServiceID := range idList {
		if err := sm.unPauseService(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("pause service %s err: %v", chainServiceID, err)))
		}

		if err := sm.postAuditServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
		}
		// record updated service in interchain contract cache
		if err = sm.postServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
		}
	}

	return getGovernanceRet("", nil)
}

func (sm *ServiceManager) unPauseService(chainServiceID string) error {
	event := governance.EventUnpause
	sm.ServiceManager.Persister = sm.Stub

	// 1. governance pre: check if exist and status
	if _, be := sm.ServiceManager.GovernancePre(chainServiceID, event, nil); be == nil {
		// 2. change status
		if ok, data := sm.ServiceManager.ChangeStatus(chainServiceID, string(event), "", nil); !ok {
			return fmt.Errorf("change status error: %s", string(data))
		}
	}
	//service := serviceInfo.(*service_mgr.Service)

	sm.Logger().WithFields(logrus.Fields{
		"chainServiceID": chainServiceID,
	}).Info("service unpause")

	// 3. unlockProposal
	if res := sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UnLockLowPriorityProposal",
		pb.String(chainServiceID),
		pb.String(string(governance.EventUnpause))); !res.Ok {
		return fmt.Errorf("cross invoke UnLockLowPriorityProposal error: %s", string(res.Result))
	}

	return nil
}

// =========== ClearChainService clears services by chainID
func (sm *ServiceManager) ClearChainService(chainID string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub

	// 1. check permission
	specificAddrs := []string{constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := sm.checkPermission([]string{string(PermissionSpecific)}, chainID, sm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 2. get services id
	idList, err := sm.ServiceManager.GetIDListByChainID(chainID)
	if err != nil {
		return getGovernanceRet("", nil)
	}

	sm.Logger().WithFields(logrus.Fields{
		"chainID":        chainID,
		"servicesIdList": idList,
	}).Info("clear chain services")

	// 3. clear services
	for _, chainServiceID := range idList {
		if err := sm.clearService(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("clear service %s err: %v", chainServiceID, err)))
		}

		if err := sm.postAuditServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
		}
		// record updated service in interchain contract cache
		if err = sm.postServiceEvent(chainServiceID); err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
		}
	}

	return getGovernanceRet("", nil)
}

func (sm *ServiceManager) clearService(chainServiceID string) error {
	event := governance.EventCLear

	// 1. governance pre: check if exist and status
	if _, be := sm.ServiceManager.GovernancePre(chainServiceID, event, nil); be == nil {
		// 2. change status
		if ok, data := sm.ServiceManager.ChangeStatus(chainServiceID, string(event), "", nil); !ok {
			return fmt.Errorf("change status error: %s", string(data))
		}
	}

	// 3. clear proposal
	if res := sm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "EndObjProposal",
		pb.String(chainServiceID),
		pb.String(string(ClearReason)),
		pb.Bytes(nil)); !res.Ok {
		return fmt.Errorf("cross invoke EndObjProposal error: %s", string(res.Result))
	}

	return nil
}

func (sm *ServiceManager) EvaluateService(chainServiceID, desc string, score float64) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	if score < 0 || score > 5 {
		return boltvm.Error(boltvm.ServiceIllegalEvaluateScoreCode, fmt.Sprintf(string(boltvm.ServiceIllegalEvaluateScoreMsg), strconv.FormatFloat(score, 'f', 2, 64)))
	}

	// 1. get service
	service := &servicemgr.Service{}
	ok := sm.GetObject(servicemgr.ServiceKey(chainServiceID), service)
	if !ok {
		return boltvm.Error(boltvm.ServiceNonexistentServiceCode, fmt.Sprintf(string(boltvm.ServiceNonexistentServiceMsg), chainServiceID))
	}

	// 2. Check whether caller has evaluated
	if _, ok := service.EvaluationRecords[sm.Caller()]; ok {
		return boltvm.Error(boltvm.ServiceRepeatEvaluateCode, fmt.Sprintf(string(boltvm.ServiceRepeatEvaluateMsg), sm.Caller(), chainServiceID))
	}

	// 3. get evaluation record
	evaRec := &governance.EvaluationRecord{
		Addr:       sm.Caller(),
		Score:      score,
		Desc:       desc,
		CreateTime: sm.GetTxTimeStamp(),
	}

	// 4. store record
	num := float64(len(service.EvaluationRecords))
	service.Score = num/(num+1)*service.Score + 1/(num+1)*score
	service.EvaluationRecords[sm.Caller()] = evaRec
	sm.SetObject(servicemgr.ServiceKey(chainServiceID), *service)

	if err := sm.postAuditServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
	}
	// record updated service in interchain contract cache
	if err := sm.postServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
	}
	return getGovernanceRet("", nil)
}

func (sm *ServiceManager) RecordInvokeService(fullServiceID, fromFullServiceID string, result bool) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	toStrs := strings.Split(fullServiceID, ":")
	fromStrs := strings.Split(fromFullServiceID, ":")
	chainServiceID := fmt.Sprintf("%s:%s", toStrs[1], toStrs[2])
	fromChainServiceID := fmt.Sprintf("%s:%s", fromStrs[1], fromStrs[2])

	// 1. check permission: PermissionSpecific(InterchainContractAddr)
	specificAddrs := []string{constant.InterchainContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := sm.checkPermission([]string{string(PermissionSpecific)}, "", sm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.ServiceNoPermissionCode, fmt.Sprintf(string(boltvm.ServiceNoPermissionMsg), sm.CurrentCaller(), err.Error()))
	}

	// 2. get service
	service := &servicemgr.Service{}
	ok := sm.GetObject(servicemgr.ServiceKey(chainServiceID), service)
	if !ok {
		return boltvm.Error(boltvm.ServiceNonexistentServiceCode, fmt.Sprintf(string(boltvm.ServiceNonexistentServiceMsg), chainServiceID))
	}

	// 3. get invoke record
	var rec *governance.InvokeRecord

	rec, ok = service.InvokeRecords[fromChainServiceID]
	if ok {
		if result {
			rec.Succeed++
		} else {
			rec.Failure++
		}
	} else {
		rec = &governance.InvokeRecord{}
		rec.Addr = fromChainServiceID
		if result {
			rec.Succeed = 1
			rec.Failure = 0
		} else {
			rec.Succeed = 0
			rec.Failure = 1
		}
	}

	// 4. store record
	service.InvokeRecords[fromChainServiceID] = rec
	service.InvokeCount++
	num := float64(service.InvokeCount)
	if result {
		service.InvokeSuccessRate = num/(num+1)*service.InvokeSuccessRate + 1/(num+1)
	} else {
		if service.InvokeSuccessRate != 0 {
			service.InvokeSuccessRate = num/(num+1)*service.InvokeSuccessRate - 1/(num+1)
		}
	}

	sm.SetObject(servicemgr.ServiceKey(chainServiceID), *service)

	sm.Logger().WithFields(logrus.Fields{
		"chainServiceID":     chainServiceID,
		"fromChainServiceID": fromChainServiceID,
		"result":             result,
	}).Info("record invoke service")

	if err = sm.postAuditServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("post audit service event error: %v", err)))
	}

	// record updated service in interchain contract cache
	if err = sm.postServiceEvent(chainServiceID); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("invoke Interchain serviceCache error: %v", err)))
	}
	return boltvm.Success(nil)
}

// ========================== Query interface ========================
// GetServiceInfo returns Service info by service id
func (sm *ServiceManager) GetServiceInfo(id string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	service, err := sm.ServiceManager.QueryById(id, nil)
	if err != nil {
		return boltvm.Error(boltvm.ServiceNonexistentServiceCode, fmt.Sprintf(string(boltvm.ServiceNonexistentServiceMsg), id))
	}

	data, err := json.Marshal(service.(*servicemgr.Service))
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal service: %s", err.Error())))
	}

	return boltvm.Success(data)
}

func (sm *ServiceManager) GetServiceByName(name string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	id, err := sm.ServiceManager.GetServiceIDByName(name)
	if err != nil {
		return boltvm.Error(boltvm.ServiceNonexistentServiceCode, fmt.Sprintf(string(boltvm.ServiceNonexistentServiceMsg), name))
	}

	service, err := sm.ServiceManager.QueryById(id, nil)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("service name %s exist but service %s not exist: %v", name, id, err)))
	}

	data, err := json.Marshal(service.(*servicemgr.Service))
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("marshal service: %s", err.Error())))
	}

	return boltvm.Success(data)
}

// GetAllServices returns all service
func (sm *ServiceManager) GetAllServices() *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	services, err := sm.ServiceManager.All(nil)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	}
	if data, err := json.Marshal(services.([]*servicemgr.Service)); err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

// GetPermissionServices returns all permission dapps
func (sm *ServiceManager) GetPermissionServices(chainServiceId string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	services, err := sm.ServiceManager.All(nil)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	}

	var ret []*servicemgr.Service
	for _, s := range services.([]*servicemgr.Service) {
		if _, ok := s.Permission[chainServiceId]; !ok {
			ret = append(ret, s)
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

// GetServicesByAppchainID return services of an appchain
func (sm *ServiceManager) GetServicesByAppchainID(chainID string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	idList, err := sm.ServiceManager.GetIDListByChainID(chainID)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	}

	ret := make([]*servicemgr.Service, 0)
	for _, id := range idList {
		service, err := sm.ServiceManager.QueryById(id, nil)
		if err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cannot get service by id %s", id)))
		}
		ret = append(ret, service.(*servicemgr.Service))
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

// GetServicesByAppchainID return services of an appchain
func (sm *ServiceManager) GetServicesByType(typ string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	idList, err := sm.ServiceManager.GetIDListByType(typ)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	}

	ret := make([]*servicemgr.Service, 0)
	for _, id := range idList {
		service, err := sm.ServiceManager.QueryById(id, nil)
		if err != nil {
			return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cannot get service by id %s", id)))
		}
		ret = append(ret, service.(*servicemgr.Service))
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

func (sm *ServiceManager) IsAvailable(id string) *boltvm.Response {
	sm.ServiceManager.Persister = sm.Stub
	service, err := sm.ServiceManager.QueryById(id, nil)
	if err != nil {
		return boltvm.Error(boltvm.ServiceNonexistentServiceCode, fmt.Sprintf(string(boltvm.ServiceNonexistentServiceMsg), id))
	}

	return boltvm.Success([]byte(strconv.FormatBool(service.(*servicemgr.Service).IsAvailable())))
}

func (sm *ServiceManager) checkAppchain(chainID string) error {
	res := sm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "IsAvailable", pb.String(chainID))
	if !res.Ok {
		return fmt.Errorf("cross invoke IsAvailable error: %s", string(res.Result))
	}
	if "false" == string(res.Result) {
		return fmt.Errorf("the appchain is not available: %s", chainID)
	}

	return nil
}

func (sm *ServiceManager) checkServiceInfo(service *servicemgr.Service, isRegister bool) *boltvm.Response {
	// check name
	if service.Name == "" {
		return boltvm.Error(boltvm.ServiceEmptyNameCode, string(boltvm.ServiceEmptyNameMsg))
	}
	if ok, serviceID := sm.isOccupiedName(service.Name); ok {
		if isRegister {
			return boltvm.Error(boltvm.ServiceDuplicateNameCode, fmt.Sprintf(string(boltvm.ServiceDuplicateNameMsg), service.Name, serviceID))
		} else if serviceID != fmt.Sprintf("%s:%s", service.ChainID, service.ServiceID) {
			return boltvm.Error(boltvm.ServiceDuplicateNameCode, fmt.Sprintf(string(boltvm.ServiceDuplicateNameMsg), service.Name, serviceID))
		}
	}

	// check type
	if service.Type != servicemgr.ServiceCallContract &&
		service.Type != servicemgr.ServiceDepositCertificate &&
		service.Type != servicemgr.ServiceDataMigration {
		return boltvm.Error(boltvm.ServiceIllegalTypeCode, fmt.Sprintf(string(boltvm.ServiceIllegalTypeMsg), service.Type))
	}

	// check permission info
	for p, _ := range service.Permission {
		if berr := sm.checkPermissionService(p); berr != nil {
			return boltvm.Error(berr.Code, berr.Error())
		}
	}

	return boltvm.Success(nil)
}

func (sm *ServiceManager) checkPermissionService(fullServiceID string) *boltvm.BxhError {
	sm.ServiceManager.Persister = sm.Stub
	addrs := strings.Split(fullServiceID, ":")
	if len(addrs) != 3 {
		return boltvm.BError(boltvm.ServiceIllegalPermissionFormatCode, fmt.Sprintf(string(boltvm.ServiceIllegalPermissionFormatMsg), fullServiceID, "the ID does not contain three parts"))
	}

	if addrs[0] == "" {
		return boltvm.BError(boltvm.ServiceIllegalPermissionFormatCode, fmt.Sprintf(string(boltvm.ServiceIllegalPermissionFormatMsg), fullServiceID, "BitxhubID is empty"))
	} else {
		for _, a := range addrs[0] {
			if a < 48 || a > 57 {
				return boltvm.BError(boltvm.ServiceIllegalPermissionFormatCode, fmt.Sprintf(string(boltvm.ServiceIllegalPermissionFormatMsg), fullServiceID, "illegal BitxhubID"))
			}
		}
	}

	if addrs[1] == "" || addrs[2] == "" {
		return boltvm.BError(boltvm.ServiceIllegalPermissionFormatCode, fmt.Sprintf(string(boltvm.ServiceIllegalPermissionFormatMsg), fullServiceID, "AppchainID or ServiceID is empty"))
	}

	res := sm.CrossInvoke(constant.InterchainContractAddr.Address().String(), "GetBitXHubID")
	if !res.Ok {
		return boltvm.BError(boltvm.ServiceInternalErrCode, fmt.Sprintf(string(boltvm.ServiceInternalErrMsg), fmt.Sprintf("cross invoke GetBitXHubID error: %s", string(res.Result))))
	}
	if addrs[0] == string(res.Result) {
		service, err := sm.ServiceManager.QueryById(fmt.Sprintf("%s:%s", addrs[1], addrs[2]), nil)
		if err != nil {
			return boltvm.BError(boltvm.ServiceNonexistentPermissionServiceCode, fmt.Sprintf(string(boltvm.ServiceNonexistentPermissionServiceMsg), fullServiceID, string(res.Result), err.Error()))
		}
		serviceInfo := service.(*servicemgr.Service)
		if serviceInfo.Status == governance.GovernanceForbidden {
			return boltvm.BError(boltvm.ServiceLogoutedPermissionServiceCode, fmt.Sprintf(string(boltvm.ServiceLogoutedPermissionServiceMsg), fullServiceID))
		}
	}

	return nil
}

func (sm *ServiceManager) postAuditServiceEvent(chainServiceID string) error {
	sm.ServiceManager.Persister = sm.Stub
	ok, serviceData := sm.Get(servicemgr.ServiceKey(chainServiceID))
	if !ok {
		return fmt.Errorf("not found service %s", chainServiceID)
	}

	auditInfo := &pb.AuditRelatedObjInfo{
		AuditObj: serviceData,
		RelatedChainIDList: map[string][]byte{
			strings.Split(chainServiceID, ":")[0]: {},
		},
		RelatedNodeIDList: map[string][]byte{},
	}
	sm.PostEvent(pb.Event_AUDIT_SERVICE, auditInfo)

	return nil
}

// postServiceEvent record changed service in executor's interchain contract cache
func (sm *ServiceManager) postServiceEvent(chainServiceID string) error {
	sm.ServiceManager.Persister = sm.Stub
	service := &servicemgr.Service{}
	ok := sm.GetObject(servicemgr.ServiceKey(chainServiceID), service)
	if !ok {
		return fmt.Errorf("not found service %s", chainServiceID)
	}

	sm.PostEvent(pb.Event_SERVICE, service)
	return nil
}
