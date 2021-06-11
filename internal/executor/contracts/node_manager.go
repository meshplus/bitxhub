package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
	"github.com/sirupsen/logrus"
)

type NodeManager struct {
	boltvm.Stub
	node_mgr.NodeManager
}

// extra: nodeMgr.Node
func (nm *NodeManager) Manage(eventTyp string, proposalResult, lastStatus string, extra []byte) *boltvm.Response {
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + err.Error())
	}
	res := nm.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSpecific)),
		pb.String(""),
		pb.String(nm.CurrentCaller()),
		pb.Bytes(addrsData))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	nm.NodeManager.Persister = nm.Stub
	node := &node_mgr.Node{}
	if err := json.Unmarshal(extra, node); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal json error: %v", err))
	}

	if proposalResult == string(APPOVED) {
		switch eventTyp {
		case string(governance.EventLogout), string(governance.EventRegister):
			nodeEvent := &events.NodeEvent{
				NodeId:        node.Id,
				NodeEventType: governance.EventType(eventTyp),
			}
			nm.PostEvent(pb.Event_NODEMGR, nodeEvent)
		}
	}

	ok, errData := nm.NodeManager.ChangeStatus(strconv.Itoa(int(node.Id)), proposalResult, lastStatus, nil)
	if !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(errData)))
	}

	return boltvm.Success(nil)
}

// Register registers node info
// caller is the bitxhub admin address
// return node id, proposal id and error
func (nm *NodeManager) RegisterNode(nodeId int64, nodePid, nodeAccount, nodeType string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	// 1. check permission
	res := nm.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionAdmin)),
		pb.String(strconv.Itoa(int(nodeId))),
		pb.String(nm.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	// 2. check info
	node := &node_mgr.Node{
		Id:       uint64(nodeId),
		Pid:      nodePid,
		Account:  nodeAccount,
		NodeType: node_mgr.NodeType(nodeType),
		Primary:  false,
		Status:   governance.GovernanceUnavailable,
	}
	if err := nm.checkNodeInfo(node); err != nil {
		return boltvm.Error(fmt.Sprintf("check node info error: %s", err.Error()))
	}

	// 3. store information
	nodeData, err := json.Marshal(node)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("marshal node error: %v", err))
	}
	nm.NodeManager.Logger().WithFields(logrus.Fields{
		"node.id": node.Id,
		"nodeId":  nodeId,
	}).Info("=====> register")

	ok, data := nm.NodeManager.Register(nodeData)
	if !ok {
		return boltvm.Error(fmt.Sprintf("register error: %s", string(data)))
	}

	// 4. submit proposal
	res = nm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(nm.Caller()),
		pb.String(string(governance.EventRegister)),
		pb.String(""),
		pb.String(string(NodeMgr)),
		pb.String(strconv.Itoa(int(node.Id))),
		pb.String(string(node.Status)),
		pb.Bytes(nodeData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 5. change status
	if ok, data := nm.NodeManager.ChangeStatus(strconv.Itoa(int(nodeId)), string(governance.EventRegister), string(node.Status), nil); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s, %d", string(data), nodeId))
	}
	return getGovernanceRet(string(res.Result), []byte(strconv.Itoa(int(nodeId))))
}

