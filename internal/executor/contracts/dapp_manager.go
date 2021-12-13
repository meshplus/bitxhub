package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/looplab/fsm"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

const (
	DappPrefix               = "dapp"
	DappOwnerPrefix          = "owner-dapp"
	DappNamePrefix           = "name-dapp"
	DappOccupyNamePrefix     = "occupy-dapp-name"
	DappOccupyContractPrefix = "occupy-dapp-contract"
)

type DappManager struct {
	boltvm.Stub
}

type DappType string

var (
	DappTool        DappType = "tool"
	DappApplication DappType = "application"
	DappGame        DappType = "game"
	DappOthers      DappType = "others"
)

type Dapp struct {
	DappID       string              `json:"dapp_id"` // first owner address + num
	Name         string              `json:"name"`
	Type         DappType            `json:"type"`
	Desc         string              `json:"desc"`
	Url          string              `json:"url"`
	ContractAddr map[string]struct{} `json:"contract_addr"`
	Permission   map[string]struct{} `json:"permission"` // users which are not allowed to see the dapp
	OwnerAddr    string              `json:"owner_addr"`
	CreateTime   int64               `json:"create_time"`

	Score             float64                                 `json:"score"`
	EvaluationRecords map[string]*governance.EvaluationRecord `json:"evaluation_records"`
	TransferRecords   []*TransferRecord                       `json:"transfer_records"`

	Status governance.GovernanceStatus `json:"status"`
	FSM    *fsm.FSM                    `json:"fsm"`
}

type TransferRecord struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Reason     string `json:"reason"`
	Confirm    bool   `json:"confirm"`
	CreateTime int64  `json:"create_time"`
}

type UpdateDappInfo struct {
	DappName     UpdateInfo    `json:"dapp_name"`
	Desc         UpdateInfo    `json:"desc"`
	Url          UpdateInfo    `json:"url"`
	ContractAddr UpdateMapInfo `json:"contract_addr"`
	Permission   UpdateMapInfo `json:"permission"`
}

var dappStateMap = map[governance.EventType][]governance.GovernanceStatus{
	governance.EventRegister: {governance.GovernanceUnavailable},
	governance.EventUpdate:   {governance.GovernanceAvailable, governance.GovernanceFrozen, governance.GovernanceTransferring},
	governance.EventFreeze:   {governance.GovernanceAvailable, governance.GovernanceTransferring},
	governance.EventActivate: {governance.GovernanceFrozen},
	governance.EventTransfer: {governance.GovernanceAvailable},
}

var dappAvailableMap = map[governance.GovernanceStatus]struct{}{
	governance.GovernanceAvailable:    {},
	governance.GovernanceFreezing:     {},
	governance.GovernanceTransferring: {},
}

func (d *Dapp) IsAvailable() bool {
	if _, ok := dappAvailableMap[d.Status]; ok {
		return true
	} else {
		return false
	}
}

func (d *Dapp) setFSM(lastStatus governance.GovernanceStatus) {
	d.FSM = fsm.NewFSM(
		string(d.Status),
		fsm.Events{
			// register 3
			{Name: string(governance.EventRegister), Src: []string{string(governance.GovernanceUnavailable)}, Dst: string(governance.GovernanceRegisting)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceRegisting)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceRegisting)}, Dst: string(lastStatus)},

			// update 1
			{Name: string(governance.EventUpdate), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceTransferring), string(governance.GovernanceFrozen)}, Dst: string(governance.GovernanceUpdating)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceFrozen)},

			// freeze 2
			{Name: string(governance.EventFreeze), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceTransferring)}, Dst: string(governance.GovernanceFreezing)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceFreezing)}, Dst: string(governance.GovernanceFrozen)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceFreezing)}, Dst: string(lastStatus)},

			// activate 1
			{Name: string(governance.EventActivate), Src: []string{string(governance.GovernanceFrozen), string(governance.GovernanceFreezing)}, Dst: string(governance.GovernanceActivating)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceActivating)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceActivating)}, Dst: string(lastStatus)},

			// transfer 1
			{Name: string(governance.EventTransfer), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceFreezing), string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceTransferring)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceTransferring)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceTransferring)}, Dst: string(lastStatus)},
		},
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) { d.Status = governance.GovernanceStatus(d.FSM.Current()) },
		},
	)
}

