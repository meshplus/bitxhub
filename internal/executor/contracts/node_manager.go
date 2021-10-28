package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	nodemgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/sirupsen/logrus"
)

type NodeManager struct {
	boltvm.Stub
	nodemgr.NodeManager
}

const (
	MinimumVPNode = 4
)

func (nm *NodeManager) checkPermission(permissions []string, regulatorAddr string, specificAddrsData []byte) error {
	for _, permission := range permissions {
		switch permission {
		case string(PermissionSelf):
		case string(PermissionAdmin):
			res := nm.CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin",
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
// extra: nil
func (nm *NodeManager) Manage(eventTyp, proposalResult, lastStatus, objId string, extra []byte) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	// 1. check permission: PermissionSpecific(GovernanceContractAddr)
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("marshal specificAddrs error: %v", err)))
	}
	if err := nm.checkPermission([]string{string(PermissionSpecific)}, nm.CurrentCaller(), addrsData); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), fmt.Sprintf("check permission error:%v", err)))
	}

	// 2. change status
	ok, errData := nm.NodeManager.ChangeStatus(objId, proposalResult, lastStatus, nil)
	if !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(errData))))
	}

	// 3. other operation
	if proposalResult == string(APPROVED) {
		switch eventTyp {
		case string(governance.EventLogout):
			node, err := nm.NodeManager.QueryById(objId, nil)
			if err != nil {
				return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
			}
			nodeInfo := node.(*nodemgr.Node)
			if nodemgr.VPNode == nodeInfo.NodeType {
				nodeEvent := &events.NodeEvent{
					NodeId:        nodeInfo.VPNodeId,
					NodeEventType: governance.EventType(eventTyp),
				}
				nm.PostEvent(pb.Event_NODEMGR, nodeEvent)
			}
		}
	}

	return boltvm.Success(nil)
}

// =========== RegisterNode registers node info, returns proposal id and error
func (nm *NodeManager) RegisterNode(nodePid string, nodeVpId uint64, nodeAccount, nodeType, reason string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventRegister

	// 1. check permission: PermissionAdmin
	if err := nm.checkPermission([]string{string(PermissionAdmin)}, nm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.NodeNoPermissionCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), fmt.Sprintf("check permission error:%v", err)))
	}

	// 2. check info
	node := &nodemgr.Node{
		Pid:      nodePid,
		VPNodeId: nodeVpId,
		Account:  nodeAccount,
		NodeType: nodemgr.NodeType(nodeType),
		Primary:  false,
		Status:   governance.GovernanceUnavailable,
	}
	res := nm.checkNodeInfo(node)
	if !res.Ok {
		return res
	}

	// 3. governancePre: check status
	if _, be := nm.NodeManager.GovernancePre(nodePid, event, nil); be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}

	// 4. submit proposal
	res = nm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(nm.Caller()),
		pb.String(string(event)),
		pb.String(string(NodeMgr)),
		pb.String(node.Pid),
		pb.String(string(node.Status)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 5. register info
	node.Status = governance.GovernanceRegisting

	ok, data := nm.NodeManager.Register(node)
	if !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("register error: %s", string(data))))
	}

	return getGovernanceRet(string(res.Result), []byte(node.Pid))
}

