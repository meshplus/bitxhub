package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	nodemgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
)

type NodeManager struct {
	boltvm.Stub
	nodemgr.NodeManager
}

type UpdateNodeInfo struct {
	NodeName   UpdateInfo    `json:"node_name"`
	Permission UpdateMapInfo `json:"permission"`
}

const (
	MinimumVPNode = 4
)

func (nm *NodeManager) checkPermission(permissions []string, nodeAccount, regulatorAddr string, specificAddrsData []byte) error {
	nm.NodeManager.Persister = nm.Stub

	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
			node, err := nm.NodeManager.QueryById(nodeAccount, nil)
			if err != nil {
				return err
			}
			nodeInfo := node.(*nodemgr.Node)
			if regulatorAddr == nodeInfo.AuditAdminAddr {
				return nil
			}
		case string(PermissionAdmin):
			res := nm.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin",
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

func (nm *NodeManager) occupyNodeName(name string, nodeAccount string) {
	nm.NodeManager.Persister = nm.Stub
	nm.SetObject(nodemgr.NodeOccupyNameKey(name), nodeAccount)
}

func (nm *NodeManager) freeNodeName(name string) {
	nm.NodeManager.Persister = nm.Stub
	nm.Delete(nodemgr.NodeOccupyNameKey(name))
}

func (nm *NodeManager) isOccupiedName(name string) (bool, string) {
	nm.NodeManager.Persister = nm.Stub
	id := ""
	return nm.GetObject(nodemgr.NodeOccupyNameKey(name), &id), id
}

func (nm *NodeManager) occupyNodePid(pid string, nodeAccount string) {
	nm.NodeManager.Persister = nm.Stub
	nm.SetObject(nodemgr.NodeOccupyPidKey(pid), nodeAccount)
}

func (nm *NodeManager) freeNodePid(pid string) {
	nm.NodeManager.Persister = nm.Stub
	nm.Delete(nodemgr.NodeOccupyPidKey(pid))
}

func (nm *NodeManager) isOccupiedPid(pid string) (bool, string) {
	nm.NodeManager.Persister = nm.Stub
	id := ""
	return nm.GetObject(nodemgr.NodeOccupyPidKey(pid), &id), id
}

func (nm *NodeManager) hasVpNodeGoverned() bool {
	nm.NodeManager.Persister = nm.Stub
	nodesTmp, err := nm.NodeManager.All(nil)
	if err != nil {
		return false
	}
	nodes := nodesTmp.([]*nodemgr.Node)

	for _, node := range nodes {
		if node.NodeType == nodemgr.NVPNode {
			continue
		}
		if node.Status == governance.GovernanceRegisting || node.Status == governance.GovernanceLogouting {
			return true
		}
	}
	return false
}

// =========== Manage does some subsequent operations when the proposal is over
// other operation:
// - register - approve : register info
// - register - reject : free occupation info
// - update(only nvp) - approve : 1. update info; 2. if the bound audit admin is unavailable, need to unbind the node
// - update(only nvp) - reject : 1. free occupation info; 2. if the bound audit admin is unavailable, need to unbind the node
// - logout - approve - vp : post logout event
// - logout - reject - vp : no other operation
// - logout - approve - nvp : 1. reject the proposal for the role registration or binding if the bound audit admin has; 2. pause the role (going into the unavailable or frozen state)
// - logout - reject - nvp : 1. restore the proposal for the role registration or binding if the bound audit admin has; 2. if the bound audit admin is unavailable, need to unbind the node; 3. if the audit admin which would be bound is unavailable, need to reject binding proposal
func (nm *NodeManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := nm.checkPermission([]string{string(PermissionSpecific)}, objId, nm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), fmt.Sprintf("check permission error:%v", err)))
	}

	// 2. change status
	ok, errData := nm.NodeManager.ChangeStatus(objId, proposalResult, lastStatus, nil)
	if !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(errData))))
	}

	node, err := nm.NodeManager.QueryById(objId, nil)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	}
	nodeInfo := node.(*nodemgr.Node)

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventRegister):
			nm.NodeManager.Register(nodeInfo)
		case string(governance.EventUpdate):
			updateInfo := &UpdateNodeInfo{}
			if err := json.Unmarshal(extra, updateInfo); err != nil {
				return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("unmarshal update data error:%v", err)))
			}
			updateNode := &nodemgr.Node{
				Account:     objId,
				Name:        updateInfo.NodeName.NewInfo.(string),
				Permissions: updateInfo.Permission.NewInfo,
			}
			if ok, data := nm.NodeManager.Update(updateNode); !ok {
				return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("update node error: %s", string(data))))
			}

			if updateInfo.NodeName.IsEdit {
				nm.freeNodeName(updateInfo.NodeName.OldInfo.(string))
			}

			if nodeInfo.NodeType == nodemgr.NVPNode && nodeInfo.AuditAdminAddr != "" {
				res := nm.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(nodeInfo.AuditAdminAddr), pb.String(string(AuditAdmin)))
				if !res.Ok {
					return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("cross invoke IsAnyAvailableAdmin error: %s", string(res.Result))))
				}
				if string(res.Result) == FALSE {
					if res := nm.unbindNode(nodeInfo.Account); !res.Ok {
						return res
					}
				}
			}
		case string(governance.EventLogout):
			switch nodeInfo.NodeType {
			case nodemgr.VPNode:
				nodeEvent := &events.NodeEvent{
					NodeId:        nodeInfo.VPNodeId,
					NodeEventType: governance.EventType(eventTyp),
				}
				nm.PostEvent(pb.Event_NODEMGR, nodeEvent)
			case nodemgr.NVPNode:
				nm.CrossInvoke(constant.RoleContractAddr.Address().String(), "PauseAuditAdmin", pb.String(objId))
			}
		}
	} else {
		switch eventTyp {
		case string(governance.EventRegister):
			if res := nm.CrossInvoke(constant.RoleContractAddr.String(), "FreeAccount", pb.String(nodeInfo.Account)); !res.Ok {
				return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("cross invoke FreeAccount error: %s", string(res.Result))))
			}
			switch nodeInfo.NodeType {
			case nodemgr.VPNode:
				nm.freeNodePid(nodeInfo.Pid)
			case nodemgr.NVPNode:
				nm.freeNodeName(nodeInfo.Name)
			}
		case string(governance.EventUpdate):
			nodeUpdateInfo := &UpdateNodeInfo{}
			if err := json.Unmarshal(extra, nodeUpdateInfo); err != nil {
				return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("unmarshal node error: %v", err)))
			}
			if nodeUpdateInfo.NodeName.IsEdit {
				nm.freeNodeName(nodeUpdateInfo.NodeName.NewInfo.(string))
			}

			if nodeInfo.NodeType == nodemgr.NVPNode && nodeInfo.AuditAdminAddr != "" {
				res := nm.CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(nodeInfo.AuditAdminAddr), pb.String(string(AuditAdmin)))
				if !res.Ok {
					return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("cross invoke IsAnyAvailableAdmin error: %s", string(res.Result))))
				}
				if string(res.Result) == FALSE {
					if res := nm.unbindNode(nodeInfo.Account); !res.Ok {
						return res
					}
				}
			}
		case string(governance.EventLogout):
			if nodeInfo.NodeType == nodemgr.NVPNode {
				if nodeInfo.AuditAdminAddr != "" {
					res := nm.CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleInfoById", pb.String(nodeInfo.AuditAdminAddr))
					if !res.Ok {
						return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("cross invoke GetRoleInfoById error: %s", string(res.Result))))
					}
					role := &Role{}
					if err := json.Unmarshal(res.Result, role); err != nil {
						return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.RoleInternalErrMsg), fmt.Sprintf("unmarshal role error: %v", err)))
					}

					nm.Logger().WithFields(logrus.Fields{
						"account":    nodeInfo.Account,
						"auditAdmin": nodeInfo.AuditAdminAddr,
						"roleStatus": role.Status,
					}).Info("nvp node is logout reject")
					if role.Status == governance.GovernanceForbidden {
						if res := nm.unbindNode(nodeInfo.Account); !res.Ok {
							return res
						}
					} else if role.Status == governance.GovernanceBinding || role.Status == governance.GovernanceRegisting {
						if res := nm.CrossInvoke(constant.RoleContractAddr.Address().String(), "RestoreAuditAdminBinding", pb.String(nodeInfo.Account)); !res.Ok {
							return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("cross invoke RestoreAuditAdminBinding error: %s", string(res.Result))))
						}
					}
				}
			}
		}
	}

	if err = nm.postAuditNodeEvent(objId); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("post audit node event error: %v", err)))
	}
	return boltvm.Success(nil)
}

