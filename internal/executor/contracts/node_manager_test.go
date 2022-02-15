package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	NODE_PID         = "QmWjeMdhS3L244WyFJGfasU4wDvaZfLTC7URq8aKxWvKmk"
	NODE_ACCOUNT     = "0x9150264e20237Cb2693aa9896e1Ca671e52AF7FD"
	NVP_NODE_ACCOUNT = "0x8150264e20237Cb2693aa9896e1Ca671e52AF7FD"
	NODE_NAME        = "nodeName"
)

func TestNodeManager_RegisterNode(t *testing.T) {
	nm, mockStub, nodes, nodesData := vpNodePrepare(t)

	accountMap := orderedmap.New()
	accountMap.Set(nodes[0].Account, struct{}{})

	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *accountMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeOccupyPidKey(nodes[0].Pid), gomock.Any()).SetArg(1, nodes[0].Account).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeOccupyPidKey(nodes[5].Pid), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Account), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Account), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[1].Account), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "OccupyAccount",
		gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "CheckOccupiedAccount",
		gomock.Any()).Return(boltvm.Success([]byte(""))).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(1)).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, nodesData).Times(1)
	mockStub.EXPECT().Query(gomock.Any()).Return(false, nil).AnyTimes()

	// 1. CheckPermission error
	res := nm.RegisterNode(nodes[5].Account, string(node_mgr.VPNode), nodes[5].Pid, 1, NODE_NAME, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	// 2. a vp node is governed
	res = nm.RegisterNode(nodes[5].Account, string(node_mgr.VPNode), nodes[5].Pid, 1, NODE_NAME, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	// 3. info(pid) error
	res = nm.RegisterNode(nodes[5].Account, string(node_mgr.VPNode), nodes[0].Pid, 9, NODE_NAME, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	// 4. governance pre error
	res = nm.RegisterNode(nodes[1].Account, string(node_mgr.VPNode), nodes[5].Pid, 9, NODE_NAME, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	// 5. SubmitProposal error
	res = nm.RegisterNode(nodes[5].Account, string(node_mgr.VPNode), nodes[5].Pid, 9, NODE_NAME, "", reason)
	assert.False(t, res.Ok, string(res.Result))

	res = nm.RegisterNode(nodes[5].Account, string(node_mgr.VPNode), nodes[5].Pid, 9, NODE_NAME, "", reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_LogoutNode(t *testing.T) {
	nm, mockStub, nodes, nodesData := vpNodePrepare(t)

	cannotLogoutAccountMap := orderedmap.New()
	cannotLogoutAccountMap.Set(nodes[6].Account, struct{}{})
	accountMap := orderedmap.New()
	accountMap.Set(nodes[0].Account, struct{}{})
	accountMap.Set(nodes[1].Account, struct{}{})
	accountMap.Set(nodes[2].Account, struct{}{})
	accountMap.Set(nodes[3].Account, struct{}{})
	accountMap.Set(nodes[4].Account, struct{}{})
	accountMap.Set(nodes[5].Account, struct{}{})

	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *cannotLogoutAccountMap).Return(true).Times(1)
	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *accountMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Account), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[1].Account), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[2].Account), gomock.Any()).SetArg(1, *nodes[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[3].Account), gomock.Any()).SetArg(1, *nodes[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[4].Account), gomock.Any()).SetArg(1, *nodes[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Account), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[6].Account), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[7].Account), gomock.Any()).SetArg(1, *nodes[7]).Return(true).AnyTimes()

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()

	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.RoleContractAddr.Address().String()), gomock.Eq("PauseAuditAdminBinding"),
		gomock.Any()).Return(boltvm.Error("", "PauseAuditAdminBinding error")).Times(1)
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.RoleContractAddr.Address().String()), gomock.Eq("PauseAuditAdminBinding"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, nodesData).Times(1)
	mockStub.EXPECT().Query(gomock.Any()).Return(false, nil).AnyTimes()

	// 1. CheckPermission error
	res := nm.LogoutNode(nodes[1].Account, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 2. status error
	res = nm.LogoutNode(nodes[5].Account, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 3. a vp node is governed
	res = nm.LogoutNode(nodes[1].Account, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 4. primary error
	res = nm.LogoutNode(nodes[0].Account, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 5. 0 available node error
	res = nm.LogoutNode(nodes[1].Account, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 6. SubmitProposal error
	res = nm.LogoutNode(nodes[1].Account, reason)
	assert.False(t, res.Ok, string(res.Result))
	// 7. PauseAuditAdminBinding error
	res = nm.LogoutNode(nodes[7].Account, reason)
	assert.False(t, res.Ok, string(res.Result))

	// ok
	res = nm.LogoutNode(nodes[7].Account, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_UpdateNode(t *testing.T) {
	nm, mockStub, vpNodes, _ := vpNodePrepare(t)
	_, _, nvpNodes, _ := nvpNodePrepare(t)

	mockStub.EXPECT().GetObject(node_mgr.NodeOccupyNameKey(nvpNodes[2].Name), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeOccupyNameKey(nvpNodes[1].Name), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(adminAddr).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("", "")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.Address().String(), "GetAppchain", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[0].Account), gomock.Any()).SetArg(1, *nvpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(vpNodes[0].Account), gomock.Any()).SetArg(1, *vpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[1].Account), gomock.Any()).SetArg(1, *nvpNodes[1]).Return(true).AnyTimes()

	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()

	// 1. CheckPermission error
	res := nm.UpdateNode(nvpNodes[0].Account, nvpNodes[2].Name, appchainID, reason)
	assert.False(t, res.Ok, string(res.Result))

	// 2. status error(0: forbidden)
	res = nm.UpdateNode(nvpNodes[0].Account, nvpNodes[2].Name, appchainID, reason)
	assert.False(t, res.Ok, string(res.Result))

	// 3. node type error
	res = nm.UpdateNode(vpNodes[0].Account, nvpNodes[2].Name, appchainID, reason)
	assert.False(t, res.Ok, string(res.Result))

	// 4. check name error
	res = nm.UpdateNode(nvpNodes[1].Account, "", appchainID, reason)
	assert.False(t, res.Ok, string(res.Result))

	// 5. nothing update
	res = nm.UpdateNode(nvpNodes[1].Account, nvpNodes[1].Name, appchainID, reason)
	assert.True(t, res.Ok, string(res.Result))

	// 6. SubmitProposal error
	res = nm.UpdateNode(nvpNodes[1].Account, nvpNodes[2].Name, appchainID, reason)
	assert.False(t, res.Ok, string(res.Result))

	res = nm.UpdateNode(nvpNodes[1].Account, nvpNodes[2].Name, appchainID, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_BindNode(t *testing.T) {
	nm, mockStub, vpNodes, _ := vpNodePrepare(t)
	_, _, nvpNodes, _ := nvpNodePrepare(t)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RoleContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[0].Account), gomock.Any()).SetArg(1, *nvpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(vpNodes[0].Account), gomock.Any()).SetArg(1, *vpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[1].Account), gomock.Any()).SetArg(1, *nvpNodes[1]).Return(true).AnyTimes()

	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()

	// 1. CheckPermission error
	res := nm.BindNode(nvpNodes[0].Account, "")
	assert.False(t, res.Ok, string(res.Result))

	// 2. status error(0: forbidden)
	res = nm.BindNode(nvpNodes[0].Account, "")
	assert.False(t, res.Ok, string(res.Result))

	// 3. node type error
	res = nm.BindNode(vpNodes[0].Account, "")
	assert.False(t, res.Ok, string(res.Result))

	res = nm.BindNode(nvpNodes[1].Account, "")
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_ManageBindNode(t *testing.T) {
	nm, mockStub, vpNodes, _ := vpNodePrepare(t)
	_, _, nvpNodes, _ := nvpNodePrepare(t)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RoleContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[0].Account), gomock.Any()).SetArg(1, *nvpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(vpNodes[0].Account), gomock.Any()).SetArg(1, *vpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[2].Account), gomock.Any()).SetArg(1, *nvpNodes[2]).Return(true).AnyTimes()

	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()

	// 1. CheckPermission error
	res := nm.ManageBindNode(nvpNodes[0].Account, "", string(APPROVED))
	assert.False(t, res.Ok, string(res.Result))

	// 2. status error(0: forbidden)
	res = nm.ManageBindNode(nvpNodes[0].Account, "", string(APPROVED))
	assert.False(t, res.Ok, string(res.Result))

	// 3. node type error
	res = nm.ManageBindNode(vpNodes[0].Account, "", string(APPROVED))
	assert.False(t, res.Ok, string(res.Result))

	res = nm.ManageBindNode(nvpNodes[2].Account, "", string(APPROVED))
	assert.True(t, res.Ok, string(res.Result))

	res = nm.ManageBindNode(nvpNodes[2].Account, "", string(REJECTED))
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_UnbindNode(t *testing.T) {
	nm, mockStub, vpNodes, _ := vpNodePrepare(t)
	_, _, nvpNodes, _ := nvpNodePrepare(t)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.RoleContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	//mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[0].Account), gomock.Any()).SetArg(1, *nvpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(vpNodes[0].Account), gomock.Any()).SetArg(1, *vpNodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nvpNodes[3].Account), gomock.Any()).SetArg(1, *nvpNodes[3]).Return(true).AnyTimes()

	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()

	// 1. CheckPermission error
	res := nm.UnbindNode(nvpNodes[0].Account)
	assert.False(t, res.Ok, string(res.Result))

	// forbidden node
	res = nm.UnbindNode(nvpNodes[0].Account)
	assert.True(t, res.Ok, string(res.Result))

	// binded node
	res = nm.UnbindNode(nvpNodes[3].Account)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_Manage_VPNode(t *testing.T) {
	nm, mockStub, nodes, nodesData := vpNodePrepare(t)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[1]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[6]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "FreeAccount",
		gomock.Any()).Return(boltvm.Error("", "FreeAccount error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "FreeAccount",
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// CheckPermission error
	res := nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceUnavailable), nodes[1].Account, nodesData[1])
	assert.False(t, res.Ok, string(res.Result))
	// ChangeStatus error
	res = nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceUnavailable), nodes[1].Account, nodesData[1])
	assert.False(t, res.Ok, string(res.Result))
	// get role error
	res = nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceUnavailable), nodes[1].Account, nodesData[1])
	assert.False(t, res.Ok, string(res.Result))

	// register, reject
	// FreeAccount error
	res = nm.Manage(string(governance.EventRegister), BallotReject, string(governance.GovernanceUnavailable), nodes[1].Account, nodesData[6])
	assert.False(t, res.Ok, string(res.Result))
	// ok
	res = nm.Manage(string(governance.EventRegister), BallotReject, string(governance.GovernanceUnavailable), nodes[1].Account, nodesData[6])
	assert.True(t, res.Ok, string(res.Result))

	// logout, approve
	res = nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceUnavailable), nodes[1].Account, nodesData[6])
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_Manage_NVPNode(t *testing.T) {
	nm, mockStub, nodes, _ := nvpNodePrepare(t)

	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.String()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[4].Account), gomock.Any()).SetArg(1, *nodes[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Account), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[6].Account), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[7].Account), gomock.Any()).SetArg(1, *nodes[7]).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "PauseAuditAdmin", pb.String(nodes[6].Account)).Return(boltvm.Error("", "PauseAuditAdmin error")).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "FreeAccount",
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// register, approve
	res := nm.Manage(string(governance.EventRegister), BallotReject, string(governance.GovernanceUnavailable), nodes[4].Account, nil)
	assert.True(t, res.Ok, string(res.Result))

	// register, reject
	res = nm.Manage(string(governance.EventRegister), BallotReject, string(governance.GovernanceUnavailable), nodes[4].Account, nil)
	assert.True(t, res.Ok, string(res.Result))

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Error("", "IsAnyAvailableAdmin error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success([]byte(FALSE))).Times(1)

	// update, approve
	nodeUpdateInfo := &UpdateNodeInfo{
		NodeName: UpdateInfo{
			OldInfo: nodes[5].Name,
			NewInfo: nodes[4].Name,
			IsEdit:  true,
		},
		Permission: UpdateMapInfo{
			OldInfo: nodes[5].Permissions,
			NewInfo: nodes[5].Permissions,
			IsEdit:  false,
		},
	}
	nodeUpdateInfoData, err := json.Marshal(nodeUpdateInfo)
	assert.Nil(t, err)
	// IsAnyAvailableAdmin error
	res = nm.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), nodes[5].Account, nodeUpdateInfoData)
	assert.False(t, res.Ok, string(res.Result))
	// ok
	res = nm.Manage(string(governance.EventUpdate), BallotApprove, string(governance.GovernanceAvailable), nodes[5].Account, nodeUpdateInfoData)
	assert.True(t, res.Ok, string(res.Result))

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Error("", "IsAnyAvailableAdmin error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success([]byte(FALSE))).Times(1)

	// update, reject
	// IsAnyAvailableAdmin error
	res = nm.Manage(string(governance.EventUpdate), BallotReject, string(governance.GovernanceAvailable), nodes[5].Account, nodeUpdateInfoData)
	assert.False(t, res.Ok, string(res.Result))
	// ok
	res = nm.Manage(string(governance.EventUpdate), BallotReject, string(governance.GovernanceAvailable), nodes[5].Account, nodeUpdateInfoData)
	assert.True(t, res.Ok, string(res.Result))

	// logout, approve
	res = nm.Manage(string(governance.EventLogout), BallotApprove, string(governance.GovernanceAvailable), nodes[6].Account, nil)
	assert.True(t, res.Ok, string(res.Result))

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleInfoById", gomock.Any()).Return(boltvm.Error("", "GetRoleInfoById error")).Times(1)
	forbiddenRole := &Role{
		ID:     nodes[6].AuditAdminAddr,
		Status: governance.GovernanceForbidden,
	}
	forbiddenRoleData, err := json.Marshal(forbiddenRole)
	assert.Nil(t, err)
	bindingRole := &Role{
		ID:     nodes[6].AuditAdminAddr,
		Status: governance.GovernanceBinding,
	}
	bindingRoleData, err := json.Marshal(bindingRole)
	assert.Nil(t, err)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleInfoById", gomock.Any()).Return(boltvm.Success(forbiddenRoleData)).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "GetRoleInfoById", gomock.Any()).Return(boltvm.Success(bindingRoleData)).AnyTimes()

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "RestoreAuditAdminBinding", gomock.Any()).Return(boltvm.Error("", "RestoreAuditAdminBinding error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "RestoreAuditAdminBinding", gomock.Any()).Return(boltvm.Success(nil)).Times(1)

	// logout, reject
	// GetRoleInfoById error
	res = nm.Manage(string(governance.EventLogout), BallotReject, string(governance.GovernanceAvailable), nodes[6].Account, nil)
	assert.False(t, res.Ok, string(res.Result))
	// logout node with role which is forbidden
	res = nm.Manage(string(governance.EventLogout), BallotReject, string(governance.GovernanceAvailable), nodes[6].Account, nil)
	assert.True(t, res.Ok, string(res.Result))
	// logout node with role which is binding
	// RestoreAuditAdminBinding error
	res = nm.Manage(string(governance.EventLogout), BallotReject, string(governance.GovernanceAvailable), nodes[6].Account, nil)
	assert.False(t, res.Ok, string(res.Result))
	res = nm.Manage(string(governance.EventLogout), BallotReject, string(governance.GovernanceAvailable), nodes[6].Account, nil)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_NodeQuery(t *testing.T) {
	nm, mockStub, nodes, nodesData := vpNodePrepare(t)
	_, _, nvpNodes, _ := nvpNodePrepare(t)

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
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	res := nm.CountAvailableNodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "5", string(res.Result))

	res = nm.CountNodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "7", string(res.Result))

	res = nm.Nodes()
	assert.True(t, res.Ok, string(res.Result))

	res = nm.IsAvailable(NODE_PID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "false", string(res.Result))
	res = nm.IsAvailable(NODE_PID)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))

	res = nm.GetNode(NODE_PID)
	assert.True(t, res.Ok, string(res.Result))

	res = nm.GetNextVpID()
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "9", string(res.Result))

	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nodesData[0]).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[0]).Return(true).Times(1)

	res = nm.GetVpNodeByVpId(strconv.Itoa(int(nodes[0].VPNodeId)))
	assert.False(t, res.Ok, string(res.Result))
	res = nm.GetVpNodeByVpId(strconv.Itoa(int(nodes[0].VPNodeId)))
	assert.True(t, res.Ok, string(res.Result))

	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nodesData[0]).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[0]).Return(true).Times(1)

	res = nm.GetVpNodeByPid(nodes[0].Pid)
	assert.False(t, res.Ok, string(res.Result))
	res = nm.GetVpNodeByPid(nodes[0].Pid)
	assert.True(t, res.Ok, string(res.Result))

	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nodesData[0]).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[0]).Return(true).Times(1)

	res = nm.GetNvpNodeByName(nvpNodes[0].Name)
	assert.False(t, res.Ok, string(res.Result))
	res = nm.GetNvpNodeByName(nodes[0].Name)
	assert.True(t, res.Ok, string(res.Result))

}

