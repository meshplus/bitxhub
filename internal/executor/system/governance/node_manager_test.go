package governance

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/ledger"
	vm "github.com/axiomesh/eth-kit/evm"
	ethledger "github.com/axiomesh/eth-kit/ledger"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNodeManager_Run(t *testing.T) {
	nm := NewNodeManager(logrus.New())

	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	ld, err := leveldb.New(filepath.Join(repoRoot, "node_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())

	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()
	initializeNode(t, stateLedger, []*NodeMember{
		{
			NodeId: "16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
		},
	})
	nm.Reset(stateLedger)

	// gabi, err := GetABI()
	// assert.Nil(t, err)

	testcases := []struct {
		Caller   string
		Data     []byte
		Expected vm.ExecutionResult
		Err      error
	}{
		{
			Caller: admin1,
			Data: generateNodeAddProposeData(t, NodeExtraArgs{
				Nodes: []*NodeMember{
					{
						NodeId: "16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
					},
				},
			}),
			Expected: vm.ExecutionResult{
				UsedGas: NodeProposalGas,
			},
			Err: nil,
		},
	}

	res, err := nm.Run(&vm.Message{
		Data: testcases[0].Data,
	})

	assert.Nil(t, err)

	assert.Equal(t, uint64(1000), res.UsedGas)
}

func initializeNode(t *testing.T, lg ethledger.StateLedger, admins []*NodeMember) {
	node := &Node{}
	node.Members = admins
	account := lg.GetOrCreateAccount(types.NewAddressByStr(common.NodeMemberContractAddr))
	b, err := json.Marshal(node)
	assert.Nil(t, err)
	account.SetState([]byte(common.NodeMemberContractAddr), b)
}

func TestRunForNodePropose(t *testing.T) {

	logger := logrus.New()
	nm := NewNodeManager(logger)

	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	ld, err := leveldb.New(filepath.Join(repoRoot, "node_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeMemberContractAddr), ledger.NewChanger())

	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()

	initializeNode(t, stateLedger, []*NodeMember{
		{
			NodeId: "16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
		},
	})
	nm.Reset(stateLedger)

	testcases := []struct {
		Caller   string
		Data     []byte
		Expected vm.ExecutionResult
		Err      error
	}{
		{
			Caller: admin1,
			Data: generateNodeAddProposeData(t, NodeExtraArgs{
				Nodes: []*NodeMember{
					{
						NodeId: "16Uiu2HAmJ38LwfY6pfgDWNvk3ypjcpEMSePNTE6Ma2NCLqjbZJSF",
					},
				},
			}),
			Expected: vm.ExecutionResult{
				UsedGas: NodeProposalGas,
			},
			Err: nil,
		},
	}

	for _, test := range testcases {
		result, err := nm.Run(&vm.Message{
			From: types.NewAddressByStr(test.Caller).ETHAddress(),
			Data: test.Data,
		})

		assert.Equal(t, test.Err, err)

		if result != nil {
			assert.Equal(t, nil, result.Err)
			assert.Equal(t, test.Expected.UsedGas, result.UsedGas)
		}
	}
}

func generateNodeAddProposeData(t *testing.T, extraArgs NodeExtraArgs) []byte {
	gabi, err := GetABI()

	title := "title"
	desc := "desc"
	blockNumber := uint64(1000)
	extra, err := json.Marshal(extraArgs)
	assert.Nil(t, err)
	data, err := gabi.Pack(ProposeMethod, uint8(NodeAdd), title, desc, blockNumber, extra)
	assert.Nil(t, err)
	return data
}

func generateNodeAddVoteData(t *testing.T, proposalID uint64, voteResult VoteResult) []byte {
	gabi, err := GetABI()

	data, err := gabi.Pack(ProposeMethod, uint8(NodeAdd), "title", "desc", uint64(1000), []byte(""))
	assert.Nil(t, err)
	res, err := nm.Run(&vm.Message{
		Data: data,
	})
	assert.Nil(t, err)

	return data
}

func TestNodeManager_EstimateGas(t *testing.T) {
	nm := NewNodeManager(logrus.New())

	gabi, err := GetABI()
	assert.Nil(t, err)

	data, err := gabi.Pack(ProposeMethod, uint8(NodeAdd), "title", "desc", uint64(1000), []byte(""))
	assert.Nil(t, err)

	from := types.NewAddressByStr(admin1).ETHAddress()
	to := types.NewAddressByStr(common.NodeManagerContractAddr).ETHAddress()
	dataBytes := hexutil.Bytes(data)

	// test propose
	gas, err := nm.EstimateGas(&types.CallArgs{
		From: &from,
		To:   &to,
		Data: &dataBytes,
	})
	assert.Nil(t, err)
	assert.Equal(t, NodeManagementProposalGas, gas)

	// test vote
	data, err = gabi.Pack(VoteMethod, uint64(1), uint8(Pass), []byte(""))
	dataBytes = hexutil.Bytes(data)
	assert.Nil(t, err)
	gas, err = nm.EstimateGas(&types.CallArgs{
		From: &from,
		To:   &to,
		Data: &dataBytes,
	})
	assert.Nil(t, err)
	assert.Equal(t, NodeManagementVoteGas, gas)
}