// =========== RegisterNode registers node info, returns proposal id and error
// Only the unavailable state can be registered, need to occupy info such as the accout, name, pid
func (nm *NodeManager) RegisterNode(nodeAccount, nodeType, nodePid string, nodeVpId uint64, nodeName, permitStr, reason string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventRegister

	// 1. check permission: PermissionAdmin
	if err := nm.checkPermission([]string{string(PermissionAdmin)}, nodeAccount, nm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), fmt.Sprintf("check permission error:%v", err)))
	}

	// 2. check vp node
	if nodeType == string(nodemgr.VPNode) && nm.hasVpNodeGoverned() {
		return boltvm.Error(boltvm.NodeVPBeingGovernedCode, fmt.Sprintf(string(boltvm.NodeVPBeingGovernedMsg)))
	}

	// 3. check info
	permits := make(map[string]struct{})
	if permitStr != "" {
		for _, addr := range strings.Split(permitStr, ",") {
			permits[addr] = struct{}{}
		}
	}
	node := &nodemgr.Node{
		Account:     nodeAccount,
		NodeType:    nodemgr.NodeType(nodeType),
		Pid:         nodePid,
		VPNodeId:    nodeVpId,
		Primary:     false,
		Name:        nodeName,
		Permissions: permits,
		Status:      governance.GovernanceUnavailable,
	}
	if nodeType == string(nodemgr.NVPNode) {
		node.VPNodeId = 0
	}
	if res := nm.checkNodeInfo(node, true); !res.Ok {
		return res
	}

	// 4. governancePre: check status
	if _, be := nm.NodeManager.GovernancePre(nodeAccount, event, nil); be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 5. pre store registration information (name,pid)
	if res := nm.CrossInvoke(constant.RoleContractAddr.String(), "OccupyAccount", pb.String(nodeAccount), pb.String(string(NodeAccount))); !res.Ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("cross invoke OccupyAccount error: %s", string(res.Result))))
	}
	switch nodemgr.NodeType(nodeType) {
	case nodemgr.VPNode:
		nm.occupyNodePid(nodePid, nodeAccount)
	case nodemgr.NVPNode:
		nm.occupyNodeName(nodeName, nodeAccount)
	}

	// 6. submit proposal
	res := nm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(nm.Caller()),
		pb.String(string(event)),
		pb.String(string(NodeMgr)),
		pb.String(node.Account),
		pb.String(string(node.Status)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 7. register info
	node.Status = governance.GovernanceRegisting
	nm.NodeManager.RegisterPre(node)

	nm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := nm.postAuditNodeEvent(nodeAccount); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("post audit node event error: %v", err)))
	}

	return getGovernanceRet(string(res.Result), []byte(node.Account))
}

