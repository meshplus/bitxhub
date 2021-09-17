package contracts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

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
	DAPPPREFIX          = "dapp"
	OWNERPREFIX         = "owner"
	DAPPCONTRACT_PREFIX = "contract"
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

	Score             float64                      `json:"score"`
	EvaluationRecords map[string]*EvaluationRecord `json:"evaluation_records"`
	TransferRecords   []*TransferRecord            `json:"transfer_records"`

	Status governance.GovernanceStatus `json:"status"`
	FSM    *fsm.FSM                    `json:"fsm"`
}

type EvaluationRecord struct {
	Addr       string  `json:"addr"`
	Score      float64 `json:"score"`
	Desc       string  `json:"desc"`
	CreateTime int64   `json:"create_time"`
}

type TransferRecord struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Reason     string `json:"reason"`
	Confirm    bool   `json:"confirm"`
	CreateTime int64  `json:"create_time"`
}

var dappStateMap = map[governance.EventType][]governance.GovernanceStatus{
	governance.EventRegister: {governance.GovernanceUnavailable},
	governance.EventUpdate:   {governance.GovernanceAvailable, governance.GovernanceFrozen},
	governance.EventFreeze:   {governance.GovernanceAvailable, governance.GovernanceUpdating, governance.GovernanceActivating},
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
			{Name: string(governance.EventUpdate), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceFrozen), string(governance.GovernanceFreezing)}, Dst: string(governance.GovernanceUpdating)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceFrozen)},

			// freeze 2
			{Name: string(governance.EventFreeze), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceUpdating), string(governance.GovernanceActivating), string(governance.GovernanceTransferring)}, Dst: string(governance.GovernanceFreezing)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceFreezing)}, Dst: string(governance.GovernanceFrozen)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceFreezing)}, Dst: string(lastStatus)},

			// activate 1
			{Name: string(governance.EventActivate), Src: []string{string(governance.GovernanceFrozen), string(governance.GovernanceFreezing)}, Dst: string(governance.GovernanceActivating)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceActivating)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceActivating)}, Dst: string(lastStatus)},

			// transfer 1
			{Name: string(governance.EventTransfer), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceFreezing)}, Dst: string(governance.GovernanceTransferring)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceTransferring)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceTransferring)}, Dst: string(lastStatus)},
		},
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) { d.Status = governance.GovernanceStatus(d.FSM.Current()) },
		},
	)
}

// GovernancePre checks if the dapp can do the event. (only check, not modify infomation)
func (dm *DappManager) governancePre(dappID string, event governance.EventType) (*Dapp, error) {
	dapp := &Dapp{}
	if ok := dm.GetObject(DappKey(dappID), dapp); !ok {
		if event == governance.EventRegister {
			return nil, nil
		} else {
			return nil, fmt.Errorf("this dapp does not exist")
		}
	}

	for _, s := range dappStateMap[event] {
		if dapp.Status == s {
			return dapp, nil
		}
	}

	return nil, fmt.Errorf("The dapp (%s) can not be %s", string(dapp.Status), string(event))
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
			res := dm.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin",
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

// ========================== Governance interface ========================
// =========== Manage does some subsequent operations when the proposal is over
// extra: update - dapp info, transfer - transfer record , register - dapp
func (dm *DappManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := dm.checkPermission([]string{string(PermissionSpecific)}, objId, dm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. change status
	if ok, data := dm.changeStatus(objId, proposalResult, lastStatus); !ok {
		return boltvm.Error(fmt.Sprintf("change status error:%s", string(data)))
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			if err := dm.manegeRegister(objId, extra); err != nil {
				return boltvm.Error(fmt.Sprintf("manage register error: %v", err))
			}
		case string(governance.EventUpdate):
			if err := dm.manegeUpdate(objId, extra); err != nil {
				return boltvm.Error(fmt.Sprintf("manage update error: %v", err))
			}
		case string(governance.EventTransfer):
			if err := dm.manegeTransfer(objId, extra); err != nil {
				return boltvm.Error(fmt.Sprintf("manage update error: %v", err))
			}
		}
	}

	return boltvm.Success(nil)
}

func (dm *DappManager) manegeRegister(id string, registerData []byte) error {
	registerInfo := &Dapp{}
	if err := json.Unmarshal(registerData, registerInfo); err != nil {
		return fmt.Errorf("unmarshal register data error:%v", err)
	}

	dappContractMap := make(map[string]string)
	_ = dm.GetObject(DAPPCONTRACT_PREFIX, &dappContractMap)

	for addr, _ := range registerInfo.ContractAddr {
		dappContractMap[addr] = registerInfo.DappID
	}

	dm.SetObject(DAPPCONTRACT_PREFIX, dappContractMap)
	return nil
}

func (dm *DappManager) manegeUpdate(id string, updateData []byte) error {
	updataInfo := &Dapp{}
	if err := json.Unmarshal(updateData, updataInfo); err != nil {
		return fmt.Errorf("unmarshal update data error:%v", err)
	}

	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return fmt.Errorf("the dapp is not exist")
	}

	dapp.Name = updataInfo.Name
	dapp.Type = updataInfo.Type
	dapp.Desc = updataInfo.Desc
	dapp.Permission = updataInfo.Permission
	dm.SetObject(DappKey(dapp.DappID), *dapp)
	return nil
}

