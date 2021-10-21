package contracts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/iancoleman/orderedmap"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

var (
	NODEPID     = "QmWjeMdhS3L244WyFJGfasU4wDvaZfLTC7URq8aKxWvKmk"
	NODEACCOUNT = "0x9150264e20237Cb2693aa9896e1Ca671e52AF7FD"
)

func TestNodeManager_RegisterNode(t *testing.T) {
	nm, mockStub, nodes, nodesData := nodePrepare(t)

	idMap := orderedmap.New()
	idMap.Set(nodes[0].Pid, struct{}{})
	idMap.Set(nodes[1].Pid, struct{}{})
	idMap.Set(nodes[2].Pid, struct{}{})
	idMap.Set(nodes[3].Pid, struct{}{})
	idMap.Set(nodes[4].Pid, struct{}{})
	idMap.Set(nodes[5].Pid, struct{}{})
	idMap.Set(nodes[6].Pid, struct{}{})

	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Pid), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[1].Pid), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[2].Pid), gomock.Any()).SetArg(1, *nodes[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[3].Pid), gomock.Any()).SetArg(1, *nodes[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[4].Pid), gomock.Any()).SetArg(1, *nodes[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Pid), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[6].Pid), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[6]).Return(true).Times(1)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[5]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[6]).Return(true).Times(1)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).AnyTimes()

	// 1. CheckPermission error
	res := nm.RegisterNode(NODEPID, 1, NODEACCOUNT, string(node_mgr.VPNode), reason)
	assert.False(t, res.Ok, string(res.Result))
	// 2. info(id) error
	res = nm.RegisterNode(NODEPID, 1, NODEACCOUNT, string(node_mgr.VPNode), reason)
	assert.False(t, res.Ok, string(res.Result))
	// 3. info(pid) error
	res = nm.RegisterNode(NODEPID, 6, NODEACCOUNT, string(node_mgr.VPNode), reason)
	assert.False(t, res.Ok, string(res.Result))
	// 4. governance pre error
	res = nm.RegisterNode(NODEPID, 6, NODEACCOUNT, string(node_mgr.VPNode), reason)
	assert.False(t, res.Ok, string(res.Result))
	// 5. SubmitProposal error
	res = nm.RegisterNode(NODEPID, 6, NODEACCOUNT, string(node_mgr.VPNode), reason)
	assert.False(t, res.Ok, string(res.Result))

	res = nm.RegisterNode(NODEPID, 6, NODEACCOUNT, string(node_mgr.VPNode), reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_LogoutNode(t *testing.T) {
	nm, mockStub, nodes, _ := nodePrepare(t)

	idMap := orderedmap.New()
	idMap.Set(nodes[0].Pid, struct{}{})
	idMap.Set(nodes[1].Pid, struct{}{})
	idMap.Set(nodes[2].Pid, struct{}{})
	idMap.Set(nodes[3].Pid, struct{}{})
	idMap.Set(nodes[4].Pid, struct{}{})
	idMap.Set(nodes[5].Pid, struct{}{})
	idMap.Set(nodes[6].Pid, struct{}{})
	idMap1 := orderedmap.New()

	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *idMap1).Return(true).Times(1)
	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Pid), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[1].Pid), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[2].Pid), gomock.Any()).SetArg(1, *nodes[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[3].Pid), gomock.Any()).SetArg(1, *nodes[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[4].Pid), gomock.Any()).SetArg(1, *nodes[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Pid), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[6].Pid), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	primaryErrReq := mockStub.EXPECT().GetObject(NodeKey(NODEPID), gomock.Any()).SetArg(1, *nodes[0]).Return(true)
	primaryOkReq := mockStub.EXPECT().GetObject(NodeKey(NODEPID), gomock.Any()).SetArg(1, *nodes[4]).Return(true)
	//numErrReq := mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(false, nil)
	primaryOkReq1 := mockStub.EXPECT().GetObject(NodeKey(NODEPID), gomock.Any()).SetArg(1, *nodes[4]).Return(true)
	//numOkReq := mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData)
	primaryOkReq2 := mockStub.EXPECT().GetObject(NodeKey(NODEPID), gomock.Any()).SetArg(1, *nodes[4]).Return(true)
	//numOkReq1 := mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData)
	primaryOkReq3 := mockStub.EXPECT().GetObject(NodeKey(NODEPID), gomock.Any()).SetArg(1, *nodes[4]).Return(true)
	gomock.InOrder(governancePreErrReq, primaryErrReq, primaryOkReq, primaryOkReq1, primaryOkReq2, primaryOkReq3)

	// 1. CheckPermission error
	res := nm.LogoutNode(NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 2. status error
	res = nm.LogoutNode(NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 3. primary error
	res = nm.LogoutNode(NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 4. check num error
	res = nm.LogoutNode(NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 5. SubmitProposal error
	res = nm.LogoutNode(NODEPID, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = nm.LogoutNode(NODEPID, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_Manage(t *testing.T) {
	nm, mockStub, nodes, nodesData := nodePrepare(t)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[1]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()

	// register, CheckPermission error
	res := nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceUnavailable), nodes[1].Pid, nodesData[1])
	assert.False(t, res.Ok, string(res.Result))
	// register, ChangeStatus error
	res = nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceUnavailable), nodes[1].Pid, nodesData[1])
	assert.False(t, res.Ok, string(res.Result))

	res = nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceUnavailable), nodes[1].Pid, nodesData[6])
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_VPNodeQuery(t *testing.T) {
	nm, mockStub, nodes, nodesData := nodePrepare(t)

	idMap := orderedmap.New()
	idMap.Set(nodes[0].Pid, struct{}{})
	idMap.Set(nodes[1].Pid, struct{}{})
	idMap.Set(nodes[2].Pid, struct{}{})
	idMap.Set(nodes[3].Pid, struct{}{})
	idMap.Set(nodes[4].Pid, struct{}{})
	idMap.Set(nodes[5].Pid, struct{}{})
	idMap.Set(nodes[6].Pid, struct{}{})

	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Pid), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[1].Pid), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[2].Pid), gomock.Any()).SetArg(1, *nodes[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[3].Pid), gomock.Any()).SetArg(1, *nodes[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[4].Pid), gomock.Any()).SetArg(1, *nodes[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Pid), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[6].Pid), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[0]).Return(true).Times(2)

	res := nm.CountAvailableNodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "5", string(res.Result))

	res = nm.CountNodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "7", string(res.Result))

	res = nm.Nodes()
	assert.True(t, res.Ok, string(res.Result))

	res = nm.IsAvailable(NODEPID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "false", string(res.Result))
	res = nm.IsAvailable(NODEPID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))

	res = nm.GetNode(NODEPID)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_checkPermission(t *testing.T) {
	nm, mockStub, _, _ := nodePrepare(t)

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	err := nm.checkPermission([]string{string(PermissionAdmin)}, adminAddr, nil)
	assert.Nil(t, err)
	err = nm.checkPermission([]string{string(PermissionSelf)}, noAdminAddr, nil)
	assert.NotNil(t, err)

	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	err = nm.checkPermission([]string{string(PermissionSpecific)}, constant.GovernanceContractAddr.Address().String(), addrsData)
	assert.Nil(t, err)
	err = nm.checkPermission([]string{string(PermissionSpecific)}, noAdminAddr, addrsData)
	assert.NotNil(t, err)

	err = nm.checkPermission([]string{""}, "", nil)
	assert.NotNil(t, err)
}