// =========== LogoutNode logouts node
// Logout scenarios in different node states:
// - unavailable: cannot log out without successfully registering
// - registering: cannot log out without successfully registering
// - available(vp) : logout if consensus conditions are met
// - available(nvp) : logout according to the normal management process
// - binding(only nvp): when submitting a proposal, 1. need to pause the proposal for the role registration or binding
//                      when proposal approved, if bound role is still registering or binding, 1. need to reject the proposal for the role registration or binding; 2. pause the role (going into the unavailable or frozen state)
//                                              if bound role is forbidden, 1. need to reject the proposal for the role registration or binding; 2. pause the role (not change state)
//                      when proposal rejected, if bound role is still registering or binding, 1. need to restore the proposal for the role registration or binding
//                                              if bound role is forbidden, 1. need to reject the proposal for the role registration or binding;
// - binded(only nvp): when submitting a proposal, no additional operations
//                     when proposal approved, if bound role is available 1. need to pause the role (going into the frozen state)
//                                             if bound role is forbidden 1. need to pause the role (not change state)
//                     when proposal rejected, if bound role is available, no additional operation
//                                             if bound role is forbidden, unbind node
// - updating(only nvp): when submitting a proposal, no additional operations
//                       when proposal approved, if bound role is available, 1. need to pause the role (going into the frozen state)
//                                               if bound role is forbidden, 1. need to pause the role (not change state)
//                       when proposal rejected, if bound role is available, no additional operation
//                                               if bound role is forbidden, unbind node
// - logouting: logout cannot be logged out again
// - forbidden: logout cannot be logged out again
func (nm *NodeManager) LogoutNode(nodeAccount, reason string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventLogout

	// 1. check permission: PermissionAdmin
	if err := nm.checkPermission([]string{string(PermissionAdmin)}, nodeAccount, nm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), err.Error()))
	}

	// 2. governancePre: check status
	nodeInfo, be := nm.NodeManager.GovernancePre(nodeAccount, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	node := nodeInfo.(*nodemgr.Node)

	// 3. check vp node
	if node.NodeType == nodemgr.VPNode && nm.hasVpNodeGoverned() {
		return boltvm.Error(boltvm.NodeVPBeingGovernedCode, fmt.Sprintf(string(boltvm.NodeVPBeingGovernedMsg)))
	}

	// 4. check node num
	if node.NodeType == nodemgr.VPNode {
		// 3.1 don't support delete primary vp node
		if node.Primary {
			return boltvm.Error(boltvm.NodeLogoutPrimaryNodeCode, fmt.Sprintf(string(boltvm.NodeLogoutPrimaryNodeMsg), node.Account))
		}
		// 3.2 don't support delete node when there're only 4 vp nodes
		ok, data := nm.NodeManager.CountAvailable([]byte(nodemgr.VPNode))
		if !ok {
			return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("count available nodes error: %s", string(data))))
		}

		vpNum, err := strconv.Atoi(string(data))
		if err != nil {
			return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("get vp node num error: %v", err)))
		}
		if vpNum <= MinimumVPNode {
			return boltvm.Error(boltvm.NodeLogoutTooFewNodeCode, fmt.Sprintf(string(boltvm.NodeLogoutTooFewNodeMsg), string(data)))
		}
		// 3.3 only support delete last vp node
		// TODO: solve it
		//if strconv.Itoa(int(node.VPNodeId)) != string(data) {
		//	return boltvm.Error(boltvm.NodeLogoutWrongIdNodeCode, fmt.Sprintf(string(boltvm.NodeLogoutWrongIdNodeMsg), string(data)))
		//}
	}

	// 5. submit proposal
	res := nm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(nm.Caller()),
		pb.String(string(event)),
		pb.String(string(NodeMgr)),
		pb.String(node.Account),
		pb.String(string(node.Status)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 6. change status
	if ok, data := nm.NodeManager.ChangeStatus(nodeAccount, string(event), string(node.Status), nil); !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	// 7. If the node is being bound, need to pause the binding proposal.
	if node.Status == governance.GovernanceBinding {
		if res := nm.CrossInvoke(constant.RoleContractAddr.Address().String(), "PauseAuditAdminBinding", pb.String(nodeAccount)); !res.Ok {
			return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("cross invoke PauseAuditAdminBinding error: %s", string(res.Result))))
		}
	}

	nm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := nm.postAuditNodeEvent(nodeAccount); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("post audit node event error: %v", err)))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// =========== UpdateNode updates audit node
