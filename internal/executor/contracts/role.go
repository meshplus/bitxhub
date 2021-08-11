package contracts

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/looplab/fsm"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/meshplus/eth-kit/ledger"
	"github.com/sirupsen/logrus"
)

type RoleType string

const (
	GenesisBalance = "genesis_balance"

	ROLEPREFIX = "role"

	GovernanceAdmin RoleType = "governanceAdmin"
	AuditAdmin      RoleType = "auditAdmin"
	AppchainAdmin   RoleType = "appchainAdmin"
	NoRole          RoleType = "none"
)

type Role struct {
	ID       string   `toml:"id" json:"id"`
	RoleType RoleType `toml:"role_type" json:"role_type"`

	// 	GovernanceAdmin info
	Weight uint64 `json:"weight" toml:"weight"`

	// AuditAdmin info
	NodePid string `toml:"pid" json:"pid"`

	Status governance.GovernanceStatus `toml:"status" json:"status"`
	FSM    *fsm.FSM                    `json:"fsm"`
}

type RoleManager struct {
	boltvm.Stub
}

var roleStateMap = map[governance.EventType][]governance.GovernanceStatus{
	governance.EventRegister: {governance.GovernanceUnavailable},
	governance.EventUpdate:   {governance.GovernanceAvailable, governance.GovernanceFrozen},
	governance.EventFreeze:   {governance.GovernanceAvailable, governance.GovernanceUpdating, governance.GovernanceActivating},
	governance.EventActivate: {governance.GovernanceFrozen},
	governance.EventLogout:   {governance.GovernanceAvailable, governance.GovernanceUpdating, governance.GovernanceFreezing, governance.GovernanceActivating, governance.GovernanceFrozen},
}

var RoleAvailableState = []governance.GovernanceStatus{
	governance.GovernanceAvailable,
	governance.GovernanceUpdating,
	governance.GovernanceFreezing,
	governance.GovernanceLogouting,
}

func setFSM(role *Role, lastStatus governance.GovernanceStatus) {
	role.FSM = fsm.NewFSM(
		string(role.Status),
		fsm.Events{
			// register 3
			{Name: string(governance.EventRegister), Src: []string{string(governance.GovernanceUnavailable)}, Dst: string(governance.GovernanceRegisting)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceRegisting)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceRegisting)}, Dst: string(lastStatus)},

			// update 1
			{Name: string(governance.EventUpdate), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceFrozen), string(governance.GovernanceFreezing), string(governance.GovernanceLogouting)}, Dst: string(governance.GovernanceUpdating)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceUpdating)}, Dst: string(lastStatus)},

			// freeze 2
			{Name: string(governance.EventFreeze), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceUpdating), string(governance.GovernanceActivating), string(governance.GovernanceLogouting)}, Dst: string(governance.GovernanceFreezing)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceFreezing)}, Dst: string(governance.GovernanceFrozen)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceFreezing)}, Dst: string(lastStatus)},

			// active 1
			{Name: string(governance.EventActivate), Src: []string{string(governance.GovernanceFrozen), string(governance.GovernanceFreezing), string(governance.GovernanceLogouting)}, Dst: string(governance.GovernanceActivating)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceActivating)}, Dst: string(governance.GovernanceAvailable)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceActivating)}, Dst: string(lastStatus)},

			// logout 3
			{Name: string(governance.EventLogout), Src: []string{string(governance.GovernanceAvailable), string(governance.GovernanceUpdating), string(governance.GovernanceFreezing), string(governance.GovernanceFrozen), string(governance.GovernanceActivating)}, Dst: string(governance.GovernanceLogouting)},
			{Name: string(governance.EventApprove), Src: []string{string(governance.GovernanceLogouting)}, Dst: string(governance.GovernanceForbidden)},
			{Name: string(governance.EventReject), Src: []string{string(governance.GovernanceLogouting)}, Dst: string(lastStatus)},
		},
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) {
				role.Status = governance.GovernanceStatus(role.FSM.Current())
			},
		},
	)
}