func TestNodeManager_checkPermission(t *testing.T) {
	nm, mockStub, nodes, _ := nvpNodePrepare(t)

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(noAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(FALSE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(adminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Account), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()

	err := nm.checkPermission([]string{string(PermissionAdmin)}, "", adminAddr, nil)
	assert.Nil(t, err)
	err = nm.checkPermission([]string{string(PermissionSelf)}, "", noAdminAddr, nil)
	assert.NotNil(t, err)
	err = nm.checkPermission([]string{string(PermissionSelf)}, nodes[0].Account, nodes[0].AuditAdminAddr, nil)
	assert.Nil(t, err)

	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	err = nm.checkPermission([]string{string(PermissionSpecific)}, "", constant.GovernanceContractAddr.Address().String(), addrsData)
	assert.Nil(t, err)
	err = nm.checkPermission([]string{string(PermissionSpecific)}, "", noAdminAddr, addrsData)
	assert.NotNil(t, err)

	err = nm.checkPermission([]string{""}, "", "", nil)
	assert.NotNil(t, err)
}

func TestNodeManager_checkNodeInfo(t *testing.T) {
	nm, mockStub, nodes, nodesData := vpNodePrepare(t)

	accountMap := orderedmap.New()
	accountMap.Set(nodes[0].Account, struct{}{})
	accountMap.Set(nodes[1].Account, struct{}{})
	accountMap.Set(nodes[2].Account, struct{}{})
	accountMap.Set(nodes[3].Account, struct{}{})
	accountMap.Set(nodes[4].Account, struct{}{})
	accountMap.Set(nodes[5].Account, struct{}{})
	accountMap.Set(nodes[6].Account, struct{}{})

	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeTypeKey(string(node_mgr.VPNode)), gomock.Any()).SetArg(1, *accountMap).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[0].Account), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[1].Account), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[2].Account), gomock.Any()).SetArg(1, *nodes[2]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[3].Account), gomock.Any()).SetArg(1, *nodes[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[4].Account), gomock.Any()).SetArg(1, *nodes[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[5].Account), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(node_mgr.NodeKey(nodes[6].Account), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(NodeKey(NODE_ACCOUNT), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "CheckOccupiedAccount",
		gomock.Any()).Return(boltvm.Success([]byte(""))).AnyTimes()

	// check account error
	err := nm.checkNodeInfo(&node_mgr.Node{
		Account:  "124",
		Pid:      NODE_PID,
		NodeType: node_mgr.NVPNode,
	}, true)
	assert.NotNil(t, err)

	// check vpNodeID error
	err = nm.checkNodeInfo(&node_mgr.Node{
		Account:  NODE_ACCOUNT,
		Pid:      NODE_PID,
		NodeType: "",
		VPNodeId: 1,
	}, true)
	assert.NotNil(t, err)

	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).AnyTimes()
	err = nm.checkNodeInfo(&node_mgr.Node{
		Account:  NODE_ACCOUNT,
		Pid:      NODE_PID,
		NodeType: node_mgr.VPNode,
	}, true)
	assert.NotNil(t, err)
}