// GovernancePre checks if the dapp can do the event. (only check, not modify infomation)
func (dm *DappManager) governancePre(dappID string, event governance.EventType) (*Dapp, *boltvm.BxhError) {
	dapp := &Dapp{}
	if ok := dm.GetObject(DappKey(dappID), dapp); !ok {
		if event == governance.EventRegister {
			return nil, nil
		} else {
			return nil, boltvm.BError(boltvm.DappNonexistentDappCode, fmt.Sprintf(string(boltvm.DappNonexistentDappMsg), dappID))
		}
	}

	for _, s := range dappStateMap[event] {
		if dapp.Status == s {
			return dapp, nil
		}
	}

	return nil, boltvm.BError(boltvm.DappStatusErrorCode, fmt.Sprintf(string(boltvm.DappStatusErrorMsg), dappID, string(dapp.Status), string(event)))
}

func (dm *DappManager) changeStatus(dappID, trigger, lastStatus string) (bool, []byte) {
	dapp := &Dapp{}
	if ok := dm.GetObject(DappKey(dappID), dapp); !ok {
		return false, []byte("this dapp does not exist")
	}

	dapp.setFSM(governance.GovernanceStatus(lastStatus))
	err := dapp.FSM.Event(trigger)
	if err != nil {
		return false, []byte(fmt.Sprintf("change status error: %v", err))
	}

	dm.SetObject(DappKey(dappID), *dapp)
	return true, nil
}

func (dm *DappManager) checkPermission(permissions []string, ownerAddr string, regulatorAddr string, specificAddrsData []byte) error {
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			if ownerAddr == regulatorAddr {
				return nil
			}
		case string(PermissionAdmin):
			if ownerAddr != regulatorAddr {
				res := dm.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin",
					pb.String(regulatorAddr),
					pb.String(string(GovernanceAdmin)))
				if !res.Ok {
					return fmt.Errorf("cross invoke IsAvailableGovernanceAdmin error:%s", string(res.Result))
				}
				if "true" == string(res.Result) {
					return nil
				}
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

func (dm *DappManager) occupyDappName(name string, dappID string) {
	dm.SetObject(DappOccupyNameKey(name), dappID)
}

func (dm *DappManager) freeDappName(name string) {
	dm.Delete(DappOccupyNameKey(name))
}

func (dm *DappManager) isOccupiedName(name string) (bool, string) {
	dappId := ""
	ok := dm.GetObject(DappOccupyNameKey(name), &dappId)
	return ok, dappId
}

func (dm *DappManager) occupyContractAddr(addrs map[string]struct{}, dappID string) {
	for addr := range addrs {
		dm.SetObject(DappOccupyContractKey(addr), dappID)
	}
}

func (dm *DappManager) freeContractAddr(addrs map[string]struct{}) {
	for addr := range addrs {
		dm.Delete(DappOccupyContractKey(addr))
	}
}

func (dm *DappManager) isOccupiedContractAddr(contractAddr string) (bool, string) {
	dappID := ""
	ok := dm.GetObject(DappOccupyContractKey(contractAddr), &dappID)
	return ok, dappID
}

func (dm *DappManager) addToOwner(ownerAddr, dappID string) {
	dappMap := make(map[string]struct{})
	_ = dm.GetObject(OwnerKey(ownerAddr), &dappMap)
	dappMap[dappID] = struct{}{}
	dm.SetObject(OwnerKey(ownerAddr), dappMap)
}

func (dm *DappManager) registerPre(dapp *Dapp) {
	dm.SetObject(DappKey(dapp.DappID), *dapp)
}

func (dm *DappManager) register(dapp *Dapp) {
	dm.SetObject(DappKey(dapp.DappID), *dapp)
	dm.SetObject(DappNameKey(dapp.Name), dapp.DappID)
	dm.addToOwner(dapp.OwnerAddr, dapp.DappID)
	dm.Logger().WithFields(logrus.Fields{
		"id":   dapp.DappID,
		"name": dapp.Name,
	}).Info("Dapp is registering")
}

func (dm *DappManager) update(updataInfo *Dapp) error {
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(updataInfo.DappID), dapp)
	if !ok {
		return fmt.Errorf("the dapp is not exist")
	}

	oldName := dapp.Name
	dapp.Name = updataInfo.Name
	dapp.Desc = updataInfo.Desc
	dapp.Permission = updataInfo.Permission
	dapp.ContractAddr = updataInfo.ContractAddr
	dapp.Url = updataInfo.Url
	dm.SetObject(DappKey(dapp.DappID), *dapp)
	if oldName != dapp.Name {
		dm.Delete(DappNameKey(oldName))
		dm.SetObject(DappNameKey(oldName), dapp.DappID)
	}
	dm.Logger().WithFields(logrus.Fields{
		"id":   dapp.DappID,
		"name": dapp.Name,
	}).Info("Dapp is updating")
	return nil
}