// GovernancePre checks if the node can do the event. (only check, not modify infomation)
func (rm *RoleManager) governancePre(roleId string, event governance.EventType, _ []byte) (*Role, error) {
	role := &Role{}
	if ok := rm.GetObject(rm.roleKey(roleId), role); !ok {
		if event == governance.EventRegister {
			return nil, nil
		} else {
			return nil, fmt.Errorf("this role does not exist")
		}
	}

	for _, s := range roleStateMap[event] {
		if role.Status == s {
			return role, nil
		}
	}

	return nil, fmt.Errorf("The role (%s) can not be %s", string(role.Status), string(event))
}

func (rm *RoleManager) changeStatus(roleId string, trigger, lastStatus string) (bool, []byte) {
	role := &Role{}
	if ok := rm.GetObject(rm.roleKey(roleId), role); !ok {
		return false, []byte("this role does not exist")
	}

	setFSM(role, governance.GovernanceStatus(lastStatus))
	err := role.FSM.Event(trigger)
	if err != nil {
		return false, []byte(fmt.Sprintf("change status error: %v", err))
	}

	rm.SetObject(rm.roleKey(roleId), *role)
	return true, nil
}

// extra: Role
func (rm *RoleManager) Manage(eventTyp string, proposalResult, lastStatus string, extra []byte) *boltvm.Response {
	// 1. check permission
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + err.Error())
	}

	res := rm.CheckPermission(string(PermissionSpecific), "", rm.CurrentCaller(), addrsData)
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	// 2. change status
	role := &Role{}
	if err := json.Unmarshal(extra, role); err != nil {
		return boltvm.Error("unmarshal json error:" + err.Error())
	}

	ok, errData := rm.changeStatus(role.ID, proposalResult, lastStatus)
	if !ok {
		return boltvm.Error("change status error:" + string(errData))
	}

	// 3. other handle
	if proposalResult == string(APPOVED) {
		switch eventTyp {
		case string(governance.EventUpdate):
			rm.SetObject(rm.roleKey(role.ID), *role)
		case string(governance.EventFreeze):
			fallthrough
		case string(governance.EventLogout):
			fallthrough
		case string(governance.EventActivate):
			if err := rm.updateRoleRelatedProposalInfo(role.ID, governance.EventType(eventTyp)); err != nil {
				return boltvm.Error(err.Error())
			}
		}
	}

	return boltvm.Success(nil)
}

// Update proposal information related to the administrator
func (rm *RoleManager) updateRoleRelatedProposalInfo(roleID string, eventTyp governance.EventType) error {
	res := rm.CrossInvoke(constant.GovernanceContractAddr.String(), "GetNotClosedProposals")
	if !res.Ok {
		return fmt.Errorf("cross invoke GetProposalsByStatus error: %s", string(res.Result))
	}
	var proposals []Proposal
	err := json.Unmarshal(res.Result, &proposals)
	if err != nil {
		return fmt.Errorf("unmarshal proposals error: %v", err.Error())
	}

	for _, p := range proposals {
		for _, e := range p.ElectorateList {
			if e.ID == roleID {
				switch eventTyp {
				case governance.EventFreeze:
					fallthrough
				case governance.EventLogout:
					p.AvaliableElectorateNum--
				case governance.EventActivate:
					p.AvaliableElectorateNum++
				default:
					break
				}
				res := rm.CrossInvoke(constant.GovernanceContractAddr.String(), "UpdateAvaliableElectorateNum", pb.String(p.Id), pb.Uint64(p.AvaliableElectorateNum))
				if !res.Ok {
					return fmt.Errorf("cross invoke UpdateAvaliableElectorateNum error: %s", string(res.Result))
				}
				break
			}
		}
	}
	return nil
}