func (nm *NodeManager) UpdateNode(nodeAccount, nodeName, permitStr, reason string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventUpdate

	// 1. check permission: PermissionAdmin, PermissionSelf
	if err := nm.checkPermission([]string{string(PermissionAdmin), string(PermissionSelf)}, nodeAccount, nm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), err.Error()))
	}

	// 2. governancePre: check status
	nodeInfo, be := nm.NodeManager.GovernancePre(nodeAccount, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	node := nodeInfo.(*nodemgr.Node)

	// 3. check node type
	if node.NodeType == nodemgr.VPNode {
		return boltvm.Error(boltvm.NodeUpdateVPNodeCode, fmt.Sprintf(string(boltvm.NodeUpdateVPNodeMsg), nodeAccount))
	}

	// 4. check node info
	permits := make(map[string]struct{})
	if permitStr != "" {
		for _, addr := range strings.Split(permitStr, ",") {
			permits[addr] = struct{}{}
		}
	}
	newNode := &nodemgr.Node{
		Account:     node.Account,
		NodeType:    node.NodeType,
		Name:        nodeName,
		Permissions: permits,
	}
	if res := nm.checkNodeInfo(newNode, false); !res.Ok {
		return res
	}

	// 5. pre store update information (name,)
	updatePermission := false
	if len(node.Permissions) != len(newNode.Permissions) {
		updatePermission = true
	} else {
		for permit, _ := range newNode.Permissions {
			if _, ok := node.Permissions[permit]; !ok {
				updatePermission = true
				break
			}
		}
	}
	updateNodeInfo := &UpdateNodeInfo{
		NodeName: UpdateInfo{
			OldInfo: node.Name,
			NewInfo: newNode.Name,
			IsEdit:  node.Name != newNode.Name,
		},
		Permission: UpdateMapInfo{
			OldInfo: node.Permissions,
			NewInfo: newNode.Permissions,
			IsEdit:  updatePermission,
		},
	}
	updateNodeInfoData, err := json.Marshal(updateNodeInfo)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("marshal updateNodeInfo error: %v", err)))
	}
	if updateNodeInfo.NodeName.IsEdit {
		nm.occupyNodeName(nodeName, nodeAccount)
	}
	// nothing update
	if !updateNodeInfo.NodeName.IsEdit && !updateNodeInfo.Permission.IsEdit {
		return getGovernanceRet("", nil)
	}

	// 6. submit proposal
	res := nm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(nm.Caller()),
		pb.String(string(event)),
		pb.String(string(NodeMgr)),
		pb.String(node.Account),
		pb.String(string(node.Status)),
		pb.String(reason),
		pb.Bytes(updateNodeInfoData),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 5. change status
	if ok, data := nm.NodeManager.ChangeStatus(nodeAccount, string(event), string(node.Status), nil); !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	nm.CrossInvoke(constant.GovernanceContractAddr.Address().String(), "ZeroPermission", pb.String(string(res.Result)))

	if err := nm.postAuditNodeEvent(nodeAccount); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("post audit node event error: %v", err)))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// =========== BindNode binds audit node to audit admin