// =========== LogoutNode logout node
func (nm *NodeManager) LogoutNode(nodePid, reason string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	event := governance.EventLogout

	// 1. check permission: PermissionAdmin
	if err := nm.checkPermission([]string{string(PermissionAdmin)}, nm.CurrentCaller(), nil); err != nil {
		return boltvm.Error(boltvm.NodeNonexistentNodeCode, fmt.Sprintf(string(boltvm.NodeNoPermissionMsg), nm.CurrentCaller(), err.Error()))
	}

	// 2. governancePre: check status
	nodeInfo, be := nm.NodeManager.GovernancePre(nodePid, event, nil)
	if be != nil {
		return boltvm.Error(be.Code, string(be.Msg))
	}
	node := nodeInfo.(*nodemgr.Node)

	// 3. check node num
	if node.NodeType == nodemgr.VPNode {
		// 3.1 don't support delete primary vp node
		if node.Primary {
			return boltvm.Error(boltvm.NodeLogoutPrimaryNodeCode, fmt.Sprintf(string(boltvm.NodeLogoutPrimaryNodeMsg), node.Pid))
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
		if strconv.Itoa(int(node.VPNodeId)) != string(data) {
			return boltvm.Error(boltvm.NodeLogoutWrongIdNodeCode, fmt.Sprintf(string(boltvm.NodeLogoutWrongIdNodeMsg), string(data)))
		}
	}

	// 4. submit proposal
	res := nm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(nm.Caller()),
		pb.String(string(event)),
		pb.String(string(NodeMgr)),
		pb.String(node.Pid),
		pb.String(string(node.Status)),
		pb.String(reason),
		pb.Bytes(nil),
	)
	if !res.Ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("submit proposal error: %s", string(res.Result))))
	}

	// 5. change status
	if ok, data := nm.NodeManager.ChangeStatus(nodePid, string(event), string(node.Status), nil); !ok {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Sprintf("change status error: %s", string(data))))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// ========================== Query interface ========================

// CountAvailableVPNodes counts all available node
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
func (nm *NodeManager) IsAvailable(nodePid string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	node, err := nm.NodeManager.QueryById(nodePid, nil)
	if err != nil {
		return boltvm.Success([]byte(FALSE))
	}

	return boltvm.Success([]byte(strconv.FormatBool(node.(*nodemgr.Node).IsAvailable())))
}

// GetNode returns node info by node id
func (nm *NodeManager) GetNode(nodePid string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	node, err := nm.NodeManager.QueryById(nodePid, nil)
	if err != nil {
		return boltvm.Error(boltvm.NodeNonexistentNodeCode, fmt.Sprintf(string(boltvm.NodeNonexistentNodeMsg), nodePid))
	}
	if data, err := json.Marshal(node.(*nodemgr.Node)); err != nil {
		return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
	} else {
		return boltvm.Success(data)
	}
}

func (nm *NodeManager) checkNodeInfo(node *nodemgr.Node) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	// 1. check noed type
	switch node.NodeType {
	case nodemgr.VPNode:
	case nodemgr.NVPNode:
		node.VPNodeId = 0
	default:
		return boltvm.Error(boltvm.NodeIllegalNodeTypeCode, fmt.Sprintf(string(boltvm.NodeIllegalNodeTypeMsg), string(node.NodeType)))
	}

	// 2. check vp node id
	if node.NodeType == nodemgr.VPNode {
		ok, data := nm.NodeManager.CountAvailable([]byte(node.NodeType))
		nm.Logger().WithFields(logrus.Fields{
			"num": string(data),
		}).Debug("available node num")
		if !ok {
			return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), fmt.Errorf("count all error: %s", string(data))))
		}
		if strconv.Itoa(int(node.VPNodeId)-1) != string(data) {
			tmpId, err := strconv.Atoi(string(data))
			if err != nil {
				return boltvm.Error(boltvm.NodeInternalErrCode, fmt.Sprintf(string(boltvm.NodeInternalErrMsg), err.Error()))
			}
			return boltvm.Error(boltvm.NodeIllegalVpIdCode, fmt.Sprintf(string(boltvm.NodeIllegalVpIdMsg), node.VPNodeId, strconv.Itoa(tmpId+1)))
		}
	}

	// 3. check node Pid
	nodeInfo, err := nm.NodeManager.QueryById(node.Pid, nil)
	// 3.1 not exist
	if err != nil {
		return boltvm.Success(nil)
	}
	// 3.2 exist && available
	if nodeInfo.(*nodemgr.Node).Status != governance.GovernanceUnavailable {
		return boltvm.Error(boltvm.NodeDuplicatePidCode, fmt.Sprintf(string(boltvm.NodeDuplicatePidMsg), nodeInfo.(*nodemgr.Node).Pid))
	}

	return boltvm.Success(nil)
}