func (dm *DappManager) manegeTransfer(id string, transferData []byte) error {
	transRec := &TransferRecord{}
	if err := json.Unmarshal(transferData, transRec); err != nil {
		return fmt.Errorf("unmarshal update data error:%v", err)
	}
	transRec.CreateTime = dm.GetTxTimeStamp()

	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return fmt.Errorf("the dapp is not exist")
	}

	dapp.TransferRecords = append(dapp.TransferRecords, transRec)
	dapp.OwnerAddr = transRec.To
	dm.SetObject(DappKey(dapp.DappID), *dapp)
	dm.addToOwner(dapp.OwnerAddr, dapp.DappID)
	return nil
}

func (dm *DappManager) addToOwner(ownerAddr, dappID string) {
	dappMap := make(map[string]struct{})
	_ = dm.GetObject(OwnerKey(ownerAddr), &dappMap)
	dappMap[dappID] = struct{}{}
	dm.SetObject(OwnerKey(ownerAddr), dappMap)
}

// =========== RegisterDapp registers dapp info, returns proposal id and error
func (dm *DappManager) RegisterDapp(name, typ, desc, url, conAddrs, permits, reason string) *boltvm.Response {
	event := governance.EventRegister

	// 1. get dapp info
	dapp, err := dm.packageDappInfo("", name, typ, desc, url, conAddrs, permits, dm.Caller(), 0, dm.GetTxTimeStamp(), make(map[string]*EvaluationRecord), nil, governance.GovernanceRegisting)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("get dapp info error: %v", err))
	}

	// 2. check dapp info
	if err := dm.checkDappInfo(dapp, true); err != nil {
		return boltvm.Error(fmt.Sprintf("check dapp info error : %v", err))
	}

	// 3. governancePre: check status
	if _, err := dm.governancePre(dapp.DappID, event); err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 4. submit proposal
	dappData, err := json.Marshal(dapp)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal dapp err: %v", err))
	}
	res := dm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(dm.Caller()),
		pb.String(string(event)),
		pb.String(""),
		pb.String(string(DappMgr)),
		pb.String(dapp.DappID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.String(reason),
		pb.Bytes(dappData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 5. register info
	dm.SetObject(DappKey(dapp.DappID), *dapp)
	dm.addToOwner(dapp.OwnerAddr, dapp.DappID)
	dm.Logger().WithFields(logrus.Fields{
		"id": dapp.DappID,
	}).Info("Dapp is registering")

	return getGovernanceRet(string(res.Result), []byte(dapp.DappID))
}

// =========== UpdateDapp updates dapp info.
func (dm *DappManager) UpdateDapp(id, name, typ, desc, url, conAddrs, permits, reason string) *boltvm.Response {
	event := governance.EventUpdate

	// 1. governance pre: check if exist and status
	oldDapp, err := dm.governancePre(id, event)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 2. check permission: PermissionSelf
	if err := dm.checkPermission([]string{string(PermissionSelf)}, oldDapp.OwnerAddr, dm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 3. get info
	newDapp, err := dm.packageDappInfo(id, name, typ, desc, url, conAddrs, permits, oldDapp.OwnerAddr, oldDapp.Score, oldDapp.CreateTime, oldDapp.EvaluationRecords, oldDapp.TransferRecords, oldDapp.Status)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("get dapp info error: %v", err))
	}
	// 4. check info
	if err := dm.checkDappInfo(newDapp, false); err != nil {
		return boltvm.Error(fmt.Sprintf("check dapp info error : %v", err))
	}

	// 5. submit proposal
	dappData, err := json.Marshal(newDapp)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("dapp marshal error: %v", err))
	}

	res := dm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(dm.Caller()),
		pb.String(string(event)),
		pb.String(""),
		pb.String(string(DappMgr)),
		pb.String(id),
		pb.String(string(oldDapp.Status)),
		pb.String(reason),
		pb.Bytes(dappData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	// 6. change status
	if ok, data := dm.changeStatus(id, string(event), string(oldDapp.Status)); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
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
		return boltvm.Error(fmt.Sprintf("illegal new owner addr: %s", newOwnerAddr))
	}

	transRec := &TransferRecord{
		From:    dm.Caller(),
		To:      newOwnerAddr,
		Reason:  reason,
		Confirm: false,
	}
	extra, err := json.Marshal(transRec)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal extra error: %v", err))
	}
	return dm.basicGovernance(id, reason, []string{string(PermissionSelf)}, governance.EventTransfer, extra)
}

