package contracts

import (
	"encoding/json"
	"testing"

	"github.com/meshplus/bitxhub-kit/log"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/stretchr/testify/assert"
)

var (
	NODEPID     = "QmWjeMdhS3L244WyFJGfasU4wDvaZfLTC7URq8aKxWvKmk"
	NODEACCOUNT = "0x9150264e20237Cb2693aa9896e1Ca671e52AF7FD"
)

func TestNodeManager_RegisterNode(t *testing.T) {
	nm, mockStub, nodes, nodesData := nodePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("CheckPermission error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[1]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[5]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nodesData[0]).AnyTimes()
	mockStub.EXPECT().Get(NODEPID).Return(true, nil).AnyTimes()

	// 1. CheckPermission error
	res := nm.RegisterNode(1, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	assert.False(t, res.Ok, string(res.Result))
	// 2. info(id) error
	res = nm.RegisterNode(1, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	assert.False(t, res.Ok, string(res.Result))
	// 3. info(pid) error
	res = nm.RegisterNode(4, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	assert.False(t, res.Ok, string(res.Result))
	// 4. SubmitProposal error
	res = nm.RegisterNode(4, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	assert.False(t, res.Ok, string(res.Result))

	res = nm.RegisterNode(4, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	res = nm.RegisterNode(6, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	assert.False(t, res.Ok, string(res.Result))
	// 4. SubmitProposal error
	res = nm.RegisterNode(6, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	assert.False(t, res.Ok, string(res.Result))

	res = nm.RegisterNode(6, NODEPID, NODEACCOUNT, string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_LogoutNode(t *testing.T) {
	nm, mockStub, nodes, nodesData := nodePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("CheckPermission error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[1]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nodesData[0]).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nodesData[4]).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).Times(1)
	var nodesData2 [][]byte
	nodesData2 = append(nodesData2, nodesData[0], nodesData[0], nodesData[0], nodesData[0], nodesData[0])
	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData2).AnyTimes()
	var nodesData2 [][]byte
	nodesData2 = append(nodesData2, nodesData[0])
	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData2).Times(1)
	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).AnyTimes()

	// 1. CheckPermission error
	res := nm.LogoutNode(1)
	assert.False(t, res.Ok, string(res.Result))
	// 2. status error
	res = nm.LogoutNode(1)
	assert.False(t, res.Ok, string(res.Result))
	// 3. QueryById error
	res = nm.LogoutNode(1)
	assert.False(t, res.Ok, string(res.Result))
	// 4. check num error
	res = nm.LogoutNode(1)
	assert.False(t, res.Ok, string(res.Result))
	// 5. SubmitProposal error
	res = nm.LogoutNode(1)
	assert.False(t, res.Ok, string(res.Result))

	res = nm.LogoutNode(1)
	res = nm.LogoutNode(5)
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_Manage(t *testing.T) {
	nm, mockStub, nodes, nodesData := nodePrepare(t)

	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("CheckPermission error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[1]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[2]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *nodes[6]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()

	// register, CheckPermission error
	res := nm.Manage(string(governance.EventRegister), BallotApprove, string(governance.GovernanceUnavailable), nodesData[1])
	assert.False(t, res.Ok, string(res.Result))
	// register, ChangeStatus error
	res = nm.Manage(string(governance.EventRegister), BallotApprove, string(governance.GovernanceUnavailable), nodesData[1])
	assert.False(t, res.Ok, string(res.Result))

	res = nm.Manage(string(governance.EventRegister), BallotApprove, string(governance.GovernanceUnavailable), nodesData[1])
	res = nm.Manage(string(governance.EventRegister), BallotApprove, string(governance.GovernanceUnavailable), nodesData[6])
	assert.True(t, res.Ok, string(res.Result))
}

func TestNodeManager_VPNodeQuery(t *testing.T) {
	nm, mockStub, _, nodesData := nodePrepare(t)

	mockStub.EXPECT().Query(node_mgr.NODEPREFIX).Return(true, nodesData).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nodesData[0]).AnyTimes()

	res := nm.CountAvailableNodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "1", string(res.Result))

	res = nm.CountNodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "3", string(res.Result))
	assert.Equal(t, "5", string(res.Result))

	res = nm.CountNodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "7", string(res.Result))

	res = nm.Nodes(string(node_mgr.VPNode))
	assert.True(t, res.Ok, string(res.Result))

	res = nm.IsAvailable(1)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "false", string(res.Result))
	res = nm.IsAvailable(1)
	assert.True(t, res.Ok, string(res.Result))
	assert.Equal(t, "true", string(res.Result))

	res = nm.GetNode(1)
	assert.True(t, res.Ok, string(res.Result))
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
	nodeStatus := []string{string(governance.GovernanceAvailable), string(governance.GovernanceUnavailable), string(governance.GovernanceRegisting)}

	for i := 0; i < 3; i++ {
		node := &node_mgr.Node{
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
			Id:       uint64(i + 1),
			Pid:      NODEPID,
			Account:  NODEACCOUNT,
			NodeType: node_mgr.VPNode,
			Status:   governance.GovernanceStatus(nodeStatus[i]),
		}

		data, err := json.Marshal(node)
		assert.Nil(t, err)

		nodesData = append(nodesData, data)
		nodes = append(nodes, node)
	}

	return nm, mockStub, nodes, nodesData
}