// Register registers role info
// caller is the bitxhub admin address
// return role id proposal id and error
func (rm *RoleManager) RegisterRole(roleId, roleType, nodePid string) *boltvm.Response {
	// 1. check permission
	res := rm.CheckPermission(string(PermissionAdmin), roleId, rm.CurrentCaller(), nil)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	// 2. check info
	role := &Role{
		ID:       roleId,
		RoleType: RoleType(roleType),
		Weight:   repo.NormalAdminWeight,
		NodePid:  nodePid,
		Status:   governance.GovernanceUnavailable,
	}
	if err := rm.checkRoleInfo(role); err != nil {
		return boltvm.Error(fmt.Sprintf("check node info error: %s", err.Error()))
	}

	// 3. check status
	if _, err := rm.governancePre(roleId, governance.EventRegister, nil); err != nil {
		return boltvm.Error(fmt.Sprintf("register prepare error: %v", err))
	}

	// 4. register
	rm.SetObject(rm.roleKey(roleId), *role)
	ok, gb := rm.Get(GenesisBalance)
	if !ok {
		return boltvm.Error("get genesis balance error")
	}
	balance, _ := new(big.Int).SetString(string(gb), 10)
	account := rm.GetAccount(role.ID)
	acc := account.(ledger.IAccount)
	acc.AddBalance(balance)
	rm.Logger().WithFields(logrus.Fields{
		"id":       role.ID,
		"roleType": role.RoleType,
	}).Info("Role is registering")

	roleData, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal role error: %v", err))
	}

	// 5. submit proposal
	res = rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventRegister)),
		pb.String(""),
		pb.String(string(RoleMgr)),
		pb.String(role.ID),
		pb.String(string(role.Status)),
		pb.Bytes(roleData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 6. change status
	if ok, data := rm.changeStatus(role.ID, string(governance.EventRegister), string(role.Status)); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s, %s", string(data), role.ID))
	}
	return getGovernanceRet(string(res.Result), []byte(role.ID))
}