func (dm *DappManager) basicGovernance(id, reason string, permissions []string, event governance.EventType, extra []byte) *boltvm.Response {
	// 1. governance pre: check if exist and status
	dapp, err := dm.governancePre(id, event)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}

	// 2. check permission
	if err := dm.checkPermission(permissions, dapp.OwnerAddr, dm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 3. submit proposal
	res := dm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(dm.Caller()),
		pb.String(string(event)),
		pb.String(""),
		pb.String(string(DappMgr)),
		pb.String(id),
		pb.String(string(dapp.Status)),
		pb.String(reason),
		pb.Bytes(extra),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 4. change status
	if ok, data := dm.changeStatus(id, string(event), string(dapp.Status)); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
	}

	return getGovernanceRet(string(res.Result), nil)
}

func (dm *DappManager) ConfirmTransfer(id string) *boltvm.Response {
	// 1. get dapp
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return boltvm.Error("the dapp is not exist")
	}

	// 2. check permission
	if err := dm.checkPermission([]string{string(PermissionSelf)}, dapp.OwnerAddr, dm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 3. confirm
	if len(dapp.TransferRecords) != 0 {
		if !dapp.TransferRecords[len(dapp.TransferRecords)-1].Confirm {
			dapp.TransferRecords[len(dapp.TransferRecords)-1].Confirm = true
			dm.SetObject(DappKey(id), *dapp)
		}
	}

	return boltvm.Success(nil)
}

func (dm *DappManager) EvaluateDapp(id, desc string, score float64) *boltvm.Response {
	if score < 0 || score > 5 {
		return boltvm.Error("the score should be in the range [0,5]")
	}

	// 1. get dapp
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return boltvm.Error("the dapp is not exist")
	}

	// 2. Check whether caller has evaluated
	if _, ok := dapp.EvaluationRecords[dm.Caller()]; ok {
		return boltvm.Error("the caller has evaluate the dapp")
	}

	// 3. get evaluation record
	evaRec := &EvaluationRecord{
		Addr:       dm.Caller(),
		Score:      score,
		Desc:       desc,
		CreateTime: dm.GetTxTimeStamp(),
	}

	// 4. store record
	num := float64(len(dapp.EvaluationRecords))
	dapp.Score = num/(num+1)*dapp.Score + 1/(num+1)*score
	dapp.EvaluationRecords[dm.Caller()] = evaRec
	dm.SetObject(DappKey(id), *dapp)

	return boltvm.Success(nil)
}

