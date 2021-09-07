package contracts

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/iancoleman/orderedmap"
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
type Permission string

const (
	GenesisBalance = "genesis_balance"

	ROLEPREFIX          = "role"
	ROLE_TYPE_PREFIX    = "type"
	APPCHAINADMINPREFIX = "appchain"

	GovernanceAdmin      RoleType = "governanceAdmin"
	SuperGovernanceAdmin RoleType = "superGovernanceAdmin"
	AuditAdmin           RoleType = "auditAdmin"
	AppchainAdmin        RoleType = "appchainAdmin"
	NoRole               RoleType = "none"

	PermissionSelf     Permission = "PermissionSelf"
	PermissionAdmin    Permission = "PermissionAdmin"
	PermissionSpecific Permission = "PermissionSpecific"
)

type Role struct {
	ID       string   `toml:"id" json:"id"`
	RoleType RoleType `toml:"role_type" json:"role_type"`

	// 	GovernanceAdmin info
	Weight uint64 `json:"weight" toml:"weight"`

	// AuditAdmin info
	NodePid string `toml:"pid" json:"pid"`

	// Appchain info
	AppchainID string `toml:"appchain_id" json:"appchain_id"`

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

var roleAvailableMap = map[governance.GovernanceStatus]struct{}{
	governance.GovernanceAvailable: {},
	governance.GovernanceFreezing:  {},
}

func (r *Role) IsAvailable() bool {
	if _, ok := roleAvailableMap[r.Status]; ok {
		return true
	} else {
		return false
	}
}

func (role *Role) setFSM(lastStatus governance.GovernanceStatus) {
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

// GovernancePre checks if the role can do the event. (only check, not modify infomation)
func (rm *RoleManager) governancePre(roleId string, event governance.EventType, chainID string) (*Role, error) {
	role := Role{}
	if chainID != "" {
		// check if the admin for the appchain is registered
		chainAdminID := ""
		if ok := rm.GetObject(AppchainAdminKey(chainID), &chainAdminID); ok {
			return nil, fmt.Errorf("admin for appchain(%s) has registered: %s", chainID, role.ID)
		}

		return nil, nil
	}

	if ok := rm.GetObject(RoleKey(roleId), &role); !ok {
		if event == governance.EventRegister {
			return nil, nil
		} else {
			return nil, fmt.Errorf("this role does not exist")
		}
	}

	for _, s := range roleStateMap[event] {
		if role.Status == s {
			return &role, nil
		}
	}

	return nil, fmt.Errorf("The role (%s,%s,%s) can not be %s", roleId, string(role.RoleType), string(role.Status), string(event))
}

func (rm *RoleManager) changeStatus(roleId string, trigger, lastStatus string) (bool, []byte) {
	role := &Role{}
	if ok := rm.GetObject(RoleKey(roleId), role); !ok {
		return false, []byte("this role does not exist")
	}

	role.setFSM(governance.GovernanceStatus(lastStatus))
	err := role.FSM.Event(trigger)
	if err != nil {
		return false, []byte(fmt.Sprintf("change status error: %v", err))
	}

	rm.SetObject(RoleKey(roleId), *role)
	return true, nil
}

func (rm *RoleManager) checkPermission(permissions []string, roleID string, regulatorAddr string, specificAddrsData []byte) error {
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			if regulatorAddr == roleID {
				return nil
			}
		case string(PermissionAdmin):
			if rm.isAvailableAdmin(regulatorAddr, GovernanceAdmin) {
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
// extra: update - role info
func (rm *RoleManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := rm.checkPermission([]string{string(PermissionSpecific)}, "", rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. change status
	ok, errData := rm.changeStatus(objId, proposalResult, lastStatus)
	if !ok {
		return boltvm.Error(fmt.Sprintf("change status error:%s", string(errData)))
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventUpdate):
			role := &Role{}
			if err := json.Unmarshal(extra, role); err != nil {
				return boltvm.Error(fmt.Sprintf("unmarshal json error:%v", err))
			}
			rm.SetObject(RoleKey(objId), *role)
		case string(governance.EventFreeze):
			fallthrough
		case string(governance.EventLogout):
			fallthrough
		case string(governance.EventActivate):
			if err := rm.updateRoleRelatedProposalInfo(objId, governance.EventType(eventTyp)); err != nil {
				return boltvm.Error(err.Error())
			}
		}
	}

	return boltvm.Success(nil)
}

// Update proposal information related to the administrator
func (rm *RoleManager) updateRoleRelatedProposalInfo(roleID string, eventTyp governance.EventType) error {
	res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "GetNotClosedProposals")
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
				res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "UpdateAvaliableElectorateNum", pb.String(p.Id), pb.Uint64(p.AvaliableElectorateNum))
				if !res.Ok {
					return fmt.Errorf("cross invoke UpdateAvaliableElectorateNum error: %s", string(res.Result))
				}
				break
			}
		}
	}
	return nil
}