func TestNodeManager_checkNodeInfo(t *testing.T) {
	nm, mockStub, nodes, nodesData := nodePrepare(t)

	idMap := orderedmap.New()
	idMap.Set(nodes[0].Pid, struct{}{})
	idMap.Set(nodes[1].Pid, struct{}{})
	idMap.Set(nodes[2].Pid, struct{}{})
	idMap.Set(nodes[3].Pid, struct{}{})
	idMap.Set(nodes[4].Pid, struct{}{})
	idMap.Set(nodes[5].Pid, struct{}{})
	idMap.Set(nodes[6].Pid, struct{}{})

	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *idMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Pid), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[1].Pid), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[2].Pid), gomock.Any()).SetArg(1, *nodes[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[3].Pid), gomock.Any()).SetArg(1, *nodes[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[4].Pid), gomock.Any()).SetArg(1, *nodes[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Pid), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[6].Pid), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(NodeKey(NODEPID), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	err := nm.checkNodeInfo(&node_mgr.Node{
		Pid:      NODEPID,
		NodeType: node_mgr.NVPNode,
	})
	assert.NotNil(t, err)

	err = nm.checkNodeInfo(&node_mgr.Node{
		Pid:      NODEPID,
		NodeType: "",
	})
	assert.NotNil(t, err)

	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).AnyTimes()
	err = nm.checkNodeInfo(&node_mgr.Node{
		Pid:      NODEPID,
		NodeType: node_mgr.VPNode,
	})
	assert.NotNil(t, err)
}

func nodePrepare(t *testing.T) (*NodeManager, *mock_stub.MockStub, []*node_mgr.Node, [][]byte) {
	// 1. prepare stub
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	nm := &NodeManager{
		Stub: mockStub,
	}

	// 2. prepare node
	var nodes []*node_mgr.Node
	var nodesData [][]byte
	nodeStatus := []string{
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceUnavailable),
		string(governance.GovernanceRegisting)}

	for i := 0; i < 7; i++ {
		node := &node_mgr.Node{
			VPNodeId: uint64(i + 1),
			Pid:      fmt.Sprintf("%s%d", NODEPID[0:len(NODEPID)-2], i),
			Account:  NODEACCOUNT,
			NodeType: node_mgr.VPNode,
			Status:   governance.GovernanceStatus(nodeStatus[i]),
		}

		if i == 0 {
			node.Primary = true
		}

		data, err := json.Marshal(node)
		assert.Nil(t, err)

		nodesData = append(nodesData, data)
		nodes = append(nodes, node)
	}

	return nm, mockStub, nodes, nodesData
}

func NodeKey(pid string) string {
	return fmt.Sprintf("%s-%s", node_mgr.NODEPREFIX, pid)
}