func (dm *DappManager) transfer(dapp *Dapp, transRec *TransferRecord) {
	dapp.TransferRecords = append(dapp.TransferRecords, transRec)
	dapp.OwnerAddr = transRec.To
	dm.SetObject(DappKey(dapp.DappID), *dapp)
	dm.addToOwner(dapp.OwnerAddr, dapp.DappID)

	dm.Logger().WithFields(logrus.Fields{
		"id":   dapp.DappID,
		"name": dapp.Name,
	}).Info("Dapp is registering")
}

// ========================== Governance interface ========================
// =========== Manage does some subsequent operations when the proposal is over
// extra: update - dapp info, transfer - transfer record, register - dapp
func (dm *DappManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := dm.checkPermission([]string{string(PermissionSpecific)}, objId, dm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.DappNoPermissionCode, fmt.Sprintf(string(boltvm.DappNoPermissionMsg), dm.CurrentCaller(), err.Error()))
	}

	// 2. change status
	if ok, data := dm.changeStatus(objId, proposalResult, lastStatus); !ok {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("change status error:%s", string(data))))
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			dapp := &Dapp{}
			ok := dm.GetObject(DappKey(objId), dapp)
			if !ok {
				return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), "the dapp is not exist"))
			}
			dapp.CreateTime = dm.GetTxTimeStamp()
			dm.register(dapp)
		case string(governance.EventUpdate):
			updateInfo := &UpdateDappInfo{}
			err := json.Unmarshal(extra, updateInfo)
			if err != nil {
				return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("unmarshal updateInfo err: %v", err)))
			}
			if updateInfo.DappName.IsEdit {
				dm.freeDappName(updateInfo.DappName.OldInfo.(string))
			}
			if updateInfo.ContractAddr.IsEdit {
				dm.freeContractAddr(updateInfo.ContractAddr.OldInfo)
				dm.occupyContractAddr(updateInfo.ContractAddr.NewInfo, objId)
			}
			if err := dm.update(&Dapp{
				DappID:       objId,
				Name:         updateInfo.DappName.NewInfo.(string),
				Desc:         updateInfo.Desc.NewInfo.(string),
				Url:          updateInfo.Url.NewInfo.(string),
				ContractAddr: updateInfo.ContractAddr.NewInfo,
				Permission:   updateInfo.Permission.NewInfo,
			}); err != nil {
				return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("update error: %v", err)))
			}
		case string(governance.EventTransfer):
			transRec := &TransferRecord{}
			if err := json.Unmarshal(extra, transRec); err != nil {
				return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("unmarshal update data error:%v", err)))
			}
			transRec.CreateTime = dm.GetTxTimeStamp()
			dapp := &Dapp{}
			ok := dm.GetObject(DappKey(objId), dapp)
			if !ok {
				return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("the dapp is not exist")))
			}
			dm.transfer(dapp, transRec)
		}
	} else {
		switch eventTyp {
		case string(governance.EventRegister):
			dapp := &Dapp{}
			ok := dm.GetObject(DappKey(objId), dapp)
			if !ok {
				return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), "the dapp is not exist"))
			}
			dm.freeDappName(dapp.Name)
			dm.freeContractAddr(dapp.ContractAddr)
		case string(governance.EventUpdate):
			updateInfo := &UpdateDappInfo{}
			err := json.Unmarshal(extra, updateInfo)
			if err != nil {
				return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("unmarshal updateInfo err: %v", err)))
			}
			if updateInfo.DappName.IsEdit {
				dm.freeDappName(updateInfo.DappName.NewInfo.(string))
			}
			if updateInfo.ContractAddr.IsEdit {
				dm.freeContractAddr(updateInfo.ContractAddr.NewInfo)
				dm.occupyContractAddr(updateInfo.ContractAddr.OldInfo, objId)
			}
		}
	}

	if err := dm.postAuditDappEvent(objId); err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("post audit dapp event error: %v", err)))
	}
	return boltvm.Success(nil)
}