func (nm *NodeManager) BindNode(nodeAccount, auditAdminAddr string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventBind

	// 1. check permission: PermissionSpecific
	specificAddrs := []string{constant.RoleContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := nm.checkPermission([]string{string(PermissionSpecific)}, nodeAccount, nm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), err.Error()))
	}

	// 2. governancePre: check status
	nodeInfo, be := nm.NodeManager.GovernancePre(nodeAccount, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	node := nodeInfo.(*nodemgr.Node)

	// 3. check node info
	if node.NodeType == nodemgr.VPNode {
		return boltvm.Error(boltvm.NodeBindVPNodeCode, fmt.Sprintf(string(boltvm.NodeBindVPNodeMsg), nodeAccount))
	}

	// 4. record node bind info
	if ok, data := nm.NodeManager.Bind(nodeAccount, auditAdminAddr); !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("record bind admin error: %s", string(data))))
	}

	// 5. change status
	if ok, data := nm.NodeManager.ChangeStatus(nodeAccount, string(event), string(node.Status), nil); !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	if err := nm.postAuditNodeEvent(nodeAccount); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("post audit node event error: %v", err)))
	}

	return getGovernanceRet("", nil)
}

func (nm *NodeManager) ManageBindNode(nodeAccount, auditAdminAddr, resultEvent string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventType(resultEvent)

	// 1. check permission: PermissionSpecific
	specificAddrs := []string{constant.RoleContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := nm.checkPermission([]string{string(PermissionSpecific)}, nodeAccount, nm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), err.Error()))
	}

	// 2. change status
	if ok, data := nm.NodeManager.ChangeStatus(nodeAccount, string(event), string(governance.GovernanceAvailable), nil); !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	// 3. record bind admin
	if event == governance.EventReject {
		if ok, data := nm.NodeManager.Bind(nodeAccount, ""); !ok {
			return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("record bind admin error: %s", string(data))))
		}
	}

	if err := nm.postAuditNodeEvent(nodeAccount); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("post audit node event error: %v", err)))
	}

	return getGovernanceRet("", nil)
}