// ========================== Query interface ========================
// GetDapp returns dapp info by dapp id
func (dm *DappManager) GetDapp(id string) *boltvm.Response {
	dapp := &Dapp{}
	ok := dm.GetObject(DappKey(id), dapp)
	if !ok {
		return boltvm.Error("the dapp is not exist")
	}

	data, err := json.Marshal(dapp)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(data)
}

// GetAllDapps returns all dapps
func (dm *DappManager) GetAllDapps() *boltvm.Response {
	ret, err := dm.getAll()
	if err != nil {
		return boltvm.Error(err.Error())
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

func (dm *DappManager) getAll() ([]*Dapp, error) {
	ret := make([]*Dapp, 0)
	ok, value := dm.Query(DAPPPREFIX)
	if ok {
		for _, data := range value {
			dapp := &Dapp{}
			if err := json.Unmarshal(data, dapp); err != nil {
				return nil, err
			}
			ret = append(ret, dapp)
		}
	}

	return ret, nil
}

// GetAllDapps returns all dapps
func (dm *DappManager) GetPermissionDapps() *boltvm.Response {
	var ret []*Dapp
	all, err := dm.getAll()
	if err == nil {
		for _, d := range all {
			if _, ok := d.Permission[dm.Caller()]; !ok {
				ret = append(ret, d)
			}
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

// get dApps by owner addr, including dApps a person currently owns and the dApps they once owned
func (dm *DappManager) GetDappsByOwner(ownerAddr string) *boltvm.Response {
	ret, err := dm.getOwnerAll(ownerAddr)

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
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
	score float64, createTime int64, evaluationRecord map[string]*EvaluationRecord, transferRecord []*TransferRecord, status governance.GovernanceStatus) (*Dapp, error) {
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
	for _, id := range strings.Split(conAddrs, ",") {
		contractAddr[id] = struct{}{}
	}

	permission := make(map[string]struct{})
	for _, id := range strings.Split(permits, ",") {
		permission[id] = struct{}{}
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

func (dm *DappManager) checkDappInfo(dapp *Dapp, isRegister bool) error {
	// check type
	if dapp.Type != DappTool &&
		dapp.Type != DappApplication &&
		dapp.Type != DappGame &&
		dapp.Type != DappOthers {
		return fmt.Errorf("illegal dapp type: %s", dapp.Type)
	}
	// check contract addr
	dappContractMap := make(map[string]string)
	_ = dm.GetObject(DAPPCONTRACT_PREFIX, &dappContractMap)
	for a, _ := range dapp.ContractAddr {
		if isRegister {
			dappID, exist := dappContractMap[a]
			if exist {
				return fmt.Errorf("the contract address belongs to dapp %s and cannot be registered repeatedly", dappID)
			}
		}
		account1 := dm.GetAccount(a)
		account := account1.(ledger.IAccount)
		if account.Code() == nil {
			return fmt.Errorf("the contract addr does not exist")
		}
	}

	// check permission info
	for p, _ := range dapp.Permission {
		_, err := types.HexDecodeString(p)
		if err != nil {
			return fmt.Errorf("illegal user addr in permission: %s", p)
		}
	}

	return nil
}

func DappKey(id string) string {
	return fmt.Sprintf("%s-%s", DAPPPREFIX, id)
}

func OwnerKey(addr string) string {
	return fmt.Sprintf("%s-%s", OWNERPREFIX, addr)
}

type Dapps []*Dapp

func (ds Dapps) Len() int { return len(ds) }

func (ds Dapps) Swap(i, j int) { ds[i], ds[j] = ds[j], ds[i] }

func (ds Dapps) Less(i, j int) bool {
	return ds[i].DappID > ds[j].DappID
}