// =========== RegisterDapp registers dapp info, returns proposal id and error
func (dm *DappManager) RegisterDapp(name, typ, desc, url, conAddrs, permits, reason string) *boltvm.Response {
	event := governance.EventRegister

	// 1. get dapp info
	dapp, err := dm.packageDappInfo("", name, typ, desc, url, conAddrs, permits, dm.Caller(), 0, dm.GetTxTimeStamp(), make(map[string]*governance.EvaluationRecord), nil, governance.GovernanceRegisting)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("get dapp info error: %v", err)))
	}

	// 2. check dapp info
	if res := dm.checkDappInfo(dapp, true); !res.Ok {
		return res
	}

	// 3. governancePre: check status
	if _, be := dm.governancePre(dapp.DappID, event); be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 4. pre store dapp contract addr
	dm.occupyDappName(dapp.Name, dapp.DappID)
	dm.occupyContractAddr(dapp.ContractAddr, dapp.DappID)

	// 5. submit proposal
	res := dm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(dm.Caller()),
		pb.String(string(event)),
		pb.String(string(DappMgr)),
		pb.String(dapp.DappID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 6. register info
	dm.registerPre(dapp)

	dm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := dm.postAuditDappEvent(dapp.DappID); err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("post audit dapp event error: %v", err)))
	}

	return getGovernanceRet(string(res.Result), []byte(dapp.DappID))
}

// =========== UpdateDapp updates dapp info.
func (dm *DappManager) UpdateDapp(id, name, desc, url, conAddrs, permits, reason string) *boltvm.Response {
	event := governance.EventUpdate

	// 1. governance pre: check if exist and status
	oldDapp, be := dm.governancePre(id, event)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 2. check permission: PermissionSelf
	if err := dm.checkPermission([]string{string(PermissionSelf)}, oldDapp.OwnerAddr, dm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.DappNoPermissionCode, fmt.Sprintf(string(boltvm.DappNoPermissionMsg), dm.CurrentCaller(), err.Error()))
	}

	// 3. get info
	newDapp, err := dm.packageDappInfo(id, name, string(oldDapp.Type), desc, url, conAddrs, permits, oldDapp.OwnerAddr, oldDapp.Score, oldDapp.CreateTime, oldDapp.EvaluationRecords, oldDapp.TransferRecords, oldDapp.Status)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("get dapp info error: %v", err)))
	}
	// 4. check info
	if res := dm.checkDappInfo(newDapp, false); !res.Ok {
		return res
	}

	// update desc do not need proposal
	updateName := newDapp.Name != oldDapp.Name
	updateUrl := newDapp.Url != oldDapp.Url
	updateContract := false
	if len(newDapp.ContractAddr) != len(oldDapp.ContractAddr) {
		updateContract = true
	} else {
		for addr := range oldDapp.ContractAddr {
			if _, ok := newDapp.ContractAddr[addr]; !ok {
				updateContract = true
				break
			}
		}
	}
	updatePermission := false
	if len(newDapp.Permission) != len(oldDapp.Permission) {
		updatePermission = true
	} else {
		for addr := range oldDapp.Permission {
			if _, ok := newDapp.Permission[addr]; !ok {
				updatePermission = true
				break
			}
		}
	}
	if !updateName && !updateUrl && !updateContract && !updatePermission {
		if err := dm.update(newDapp); err != nil {
			return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("update error: %v", err)))
		}
		if err := dm.postAuditDappEvent(id); err != nil {
			return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("post audit dapp event error: %v", err)))
		}
		return getGovernanceRet("", nil)
	}

	// 5. pre store dapp contract addr
	if updateName {
		dm.occupyDappName(newDapp.Name, id)
	}
	if updateContract {
		dm.occupyContractAddr(newDapp.ContractAddr, id)
	}

	// 6. submit proposal
	updateDappInfo := UpdateDappInfo{
		DappName: UpdateInfo{
			OldInfo: oldDapp.Name,
			NewInfo: newDapp.Name,
			IsEdit:  updateName,
		},
		Desc: UpdateInfo{
			OldInfo: oldDapp.Desc,
			NewInfo: newDapp.Desc,
			IsEdit:  oldDapp.Desc != newDapp.Desc,
		},
		Url: UpdateInfo{
			OldInfo: oldDapp.Url,
			NewInfo: newDapp.Url,
			IsEdit:  updateUrl,
		},
		ContractAddr: UpdateMapInfo{
			OldInfo: oldDapp.ContractAddr,
			NewInfo: newDapp.ContractAddr,
			IsEdit:  updateContract,
		},
		Permission: UpdateMapInfo{
			OldInfo: oldDapp.Permission,
			NewInfo: newDapp.Permission,
			IsEdit:  updatePermission,
		},
	}
	updateDappData, err := json.Marshal(updateDappInfo)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("marshal updateDappInfo error: %v", err)))
	}
	res := dm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(dm.Caller()),
		pb.String(string(event)),
		pb.String(string(DappMgr)),
		pb.String(id),
		pb.String(string(oldDapp.Status)),
		pb.String(reason),
		pb.Bytes(updateDappData),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 7. change status
	if ok, data := dm.changeStatus(id, string(event), string(oldDapp.Status)); !ok {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	dm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := dm.postAuditDappEvent(id); err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("post audit dapp event error: %v", err)))
	}
	return getGovernanceRet(string(res.Result), nil)
}