// =========== UnbindNode unbinds audit node with audit admin
func (nm *NodeManager) UnbindNode(nodeAccount string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	// check permission: PermissionSpecific
	specificAddrs := []string{constant.RoleContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := nm.checkPermission([]string{string(PermissionSpecific)}, nodeAccount, nm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), err.Error()))
	}

	return nm.unbindNode(nodeAccount)
}

func (nm *NodeManager) unbindNode(nodeAccount string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventUnbind

	// 1. governancePre: check status
	nodeInfo, be := nm.NodeManager.GovernancePre(nodeAccount, event, nil)
	if be != nil {
		// If the audit node cannot be unbind, you do not need to unbind the audit node. The possibilities are as follows:
		// - binded: ok
		// - registering /unavailable/available: The audit node is not yet bound, so it is not possible to enter this method.
		// - binding: If the node is not bound successfully, it is equivalent to that the audit node is not bound. Therefore, this method cannot be entered.
		// - updating/logouting: When the update or logout event ends, the system checks whether the audit admin is logouted. If so, the system automatically unbinds the audit admin
		// - forbidden: There is no need to unbind.
		return boltvm.Success(nil)
	}
	node := nodeInfo.(*nodemgr.Node)

	// 2. check node info
	// It doesn't actually go there, because vpNode has no binded state and was returned in the previous state check
	if node.NodeType == nodemgr.VPNode {
		return boltvm.Error(boltvm.NodeUnbindVPNodeCode, fmt.Sprintf(string(boltvm.NodeUnbindVPNodeMsg), nodeAccount))
	}

	// 3. change status
	if ok, data := nm.NodeManager.ChangeStatus(nodeAccount, string(event), string(node.Status), nil); !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	if err := nm.postAuditNodeEvent(nodeAccount); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("post audit node event error: %v", err)))
	}

	return getGovernanceRet("", nil)
}