// =========== RegisterRole registers role info, returns proposal id and error
func (rm *RoleManager) RegisterRole(roleId, roleType, nodePid, appchainID, reason string) *boltvm.Response {
	event := string(governance.EventRegister)

	// 1. check permission
	specificAddrs := []string{constant.RuleManagerContractAddr.Address().String(), constant.AppchainMgrContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal specificAddrs error: %v", err))
	}
	if err := rm.checkPermission([]string{string(PermissionAdmin), string(PermissionSpecific)}, roleId, rm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. check info
	role := &Role{
		ID:         roleId,
		RoleType:   RoleType(roleType),
		Weight:     repo.NormalAdminWeight,
		NodePid:    nodePid,
		AppchainID: appchainID,
		Status:     governance.GovernanceUnavailable,
	}
	if err := rm.checkRoleInfo(role); err != nil {
		return boltvm.Error(fmt.Sprintf("check node info error: %s", err.Error()))
	}

	// 3. check status
	if _, err := rm.governancePre(roleId, governance.EventType(event), appchainID); err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", event, err))
	}

	// 4. register
	rm.Logger().WithFields(logrus.Fields{
		"id":       role.ID,
		"roleType": role.RoleType,
	}).Info("Role is registering")
	roleIdMap := orderedmap.New()
	_ = rm.GetObject(RoleTypeKey(roleType), roleIdMap)
	roleIdMap.Set(roleId, struct{}{})
	rm.SetObject(RoleTypeKey(roleType), roleIdMap)
	switch roleType {
	case string(AppchainAdmin):
		role.Status = governance.GovernanceAvailable
		rm.SetObject(RoleKey(roleId), *role)
		rm.SetObject(AppchainAdminKey(appchainID), role.ID)
		return getGovernanceRet("", []byte(role.ID))
	case string(GovernanceAdmin):
		ok, gb := rm.Get(GenesisBalance)
		if !ok {
			return boltvm.Error("get genesis balance error")
		}
		balance, _ := new(big.Int).SetString(string(gb), 10)
		account := rm.GetAccount(role.ID)
		acc := account.(ledger.IAccount)
		acc.AddBalance(balance)
		fallthrough
	case string(AuditAdmin):
		rm.SetObject(RoleKey(roleId), *role)
		// 5. submit proposal
		res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
			pb.String(rm.Caller()),
			pb.String(event),
			pb.String(string(RoleMgr)),
			pb.String(role.ID),
			pb.String(string(role.Status)),
			pb.String(reason),
			pb.Bytes(nil),
		)
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
		}

		// 6. change status
		if ok, data := rm.changeStatus(role.ID, event, string(role.Status)); !ok {
			return boltvm.Error(fmt.Sprintf("change status error: %s, %s", string(data), role.ID))
		}
		return getGovernanceRet(string(res.Result), []byte(role.ID))
	default:
		return boltvm.Error(fmt.Sprintf("Registration for %s is not supported currently", role.RoleType))
	}
}