// =========== FreezeDapp freezes dapp
func (dm *DappManager) FreezeDapp(id, reason string) *boltvm.Response {
	return dm.basicGovernance(id, reason, []string{string(PermissionAdmin)}, governance.EventFreeze, nil)
}

// =========== ActivateDapp activates frozen dapp
func (dm *DappManager) ActivateDapp(id, reason string) *boltvm.Response {
	return dm.basicGovernance(id, reason, []string{string(PermissionSelf), string(PermissionAdmin)}, governance.EventActivate, nil)
}

// =========== TransferDapp transfers dapp
func (dm *DappManager) TransferDapp(id, newOwnerAddr, reason string) *boltvm.Response {
	_, err := types.HexDecodeString(newOwnerAddr)
	if err != nil {
		return boltvm.Error(boltvm.DappIllegalTransferAddrCode, fmt.Sprintf(string(boltvm.DappIllegalTransferAddrMsg), newOwnerAddr, err.Error()))
	}

	if newOwnerAddr == dm.Caller() {
		return boltvm.Error(boltvm.DappTransferToSelfCode, fmt.Sprintf(string(boltvm.DappTransferToSelfMsg), newOwnerAddr))
	}

	transRec := &TransferRecord{
		From:    dm.Caller(),
		To:      newOwnerAddr,
		Reason:  reason,
		Confirm: false,
	}
	extra, err := json.Marshal(transRec)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("marshal extra error: %v", err)))
	}
	return dm.basicGovernance(id, reason, []string{string(PermissionSelf)}, governance.EventTransfer, extra)
}

func (dm *DappManager) basicGovernance(id, reason string, permissions []string, event governance.EventType, extra []byte) *boltvm.Response {
	// 1. governance pre: check if exist and status
	dapp, be := dm.governancePre(id, event)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 2. check permission
	if err := dm.checkPermission(permissions, dapp.OwnerAddr, dm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.DappNoPermissionCode, fmt.Sprintf(string(boltvm.DappNoPermissionMsg), dm.CurrentCaller(), err.Error()))
	}

	// 3. submit proposal
	res := dm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(dm.Caller()),
		pb.String(string(event)),
		pb.String(string(DappMgr)),
		pb.String(id),
		pb.String(string(dapp.Status)),
		pb.String(reason),
		pb.Bytes(extra),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 4. change status
	if ok, data := dm.changeStatus(id, string(event), string(dapp.Status)); !ok {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	dm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := dm.postAuditDappEvent(id); err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("post audit dapp event error: %v", err)))
	}
	return getGovernanceRet(string(res.Result), nil)
}