// UpdateAuditAdminNode updates nodeId of nvp role
func (rm *RoleManager) UpdateAuditAdminNode(roleId, nodePid string) *boltvm.Response {
	// 1. check permission
	res := rm.CheckPermission(string(PermissionSelfAdmin), roleId, rm.CurrentCaller(), nil)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	// 2. check status
	role, err := rm.governancePre(roleId, governance.EventUpdate, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("register prepare error: %v", err))
	}

	// 3. check info
	if AuditAdmin != role.RoleType {
		return boltvm.Error(fmt.Sprintf("the role is not a AuditAdmin: %s", string(role.RoleType)))
	}
	if nodePid == role.NodePid {
		return boltvm.Error(fmt.Sprintf("the node ID is the same as before: %s", nodePid))
	}

	res = rm.CrossInvoke(constant.NodeManagerContractAddr.String(), "GetNode", pb.String(nodePid))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("cross invoke GetNode error: %s", string(res.Result)))
	}
	var nodeTmp node_mgr.Node
	if err := json.Unmarshal(res.Result, &nodeTmp); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal node error: %v", err))
	}
	if node_mgr.NVPNode != nodeTmp.NodeType {
		return boltvm.Error(fmt.Sprintf("the node is not a nvp node: %s", string(nodeTmp.NodeType)))
	}

	// 4. submit proposal
	role.NodePid = nodePid
	roleData, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal role error: %v", err))
	}

	res = rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventUpdate)),
		pb.String(""),
		pb.String(string(RoleMgr)),
		pb.String(roleId),
		pb.String(string(role.Status)),
		pb.Bytes(roleData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	// 5. change status
	if ok, data := rm.changeStatus(roleId, string(governance.EventUpdate), string(role.Status)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// FreezeRole freezes available role
func (rm *RoleManager) FreezeRole(roleId string) *boltvm.Response {
	// 1. check permission
	res := rm.CheckPermission(string(PermissionAdmin), roleId, rm.CurrentCaller(), nil)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	// 2. check status
	role, err := rm.governancePre(roleId, governance.EventFreeze, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("freeze prepare error: %v", err))
	}
	if role.Weight == repo.SuperAdminWeight {
		return boltvm.Error("super governance admin can not be freeze")
	}

	// 3. submit proposal
	roleData, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal role error: %v", err))
	}
	res = rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventFreeze)),
		pb.String(""),
		pb.String(string(RoleMgr)),
		pb.String(roleId),
		pb.String(string(role.Status)),
		pb.Bytes(roleData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	// 4. change status
	if ok, data := rm.changeStatus(roleId, string(governance.EventFreeze), string(role.Status)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// ActivateRole updates frozen role
func (rm *RoleManager) ActivateRole(roleId string) *boltvm.Response {
	// 1. check permission
	res := rm.CheckPermission(string(PermissionAdmin), roleId, rm.CurrentCaller(), nil)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	// 2. check status
	role, err := rm.governancePre(roleId, governance.EventActivate, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("active prepare error: %v", err))
	}

	// 3. submit proposal
	roleData, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal role error: %v", err))
	}
	res = rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventActivate)),
		pb.String(""),
		pb.String(string(RoleMgr)),
		pb.String(roleId),
		pb.String(string(role.Status)),
		pb.Bytes(roleData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	// 4. change status
	if ok, data := rm.changeStatus(roleId, string(governance.EventActivate), string(role.Status)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// LogoutRole logout role
func (rm *RoleManager) LogoutRole(roleId string) *boltvm.Response {
	// 1. check permission
	res := rm.CheckPermission(string(PermissionAdmin), roleId, rm.CurrentCaller(), nil)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	// 2. check status
	role, err := rm.governancePre(roleId, governance.EventLogout, nil)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("active prepare error: %v", err))
	}
	if role.Weight == repo.SuperAdminWeight {
		return boltvm.Error("super governance admin can not be logout")
	}

	// 3. submit proposal
	roleData, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal role error: %v", err))
	}
	res = rm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(governance.EventLogout)),
		pb.String(""),
		pb.String(string(RoleMgr)),
		pb.String(roleId),
		pb.String(string(role.Status)),
		pb.Bytes(roleData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	// 4. change status
	if ok, data := rm.changeStatus(roleId, string(governance.EventLogout), string(role.Status)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// GetRole return the role of the caller
func (rm *RoleManager) GetRole() *boltvm.Response {
	res, err := rm.getRole(rm.Caller())
	if err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success([]byte(res))
	}
}

// GetRole return the role of the caller
func (rm *RoleManager) GetRoleByAddr(addr string) *boltvm.Response {
	res, err := rm.getRole(addr)
	if err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success([]byte(res))
	}
}

// GetRole return the role of the caller
func (rm *RoleManager) getRole(addr string) (string, error) {

	role := &Role{}
	ok := rm.GetObject(rm.roleKey(addr), role)
	if !ok {
		res := rm.CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAppchainAdmin")
		if !res.Ok {
			return string(NoRole), nil
		} else {
			return string(AppchainAdmin), nil
		}
	}

	switch role.RoleType {
	case GovernanceAdmin:
		if role.Weight == repo.SuperAdminWeight {
			return fmt.Sprintf("%s(super)", GovernanceAdmin), nil
		} else {
			return string(GovernanceAdmin), nil
		}
	case AuditAdmin:
		return string(AuditAdmin), nil
	}
	return string(NoRole), nil
}

// GetRole query a role by roleId
func (rm *RoleManager) GetRoleById(roleId string) *boltvm.Response {
	role := &Role{}
	ok := rm.GetObject(rm.roleKey(roleId), role)
	if !ok {
		return boltvm.Error("the role is not exist")
	}

	data, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(data)
}

func (rm *RoleManager) GetAdminRoles() *boltvm.Response {
	return rm.getRoles(string(GovernanceAdmin))
}