func (nm *NodeManager) postAuditNodeEvent(nodeAccount string) error {
	nm.NodeManager.Persister = nm.Stub
	ok, nodeData := nm.Get(nodemgr.NodeKey(nodeAccount))
	if !ok {
		return fmt.Errorf("not found node %s", nodeAccount)
	}

	auditInfo := &pb.AuditRelatedObjInfo{
		AuditObj:           nodeData,
		RelatedChainIDList: map[string][]byte{},
		RelatedNodeIDList: map[string][]byte{
			nodeAccount: {},
		},
	}

	nm.PostEvent(pb.Event_AUDIT_NODE, auditInfo)

	return nil
}

// ========================== Query interface ========================
func (nm *NodeManager) GetNextVpID() *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	nextVpID, err := nm.getNextVpID()
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err))
	}

	return boltvm.Success([]byte(strconv.Itoa(nextVpID)))
}

func (nm *NodeManager) getNextVpID() (int, error) {
	nm.NodeManager.Persister = nm.Stub

	nodes, err := nm.NodeManager.All(nil)
	if err != nil {
		return 0, fmt.Errorf("get all node error: %v", err)
	}

	maxId := 0
	for _, node := range nodes.([]*nodemgr.Node) {
		if node.IsAvailable() && node.NodeType == nodemgr.VPNode && int(node.VPNodeId) > maxId {
			maxId = int(node.VPNodeId)
		}
	}

	return maxId + 1, nil
}

// CountAvailableNodes counts all available node
func (nm *NodeManager) CountAvailableNodes(nodeType string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	return boltvm.ResponseWrapper(nm.NodeManager.CountAvailable([]byte(nodeType)))
}

// CountNodes counts all nodes
func (nm *NodeManager) CountNodes(nodeType string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	return boltvm.ResponseWrapper(nm.NodeManager.CountAll([]byte(nodeType)))
}

// Nodes returns all nodes
func (nm *NodeManager) Nodes() *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	nodes, err := nm.NodeManager.All(nil)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	}

	if data, err := json.Marshal(nodes.([]*nodemgr.Node)); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}

}

// IsAvailable returns whether the node is available
func (nm *NodeManager) IsAvailable(nodeAccount string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	node, err := nm.NodeManager.QueryById(nodeAccount, nil)
	if err != nil {
		return boltvm.Success([]byte(FALSE))
	}

	return boltvm.Success([]byte(strconv.FormatBool(node.(*nodemgr.Node).IsAvailable())))
}

// GetNode returns node info by node id
func (nm *NodeManager) GetNode(nodeAccount string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	node, err := nm.NodeManager.QueryById(nodeAccount, nil)
	if err != nil {
		return boltvm.Error(boltvm.NodeNonexistentNodeCode, fmt.Sprintf(string(boltvm.NodeNonexistentNodeMsg), nodeAccount))
	}
	if data, err := json.Marshal(node.(*nodemgr.Node)); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

func (nm *NodeManager) GetNvpNodeByName(nodeName string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	nodeAccount, err := nm.NodeManager.GetAccountByName(nodeName)
	if err != nil {
		return boltvm.Error(boltvm.NodeNonexistentNodeCode, fmt.Sprintf(string(boltvm.NodeNonexistentNodeMsg), nodeName))
	}

	node, err := nm.NodeManager.QueryById(nodeAccount, nil)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("node name %s exist but node %s not exist: %v", nodeName, nodeAccount, err)))
	}
	if data, err := json.Marshal(node.(*nodemgr.Node)); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

func (nm *NodeManager) GetVpNodeByPid(nodePid string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	nodeAccount, err := nm.NodeManager.GetAccountByPid(nodePid)
	if err != nil {
		return boltvm.Error(boltvm.NodeNonexistentNodeCode, fmt.Sprintf(string(boltvm.NodeNonexistentNodeMsg), nodePid))
	}

	node, err := nm.NodeManager.QueryById(nodeAccount, nil)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("node pid %s exist but node %s not exist: %v", nodePid, nodeAccount, err)))
	}
	if data, err := json.Marshal(node.(*nodemgr.Node)); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