func (dm *DappManager) ConfirmTransfer(id string) *boltvm.Response {
	// 1. get dapp
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return boltvm.Error(boltvm.DappNonexistentDappCode, fmt.Sprintf(string(boltvm.DappNonexistentDappMsg), id))
	}

	// 2. check permission
	if err := dm.checkPermission([]string{string(PermissionSelf)}, dapp.OwnerAddr, dm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.DappNoPermissionCode, fmt.Sprintf(string(boltvm.DappNoPermissionMsg), dm.CurrentCaller(), err.Error()))
	}

	// 3. confirm
	if len(dapp.TransferRecords) != 0 {
		if !dapp.TransferRecords[len(dapp.TransferRecords)-1].Confirm {
			dapp.TransferRecords[len(dapp.TransferRecords)-1].Confirm = true
			dm.SetObject(DappKey(id), *dapp)
		}
	}

	if err := dm.postAuditDappEvent(id); err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("post audit dapp event error: %v", err)))
	}
	return getGovernanceRet("", nil)
}

func (dm *DappManager) EvaluateDapp(id, desc string, score float64) *boltvm.Response {
	if score < 0 || score > 5 {
		return boltvm.Error(boltvm.DappIllegalEvaluateScoreCode, fmt.Sprintf(string(boltvm.DappIllegalEvaluateScoreMsg), strconv.FormatFloat(score, 'f', 2, 64)))
	}

	// 1. get dapp
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return boltvm.Error(boltvm.DappNonexistentDappCode, fmt.Sprintf(string(boltvm.DappNonexistentDappMsg), id))
	}

	// 2. Check whether caller has evaluated
	if _, ok := dapp.EvaluationRecords[dm.Caller()]; ok {
		return boltvm.Error(boltvm.DappRepeatEvaluateCode, fmt.Sprintf(string(boltvm.DappRepeatEvaluateMsg), dm.Caller(), id))
	}

	// 3. get evaluation record
	evaRec := &governance.EvaluationRecord{
		Addr:       dm.Caller(),
		Score:      score,
		Desc:       desc,
		CreateTime: dm.GetTxTimeStamp(),
	}
	dm.Logger().WithFields(logrus.Fields{
		"id":     dapp.DappID,
		"evaRec": evaRec,
	}).Debug("evaluate dapp")

	// 4. store record
	num := float64(len(dapp.EvaluationRecords))
	dapp.Score = num/(num+1)*dapp.Score + 1/(num+1)*score
	dapp.EvaluationRecords[dm.Caller()] = evaRec
	dm.SetObject(DappKey(id), *dapp)

	if err := dm.postAuditDappEvent(id); err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("post audit dapp event error: %v", err)))
	}
	return getGovernanceRet("", nil)
}

// ========================== Query interface ========================
// GetDapp returns dapp info by dapp id
func (dm *DappManager) GetDapp(id string) *boltvm.Response {
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return boltvm.Error(boltvm.DappNonexistentDappCode, fmt.Sprintf(string(boltvm.DappNonexistentDappMsg), id))
	}

	data, err := json.Marshal(dapp)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}

	return boltvm.Success(data)
}

func (dm *DappManager) GetDappByName(name string) *boltvm.Response {
	id := ""
	ok := dm.GetObject(DappNameKey(name), &id)
	if !ok {
		return boltvm.Error(boltvm.DappNonexistentDappCode, fmt.Sprintf(string(boltvm.DappNonexistentDappMsg), name))
	}

	dapp := &Dapp{}
	if ok := dm.GetObject(DappKey(id), dapp); !ok {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), fmt.Sprintf("dapp name %s exist but dapp %s not exist", name, id)))
	}

	data, err := json.Marshal(dapp)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}

	return boltvm.Success(data)
}