func (rm *RoleManager) GetAuditAdminRoles() *boltvm.Response {
	return rm.getRoles(string(AuditAdmin))
}

func (rm *RoleManager) getRoles(roleType string) *boltvm.Response {
	ok, value := rm.Query(ROLEPREFIX)
	if !ok {
		return boltvm.Error("there is no admins")
	}

	ret := make([]*Role, 0)
	for _, data := range value {
		role := &Role{}
		if err := json.Unmarshal(data, role); err != nil {
			return boltvm.Error(fmt.Sprintf("unmarshal role error: %v", err.Error()))
		}
		if role.RoleType == RoleType((roleType)) {
			ret = append(ret, role)
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal role error: %v", err.Error()))
	}

	return boltvm.Success(data)
}

func (rm *RoleManager) GetAvailableRoles(roleTypesData []byte) *boltvm.Response {
	ok, value := rm.Query(ROLEPREFIX)
	if !ok {
		return boltvm.Error("there is no admins")
	}

	var roleTypes []string
	if err := json.Unmarshal(roleTypesData, &roleTypes); err != nil {
		return boltvm.Error(err.Error())
	}

	ret := make([]*Role, 0)
	for _, data := range value {
		role := &Role{}
		if err := json.Unmarshal(data, role); err != nil {
			return boltvm.Error(err.Error())
		}
		for _, rt := range roleTypes {
			if role.RoleType == RoleType(rt) && rm.isAvailable(role.ID) {
				ret = append(ret, role)
			}
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(data)
}

// IsAvailable determines whether the role  is available
func (rm *RoleManager) IsAvailable(roleId string) *boltvm.Response {
	return boltvm.Success([]byte(strconv.FormatBool(rm.isAvailable(roleId))))
}

func (rm *RoleManager) isAvailable(roleId string) bool {
	role := &Role{}
	ok := rm.GetObject(rm.roleKey(roleId), role)
	if !ok {
		return false
	}

	for _, s := range RoleAvailableState {
		if role.Status == s {
			return true
		}
	}

	return false
}

// IsSuperAdmin determines whether the role  is super GovernanceAdmin
func (rm *RoleManager) IsSuperAdmin(roleId string) *boltvm.Response {
	role := &Role{}
	ok := rm.GetObject(rm.roleKey(roleId), role)
	if !ok {
		return boltvm.Error("the role is not exist")
	}

	if GovernanceAdmin == role.RoleType && repo.SuperAdminWeight == role.Weight {
		return boltvm.Success([]byte(strconv.FormatBool(true)))
	}

	return boltvm.Success([]byte(strconv.FormatBool(false)))
}

// IsAdmin determines whether the role is GovernanceAdmin
func (rm *RoleManager) IsAdmin(roleId string) *boltvm.Response {
	return boltvm.Success([]byte(strconv.FormatBool(rm.isAdmin(roleId, GovernanceAdmin))))
}

// IsAdmin determines whether the role is Audit Admin
func (rm *RoleManager) IsAuditAdmin(roleId string) *boltvm.Response {
	return boltvm.Success([]byte(strconv.FormatBool(rm.isAdmin(roleId, AuditAdmin))))
}

func (rm *RoleManager) isAdmin(roleId string, roleType RoleType) bool {
	role := &Role{}
	ok := rm.GetObject(rm.roleKey(roleId), role)
	if !ok {
		return false
	}

	if roleType == role.RoleType {
		return true
	} else {
		return false
	}
}

func (rm *RoleManager) IsAnyAdmin(roleId string) *boltvm.Response {
	role := &Role{}
	ok := rm.GetObject(rm.roleKey(roleId), role)
	if !ok {
		return boltvm.Success([]byte(strconv.FormatBool(false)))
	}

	return boltvm.Success([]byte(strconv.FormatBool(true)))
}

func (rm *RoleManager) GetRoleWeight(roleId string) *boltvm.Response {
	role := &Role{}
	ok := rm.GetObject(rm.roleKey(roleId), role)
	if !ok {
		return boltvm.Error("the role is not exist")
	}

	if role.RoleType != GovernanceAdmin {
		return boltvm.Error("the role is not governane admin")
	}

	return boltvm.Success([]byte(strconv.Itoa(int(role.Weight))))
}

// Permission manager
type Permission string

const (
	PermissionSelf      Permission = "PermissionSelf"
	PermissionAdmin     Permission = "PermissionAdmin"
	PermissionSelfAdmin Permission = "PermissionSelfAdmin"
	PermissionSpecific  Permission = "PermissionSpecific"
)

func (rm *RoleManager) CheckPermission(permission string, regulatedAddr string, regulatorAddr string, specificAddrsData []byte) *boltvm.Response {
	switch permission {
	case string(PermissionSelf):
		if regulatorAddr != regulatedAddr {
			return boltvm.Error(fmt.Sprintf("caller(%s) is not regulated self(%s)", regulatorAddr, regulatedAddr))
		}
	case string(PermissionAdmin):
		if !rm.isAdmin(regulatorAddr, GovernanceAdmin) {
			return boltvm.Error(fmt.Sprintf("caller(%s) is not an admin account", regulatorAddr))
		}
		if !rm.isAvailable(regulatorAddr) {
			return boltvm.Error(fmt.Sprintf("caller(%s) is an unavailable admin", regulatorAddr))
		}
		if regulatorAddr == regulatedAddr {
			return boltvm.Error(fmt.Sprintf("Administrators cannot manage themselves(%s)", regulatorAddr))
		}
	case string(PermissionSelfAdmin):
		if regulatorAddr != regulatedAddr {
			if !rm.isAdmin(regulatorAddr, GovernanceAdmin) {
				return boltvm.Error(fmt.Sprintf("caller(%s) is not an admin account or regulated self(%s)", regulatorAddr, regulatedAddr))
			}
			if !rm.isAvailable(regulatorAddr) {
				return boltvm.Error(fmt.Sprintf("caller(%s) is an unavailable admin", regulatorAddr))
			}
		}
	case string(PermissionSpecific):
		specificAddrs := []string{}
		err := json.Unmarshal(specificAddrsData, &specificAddrs)
		if err != nil {
			return boltvm.Error(err.Error())
		}
		for _, addr := range specificAddrs {
			if addr == regulatorAddr {
				return boltvm.Success(nil)
			}
		}
		return boltvm.Error("caller(" + regulatorAddr + ") is not specific account")
	default:
		return boltvm.Error("unsupport permission: " + permission)
	}

	return boltvm.Success(nil)
}

func (rm *RoleManager) checkRoleInfo(role *Role) error {
	_, err := types.HexDecodeString(role.ID)
	if err != nil {
		return fmt.Errorf("illegal role id")
	}

	switch role.RoleType {
	case GovernanceAdmin:
	case AuditAdmin:
		res := rm.CrossInvoke(constant.NodeManagerContractAddr.String(), "GetNode", pb.String(role.NodePid))
		if !res.Ok {
			return fmt.Errorf("CrossInvoke GetNode error: %s", string(res.Result))
		}
		var nodeTmp node_mgr.Node
		if err := json.Unmarshal(res.Result, &nodeTmp); err != nil {
			return fmt.Errorf("unmarshal node error: %v", err)
		}
		if node_mgr.NVPNode != nodeTmp.NodeType {
			return fmt.Errorf("the node is not a nvp node: %s", string(nodeTmp.NodeType))
		}
	default:
		return fmt.Errorf("Registration for %s is not supported currently", role.RoleType)
	}

	return nil
}

func (rm *RoleManager) roleKey(id string) string {
	return fmt.Sprintf("%s-%s", ROLEPREFIX, id)
}