func vpNodePrepare(t *testing.T) (*NodeManager, *mock_stub.MockStub, []*node_mgr.Node, [][]byte) {
	// 1. prepare stub
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	nm := &NodeManager{
		Stub: mockStub,
	}

	// 2. prepare vp node
	var vpNodes []*node_mgr.Node
	var vpNodesData [][]byte
	nodeStatus := []string{
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceUnavailable),
		string(governance.GovernanceRegisting),
		string(governance.GovernanceBinding),
	}
	for i := 0; i < 8; i++ {
		node := &node_mgr.Node{
			VPNodeId: uint64(i + 1),
			Pid:      fmt.Sprintf("%s%d", NODE_PID[0:len(NODE_PID)-1], i),
			Account:  fmt.Sprintf("%s%d", NODE_ACCOUNT[0:len(NODE_ACCOUNT)-1], i),
			NodeType: node_mgr.VPNode,
			Status:   governance.GovernanceStatus(nodeStatus[i]),
		}

		if i == 0 {
			node.Primary = true
		}

		data, err := json.Marshal(node)
		assert.Nil(t, err)

		vpNodesData = append(vpNodesData, data)
		vpNodes = append(vpNodes, node)
	}

	return nm, mockStub, vpNodes, vpNodesData
}

func nvpNodePrepare(t *testing.T) (*NodeManager, *mock_stub.MockStub, []*node_mgr.Node, [][]byte) {
	// 1. prepare stub
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	nm := &NodeManager{
		Stub: mockStub,
	}

	// 2. prepare nvp node
	var nvpNodes []*node_mgr.Node
	var nvpNodesData [][]byte
	nvpNodeStatus := []string{
		string(governance.GovernanceForbidden),
		string(governance.GovernanceAvailable),
		string(governance.GovernanceBinding),
		string(governance.GovernanceBinded),
		string(governance.GovernanceRegisting),
		string(governance.GovernanceUpdating),
		string(governance.GovernanceLogouting),
		string(governance.GovernanceLogouting),
	}
	for i := 0; i < 8; i++ {
		node := &node_mgr.Node{
			Account: fmt.Sprintf("%s%d", NVP_NODE_ACCOUNT[0:len(NVP_NODE_ACCOUNT)-1], i),
			Name:    fmt.Sprintf("%s%d", NODE_NAME, i),
			Permissions: map[string]struct{}{
				appchainID: {},
			},
			NodeType:       node_mgr.NVPNode,
			Status:         governance.GovernanceStatus(nvpNodeStatus[i]),
			AuditAdminAddr: AUDIT_ADMIN_ROLE_ID,
		}
		if i == 7 {
			node.AuditAdminAddr = "111"
		}

		data, err := json.Marshal(node)
		assert.Nil(t, err)

		nvpNodesData = append(nvpNodesData, data)
		nvpNodes = append(nvpNodes, node)
	}

	return nm, mockStub, nvpNodes, nvpNodesData
}

func NodeKey(pid string) string {
	return fmt.Sprintf("%s-%s", node_mgr.NODEPREFIX, pid)
}