// LogoutNode logout available node
func (nm *NodeManager) LogoutNode(nodeId int64) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub

	// 1. check permission
	res := nm.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionAdmin)),
		pb.String(strconv.Itoa(int(nodeId))),
		pb.String(nm.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("check permission error: %s", string(res.Result)))
	}

	// 2. check status
	if ok, data := nm.NodeManager.GovernancePre(strconv.Itoa(int(nodeId)), governance.EventLogout, nil); !ok {
		return boltvm.Error(fmt.Sprintf("logout prepare error: %s", string(data)))
	}

	// 3. check node num
	ok, nodeData := nm.NodeManager.QueryById(strconv.Itoa(int(nodeId)), nil)
	if !ok {
		return boltvm.Error(string(nodeData))
	}
	node := &node_mgr.Node{}
	if err := json.Unmarshal(nodeData, node); err != nil {
		return boltvm.Error(err.Error())
	}
	if node.NodeType == node_mgr.VPNode {
		// 3.1 don't support delete primary vp node
		if node.Primary {
			return boltvm.Error(fmt.Sprintf("don't support logout primary vp node"))
		}
		// 3.2 don't support delete node when there're only 4 vp nodes
		res = nm.CountAvailableNodes(string(node_mgr.VPNode))
		if !res.Ok {
			return boltvm.Error(fmt.Sprintf("count available nodes error: %s", string(res.Result)))
		}
		if string(res.Result) <= "4" {
			return boltvm.Error(fmt.Sprintf("don't support delete node when there're only %s vp nodes", string(res.Result)))
		}
		// 3.3 only support delete last vp node
		// TODO: solve it
		if strconv.Itoa(int(node.Id)) != string(res.Result) {
			return boltvm.Error(fmt.Sprintf("only support delete last vp node currently"))
		}

	}

	// 4. submit proposal
	res = nm.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(nm.Caller()),
		pb.String(string(governance.EventLogout)),
		pb.String(""),
		pb.String(string(NodeMgr)),
		pb.String(strconv.Itoa(int(node.Id))),
		pb.String(string(node.Status)),
		pb.Bytes(nodeData),
	)
	if !res.Ok {
		return boltvm.Error(fmt.Sprintf("submit proposal error: %s", string(res.Result)))
	}

	// 4. change status
	if ok, data := nm.NodeManager.ChangeStatus(strconv.Itoa(int(nodeId)), string(governance.EventLogout), string(node.Status), nil); !ok {
		return boltvm.Error(fmt.Sprintf("change status error: %s", string(data)))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// CountAvailableVPNodes counts all available node
func (nm *NodeManager) CountAvailableNodes(nodeType string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	return responseWrapper(nm.NodeManager.CountAvailable([]byte(nodeType)))
}

// CountNodes counts all nodes
func (nm *NodeManager) CountNodes(nodeType string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	return responseWrapper(nm.NodeManager.CountAll([]byte(nodeType)))
}

// Nodes returns all nodes
func (nm *NodeManager) Nodes(nodeType string) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	return responseWrapper(nm.NodeManager.All([]byte(nodeType)))
}

// IsAvailable returns whether the node is available
func (nm *NodeManager) IsAvailable(nodeId int64) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	ok, data := nm.NodeManager.QueryById(strconv.Itoa(int(nodeId)), nil)
	if !ok {
		return boltvm.Success([]byte(strconv.FormatBool(false)))
	}

	node := &node_mgr.Node{}
	if err := json.Unmarshal(data, node); err != nil {
		return boltvm.Error(fmt.Sprintf("unmarshal error: %v", err))
	}

	for _, s := range node_mgr.NodeAvailableState {
		if node.Status == s {
			return boltvm.Success([]byte(strconv.FormatBool(true)))
		}
	}

	return boltvm.Success([]byte(strconv.FormatBool(false)))
}

// GetNode returns node info by node id
func (nm *NodeManager) GetNode(nodeId int64) *boltvm.Response {
	nm.NodeManager.Persister = nm.Stub
	return responseWrapper(nm.NodeManager.QueryById(strconv.Itoa(int(nodeId)), nil))
}

func (nm *NodeManager) checkNodeInfo(node *node_mgr.Node) error {
	// 1. check node id
	nm.NodeManager.Persister = nm.Stub
	ok, data := nm.NodeManager.CountAvailable([]byte(node.NodeType))
	if !ok {
		return fmt.Errorf("count all error: %s", string(data))
	}
	if strconv.Itoa(int(node.Id)-1) != string(data) {
		return fmt.Errorf("node id is illegal (current id: %s)", string(data))
	}

	// 2. check node Pid
	id, err := nm.NodeManager.GetIdByPid(node.Pid)
	// 2.1 not exist
	if err != nil {
		return nil
	}
	// 2.2 exist && available
	ok, data = nm.NodeManager.QueryById(id, nil)
	if !ok {
		return fmt.Errorf("QueryById error: %s", string(data))
	}
	nodeInfo := &node_mgr.Node{}
	if err := json.Unmarshal(data, nodeInfo); err != nil {
		return fmt.Errorf("unmarshal error: %v", err)
	}
	for _, s := range node_mgr.NodeAvailableState {
		if nodeInfo.Status == s {
			return fmt.Errorf("node pid is already occupied")
		}
	}

	return nil
}