// =========== UpdateAuditAdminNode updates nodeId of nvp role
func (rm *RoleManager) UpdateAuditAdminNode(roleId, nodePid, reason string) *boltvm.Response {
	event := string(governance.EventUpdate)

	// 1. check permission
	if err := rm.checkPermission([]string{string(PermissionAdmin), string(PermissionSelf)}, roleId, rm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. check status
	role, err := rm.governancePre(roleId, governance.EventType(event), "")
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", event, err))
	}

	// 3. check info
	if AuditAdmin != role.RoleType {
		return boltvm.Error(fmt.Sprintf("the role is not a AuditAdmin: %s", string(role.RoleType)))
	}
	if nodePid == role.NodePid {
		return boltvm.Error(fmt.Sprintf("the node ID is the same as before: %s", nodePid))
	}

	res := rm.CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "GetNode", pb.String(nodePid))
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

	res = rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(event),
		pb.String(string(RoleMgr)),
		pb.String(roleId),
		pb.String(string(role.Status)),
		pb.String(reason),
		pb.Bytes(roleData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 5. change status
	if ok, data := rm.changeStatus(roleId, event, string(role.Status)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// =========== FreezeRole freezes role
func (rm *RoleManager) FreezeRole(roleId, reason string) *boltvm.Response {
	return rm.basicGovernance(roleId, reason, []string{string(PermissionAdmin)}, governance.EventFreeze)
}

// =========== ActivateRole updates frozen role
func (rm *RoleManager) ActivateRole(roleId, reason string) *boltvm.Response {
	return rm.basicGovernance(roleId, reason, []string{string(PermissionAdmin), string(PermissionSelf)}, governance.EventActivate)
}

// =========== LogoutRole logouts role
func (rm *RoleManager) LogoutRole(roleId, reason string) *boltvm.Response {
	return rm.basicGovernance(roleId, reason, []string{string(PermissionAdmin), string(PermissionSelf)}, governance.EventLogout)
}

func (rm *RoleManager) basicGovernance(roleID, reason string, permissions []string, event governance.EventType) *boltvm.Response {
	// 1. check permission
	if err := rm.checkPermission(permissions, roleID, rm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(fmt.Sprintf("check permission error:%v", err))
	}

	// 2. check status
	role, err := rm.governancePre(roleID, event, "")
	if err != nil {
		return boltvm.Error(fmt.Sprintf("%s prepare error: %v", string(event), err))
	}
	if role.Weight == repo.SuperAdminWeight {
		return boltvm.Error(fmt.Sprintf("super governance admin can not be %s", string(event)))
	}
	if role.RoleType == AppchainAdmin {
		return boltvm.Error(fmt.Sprintf("appchain admin can not be %s", string(event)))
	}

	// 3. submit proposal
	res := rm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal",
		pb.String(rm.Caller()),
		pb.String(string(event)),
		pb.String(string(RoleMgr)),
		pb.String(roleID),
		pb.String(string(role.Status)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 4. change status
	if ok, data := rm.changeStatus(roleID, string(event), string(role.Status)); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// ========================== Query interface ========================

// GetRole return the role of the caller
func (rm *RoleManager) GetRole() *boltvm.Response {
	res, err := rm.getRole(rm.Caller())
	if err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success([]byte(res))
	}
}

// GetRole return the role of the addr
func (rm *RoleManager) GetRoleByAddr(addr string) *boltvm.Response {
	res, err := rm.getRole(addr)
	if err != nil {
		return boltvm.Error(err.Error())
	} else {
		return boltvm.Success([]byte(res))
	}
}

func (rm *RoleManager) getRole(addr string) (string, error) {
	role := &Role{}
	ok := rm.GetObject(RoleKey(addr), role)
	if !ok {
		return string(NoRole), nil
	}

	switch role.RoleType {
	case GovernanceAdmin:
		if role.Weight == repo.SuperAdminWeight {
			return string(SuperGovernanceAdmin), nil
		} else {
			return string(GovernanceAdmin), nil
		}
	case AuditAdmin:
		return string(AuditAdmin), nil
	case AppchainAdmin:
		return string(AppchainAdmin), nil
	}
	return string(NoRole), nil
}

// GetRoleInfoById query a role info by roleId
func (rm *RoleManager) GetRoleInfoById(roleId string) *boltvm.Response {
	role := &Role{}
	ok := rm.GetObject(RoleKey(roleId), role)
	if !ok {
		return boltvm.Error("the role is not exist")
	}

	data, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(data)
}

// GetAllRoles query all roles
func (rm *RoleManager) GetAllRoles() *boltvm.Response {
	ret, err := rm.getAll()
	if err != nil {
		return boltvm.Error(err.Error())
	}
	if ret == nil {
		return boltvm.Success(nil)
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

func (rm *RoleManager) getAll() ([]*Role, error) {
	ret := make([]*Role, 0)

	ok, value := rm.Query(ROLEPREFIX)
	if ok {
		for _, data := range value {
			role := &Role{}
			if err := json.Unmarshal(data, role); err != nil {
				return nil, err
			}
			ret = append(ret, role)
		}
	}

	return ret, nil
}

func (rm *RoleManager) GetRolesByType(roleType string) *boltvm.Response {
	ret := make([]*Role, 0)

	roleIdMap := orderedmap.New()
	ok := rm.GetObject(RoleTypeKey(roleType), roleIdMap)
	if ok {
		for _, id := range roleIdMap.Keys() {
			role := Role{}
			ok := rm.GetObject(RoleKey(id), &role)
			if !ok {
				return boltvm.Error("the role is not exist")
			}
			ret = append(ret, &role)
		}
	}

	data, err := json.Marshal(ret)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(data)
}

func (rm *RoleManager) GetAppchainAdmin(appchainID string) *boltvm.Response {
	role := &Role{}
	appchainAdminID := ""
	ok := rm.GetObject(AppchainAdminKey(appchainID), &appchainAdminID)
	if !ok {
		return boltvm.Error("there is no admin for the appchain")
	}
	ok = rm.GetObject(RoleKey(appchainAdminID), role)
	if !ok {
		return boltvm.Error(fmt.Sprintf("the appchain admin is not exist: %s", appchainAdminID))
	}

	data, err := json.Marshal(role)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return boltvm.Success(data)
}

// IsAdmin determines whether the role is GovernanceAdmin
func (rm *RoleManager) IsAnyAdmin(roleId, roleType string) *boltvm.Response {
	return boltvm.Success([]byte(strconv.FormatBool(rm.isAdmin(roleId, RoleType(roleType)))))
}

func (rm *RoleManager) isAdmin(roleId string, roleType RoleType) bool {
	role := &Role{}
	ok := rm.GetObject(RoleKey(roleId), role)
	if !ok {
		return false
	}

	if roleType == role.RoleType {
		return true
	} else {
		if roleType == SuperGovernanceAdmin {
			if role.RoleType == GovernanceAdmin && role.Weight == repo.SuperAdminWeight {
				return true
			}
		}
		return false
	}
}

// IsAdmin determines whether the role is GovernanceAdmin
func (rm *RoleManager) IsAnyAvailableAdmin(roleId, roleType string) *boltvm.Response {
	return boltvm.Success([]byte(strconv.FormatBool(rm.isAvailableAdmin(roleId, RoleType(roleType)))))
}

func (rm *RoleManager) isAvailableAdmin(roleId string, roleType RoleType) bool {
	role := &Role{}
	ok := rm.GetObject(RoleKey(roleId), role)
	if !ok {
		return false
	}

	if roleType == role.RoleType && role.IsAvailable() {
		return true
	} else {
		if roleType == SuperGovernanceAdmin {
			if role.RoleType == GovernanceAdmin && role.Weight == repo.SuperAdminWeight && role.IsAvailable() {
				return true
			}
		}
		return false
	}
}

func (rm *RoleManager) GetRoleWeight(roleId string) *boltvm.Response {
	role := &Role{}
	ok := rm.GetObject(RoleKey(roleId), role)
	if !ok {
		return boltvm.Error("the role is not exist")
	}

	if role.RoleType != GovernanceAdmin {
		return boltvm.Error("the role is not governane admin")
	}

	return boltvm.Success([]byte(strconv.Itoa(int(role.Weight))))
}

// ======================================================================

func (rm *RoleManager) checkRoleInfo(role *Role) error {
	_, err := types.HexDecodeString(role.ID)
	if err != nil {
		return fmt.Errorf("illegal role id")
	}

	switch role.RoleType {
	case GovernanceAdmin:
	case AuditAdmin:
		res := rm.CrossInvoke(constant.NodeManagerContractAddr.Address().String(), "GetNode", pb.String(role.NodePid))
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
	case AppchainAdmin:
	default:
		return fmt.Errorf("Registration for %s is not supported currently", role.RoleType)
	}

	return nil
}

func RoleKey(id string) string {
	return fmt.Sprintf("%s-%s", ROLEPREFIX, id)
}

func RoleTypeKey(typ string) string {
	return fmt.Sprintf("%s-%s", ROLE_TYPE_PREFIX, typ)
}

func AppchainAdminKey(appchainID string) string {
	return fmt.Sprintf("%s-%s", APPCHAINADMINPREFIX, appchainID)
}