func (nm *NodeManager) GetVpNodeByVpId(nodeVpId string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	nodeAccount, err := nm.NodeManager.GetAccountByVpId(nodeVpId)
	if err != nil {
		return boltvm.Error(boltvm.NodeNonexistentNodeCode, fmt.Sprintf(string(boltvm.NodeNonexistentNodeMsg), nodeVpId))
	}

	node, err := nm.NodeManager.QueryById(nodeAccount, nil)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("node vpId %s exist but node %s not exist: %v", nodeVpId, nodeAccount, err)))
	}
	if data, err := json.Marshal(node.(*nodemgr.Node)); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

func (nm *NodeManager) checkNodeInfo(node *nodemgr.Node, isRegister bool) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	// 1. check account
	_, err := types.HexDecodeString(node.Account)
	if err != nil {
		return boltvm.Error(boltvm.NodeIllegalAccountCode, fmt.Sprintf(string(boltvm.NodeIllegalAccountMsg), node.Account, err.Error()))
	}
	if isRegister {
		res := nm.CrossInvoke(constant.RoleContractAddr.String(), "CheckOccupiedAccount", pb.String(node.Account))
		if !res.Ok {
			return boltvm.Error(boltvm.NodeDuplicateAccountCode, fmt.Sprintf(string(boltvm.NodeDuplicateAccountMsg), node.Account, string(res.Result)))
		}
	}

	// 2. check noed type
	switch node.NodeType {
	case nodemgr.VPNode:
		// 3. check vp node id
		nextVpID, err := nm.getNextVpID()
		if err != nil {
			return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err))
		}
		if int(node.VPNodeId) < nextVpID {
			return boltvm.Error(boltvm.NodeIllegalVpIdCode, fmt.Sprintf(string(boltvm.NodeIllegalVpIdMsg), node.VPNodeId, fmt.Sprintf("(The id must be larger than that of all current nodes. The minimum id can be %d", nextVpID)))
		}

		// 4. check node Pid
		if node.Pid == "" {
			return boltvm.Error(boltvm.NodeEmptyPidCode, string(boltvm.NodeEmptyPidMsg))
		}
		if ok, nodeAccount := nm.isOccupiedPid(node.Pid); ok {
			return boltvm.Error(boltvm.NodeDuplicatePidCode, fmt.Sprintf(string(boltvm.NodeDuplicatePidMsg), node.Pid, nodeAccount))
		}
	case nodemgr.NVPNode:
		// 5. check name
		if node.Name == "" {
			return boltvm.Error(boltvm.NodeEmptyNameCode, string(boltvm.NodeEmptyNameMsg))
		}
		if ok, nodeAccount := nm.isOccupiedName(node.Name); ok {
			if isRegister {
				return boltvm.Error(boltvm.NodeDuplicateNameCode, fmt.Sprintf(string(boltvm.NodeDuplicateNameMsg), node.Name, string(nodeAccount)))
			} else if nodeAccount != node.Account {
				return boltvm.Error(boltvm.NodeDuplicateNameCode, fmt.Sprintf(string(boltvm.NodeDuplicateNameMsg), node.Name, nodeAccount))
			}
		}

		// 6. check permission
		if len(node.Permissions) == 0 {
			return boltvm.Error(boltvm.NodeEmptyPermissionCode, fmt.Sprintf(string(boltvm.NodeEmptyPermissionMsg)))
		}
		for p, _ := range node.Permissions {
			if res := nm.CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", pb.String(p)); !res.Ok {
				return boltvm.Error(boltvm.NodeIllegalPermissionCode, fmt.Sprintf(string(boltvm.NodeIllegalPermissionMsg), p, string(res.Result)))
			}
		}
	default:
		return boltvm.Error(boltvm.NodeIllegalNodeTypeCode, fmt.Sprintf(string(boltvm.NodeIllegalNodeTypeMsg), string(node.NodeType)))
	}

	return boltvm.Success(nil)
}