// GetAllDapps returns all dapps
func (dm *DappManager) GetAllDapps() *boltvm.Response {
	ret, err := dm.getAll()
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

func (dm *DappManager) getAll() ([]*Dapp, error) {
	ret := make([]*Dapp, 0)
	ok, value := dm.Query(DappPrefix)
	if ok {
		for _, data := range value {
			dapp := &Dapp{}
			if err := json.Unmarshal(data, dapp); err != nil {
				return nil, fmt.Errorf("unmarshal dapp error: %v", err)
			}
			ret = append(ret, dapp)
		}
	}

	sort.Sort(Dapps(ret))
	return ret, nil
}

// GetPermissionDapps returns all the DApp that the caller is allowed to call
func (dm *DappManager) GetPermissionDapps(caller string) *boltvm.Response {
	var ret []*Dapp
	all, err := dm.getAll()
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}
	for _, d := range all {
		if _, ok := d.Permission[caller]; !ok {
			ret = append(ret, d)
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

// GetPermissionAvailableDapps returns the available DApp that the caller is allowed to call
func (dm *DappManager) GetPermissionAvailableDapps(caller string) *boltvm.Response {
	var ret []*Dapp
	all, err := dm.getAll()
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}
	for _, d := range all {
		if _, ok := d.Permission[caller]; !ok {
			if d.IsAvailable() {
				ret = append(ret, d)
			}
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

// get dApps by owner addr, including dApps a person currently owns and the dApps they once owned
func (dm *DappManager) GetDappsByOwner(ownerAddr string) *boltvm.Response {
	ret, err := dm.getOwnerAll(ownerAddr)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(boltvm.DappInternalErrCode, fmt.Sprintf(string(boltvm.DappInternalErrMsg), err.Error()))
	}
	return boltvm.Success(data)
}

func (dm *DappManager) getOwnerAll(ownerAddr string) ([]*Dapp, error) {
	ret := make([]*Dapp, 0)
	dappTransferred := make([]*Dapp, 0)

	var dappMap map[string]struct{}
	ok := dm.GetObject(OwnerKey(ownerAddr), &dappMap)
	if ok {
		for dappID, _ := range dappMap {
			dapp := &Dapp{}
			if ok := dm.GetObject(DappKey(dappID), dapp); !ok {
				return nil, fmt.Errorf("the dapp(%s) is not exist", dappID)
			}
			if dapp.OwnerAddr != ownerAddr {
				dapp.Status = governance.GovernanceTransferred
				dappTransferred = append(dappTransferred, dapp)
			} else {
				ret = append(ret, dapp)
			}
		}
	}

	sort.Sort(Dapps(ret))
	sort.Sort(Dapps(dappTransferred))
	ret = append(ret, dappTransferred...)

	return ret, nil
}

func (dm *DappManager) IsAvailable(dappID string) *boltvm.Response {
	return boltvm.Success([]byte(strconv.FormatBool(dm.isAvailable(dappID))))
}

func (dm *DappManager) isAvailable(dappID string) bool {
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(dappID), dapp)
	if !ok {
		return false
	} else {
		return dapp.IsAvailable()
	}
}

func (dm *DappManager) packageDappInfo(dappID, name string, typ string, desc string, url, conAddrs string, permits, ownerAddr string,
	score float64, createTime int64, evaluationRecord map[string]*governance.EvaluationRecord, transferRecord []*TransferRecord, status governance.GovernanceStatus) (*Dapp, error) {
	if dappID == "" {
		// register
		dappMap := make(map[string]struct{})
		if ok := dm.GetObject(OwnerKey(ownerAddr), &dappMap); !ok {
			dappID = fmt.Sprintf("%s-0", ownerAddr)
		} else {
			dappID = fmt.Sprintf("%s-%d", ownerAddr, len(dappMap))
		}
	}

	contractAddr := make(map[string]struct{})
	if conAddrs != "" {
		for _, id := range strings.Split(conAddrs, ",") {
			contractAddr[id] = struct{}{}
		}
	}

	permission := make(map[string]struct{})
	if permits != "" {
		for _, id := range strings.Split(permits, ",") {
			permission[id] = struct{}{}
		}
	}

	dapp := &Dapp{
		DappID:            dappID,
		Name:              name,
		Type:              DappType(typ),
		Desc:              desc,
		Url:               url,
		ContractAddr:      contractAddr,
		Permission:        permission,
		OwnerAddr:         ownerAddr,
		Score:             score,
		CreateTime:        createTime,
		EvaluationRecords: evaluationRecord,
		TransferRecords:   transferRecord,
		Status:            status,
	}

	return dapp, nil
}

func (dm *DappManager) checkDappInfo(dapp *Dapp, isRegister bool) *boltvm.Response {
	// check url
	if strings.Trim(dapp.Url, " ") == "" {
		return boltvm.Error(boltvm.DappEmptyUrlCode, string(boltvm.DappEmptyUrlMsg))
	}

	// check name
	if dapp.Name == "" {
		return boltvm.Error(boltvm.DappEmptyNameCode, string(boltvm.DappEmptyNameMsg))
	}
	if ok, dappID := dm.isOccupiedName(dapp.Name); ok {
		if isRegister {
			return boltvm.Error(boltvm.DappDuplicateNameCode, fmt.Sprintf(string(boltvm.DappDuplicateNameMsg), dapp.Name, dappID))
		} else if dappID != dapp.DappID {
			return boltvm.Error(boltvm.DappDuplicateNameCode, fmt.Sprintf(string(boltvm.DappDuplicateNameMsg), dapp.Name, dappID))
		}

	}

	// check type
	if dapp.Type != DappTool &&
		dapp.Type != DappApplication &&
		dapp.Type != DappGame &&
		dapp.Type != DappOthers {
		return boltvm.Error(boltvm.DappIllegalTypeCode, fmt.Sprintf(string(boltvm.DappIllegalTypeMsg), dapp.Type))
	}

	// check contract addr
	for addr, _ := range dapp.ContractAddr {
		if _, err := types.HexDecodeString(addr); err != nil {
			return boltvm.Error(boltvm.DappIllegalContractAddrCode, fmt.Sprintf(string(boltvm.DappIllegalContractAddrMsg), addr))
		}

		if ok, dappID := dm.isOccupiedContractAddr(addr); ok {
			if isRegister {
				return boltvm.Error(boltvm.DappDuplicateContractRegisterCode, fmt.Sprintf(string(boltvm.DappDuplicateContractRegisterMsg), addr, dappID))
			} else if dappID != dapp.DappID {
				return boltvm.Error(boltvm.DappDuplicateContractUpdateCode, fmt.Sprintf(string(boltvm.DappDuplicateContractUpdateMsg), addr, dappID))
			}
		}

		account1 := dm.GetAccount(addr)
		account := account1.(ledger.IAccount)
		if account.CodeHash() == nil || bytes.Equal(account.CodeHash(), crypto.Keccak256(nil)) {
			return boltvm.Error(boltvm.DappNonexistentContractCode, fmt.Sprintf(string(boltvm.DappNonexistentContractMsg), addr))
		}
	}

	// check permission info
	for p, _ := range dapp.Permission {
		_, err := types.HexDecodeString(p)
		if err != nil {
			return boltvm.Error(boltvm.DappIllegalPermissionCode, fmt.Sprintf(string(boltvm.DappIllegalPermissionMsg), p, err.Error()))
		}
	}

	return boltvm.Success(nil)
}

func DappKey(id string) string {
	return fmt.Sprintf("%s-%s", DappPrefix, id)
}

func OwnerKey(addr string) string {
	return fmt.Sprintf("%s-%s", DappOwnerPrefix, addr)
}

func DappNameKey(name string) string {
	return fmt.Sprintf("%s-%s", DappNamePrefix, name)
}

func DappOccupyNameKey(name string) string {
	return fmt.Sprintf("%s-%s", DappOccupyNamePrefix, name)
}

func DappOccupyContractKey(addr string) string {
	return fmt.Sprintf("%s-%s", DappOccupyContractPrefix, addr)
}

type Dapps []*Dapp

func (ds Dapps) Len() int { return len(ds) }

func (ds Dapps) Swap(i, j int) { ds[i], ds[j] = ds[j], ds[i] }

func (ds Dapps) Less(i, j int) bool {
	return ds[i].CreateTime > ds[j].CreateTime
}

func (dm *DappManager) postAuditDappEvent(dappID string) error {
	ok, dappData := dm.Get(DappKey(dappID))
	if !ok {
		return fmt.Errorf("not found dapp %s", dappID)
	}

	auditInfo := &pb.AuditRelatedObjInfo{
		AuditObj:           dappData,
		RelatedChainIDList: map[string][]byte{},
		RelatedNodeIDList:  map[string][]byte{},
	}
	dm.PostEvent(pb.Event_AUDIT_DAPP, auditInfo)

	return nil
}
