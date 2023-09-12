package governance

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/pkg/repo"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
)

const (
	admin1 = "0x1210000000000000000000000000000000000000"
	admin2 = "0x1220000000000000000000000000000000000000"
	admin3 = "0x1230000000000000000000000000000000000000"
	admin4 = "0x1240000000000000000000000000000000000000"
)

type TestCouncilProposal struct {
	ID          uint64
	Type        ProposalType
	Proposer    string
	TotalVotes  uint64
	PassVotes   []string
	RejectVotes []string
	Status      ProposalStatus
	Candidates  []*CouncilMember
}

func TestRunForPropose(t *testing.T) {
	cm := NewCouncilManager(&common.SystemContractConfig{
		Logger: logrus.New(),
	})

	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	ld, err := leveldb.New(filepath.Join(repoRoot, "node_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())

	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()
	stateLedger.EXPECT().AddLog(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()

	err = InitCouncilMembers(stateLedger, []*repo.Admin{
		{
			Address: admin1,
			Weight:  1,
			Name:    "111",
		},
		{
			Address: admin2,
			Weight:  1,
			Name:    "222",
		},
		{
			Address: admin3,
			Weight:  1,
			Name:    "333",
		},
		{
			Address: admin4,
			Weight:  1,
			Name:    "444",
		},
	}, "10")
	assert.Nil(t, err)

	testcases := []struct {
		Caller   string
		Data     []byte
		Expected vm.ExecutionResult
		Err      error
	}{
		{
			Caller: admin1,
			Data: generateProposeData(t, CouncilExtraArgs{
				Candidates: []*CouncilMember{
					{
						Address: admin1,
						Weight:  1,
						Name:    "111",
					},
					{
						Address: admin2,
						Weight:  1,
						Name:    "222",
					},
					{
						Address: admin3,
						Weight:  1,
						Name:    "333",
					},
				},
			}),
			Expected: vm.ExecutionResult{},
			Err:      ErrMinCouncilMembersCount,
		},
		{
			Caller: admin1,
			Data: generateProposeData(t, CouncilExtraArgs{
				Candidates: []*CouncilMember{
					{
						Address: admin1,
						Weight:  1,
						Name:    "111",
					},
					{
						Address: admin1,
						Weight:  1,
						Name:    "222",
					},
					{
						Address: admin3,
						Weight:  1,
						Name:    "333",
					},
					{
						Address: admin4,
						Weight:  1,
						Name:    "444",
					},
				},
			}),
			Expected: vm.ExecutionResult{},
			Err:      ErrRepeatedAddress,
		},
		{
			Caller: admin1,
			Data: generateProposeData(t, CouncilExtraArgs{
				Candidates: []*CouncilMember{
					{
						Address: admin1,
						Weight:  1,
						Name:    "111",
					},
					{
						Address: admin2,
						Weight:  1,
						Name:    "222",
					},
					{
						Address: admin3,
						Weight:  1,
						Name:    "333",
					},
					{
						Address: admin4,
						Weight:  1,
						Name:    "444",
					},
				},
			}),
			Expected: vm.ExecutionResult{
				UsedGas: CouncilProposalGas,
				ReturnData: generateReturnData(t, &TestCouncilProposal{
					ID:          1,
					Type:        CouncilElect,
					Proposer:    admin1,
					TotalVotes:  4,
					PassVotes:   []string{},
					RejectVotes: []string{},
					Status:      Voting,
					Candidates: []*CouncilMember{
						{
							Address: admin1,
							Weight:  1,
							Name:    "111",
						},
						{
							Address: admin2,
							Weight:  1,
							Name:    "222",
						},
						{
							Address: admin3,
							Weight:  1,
							Name:    "333",
						},
						{
							Address: admin4,
							Weight:  1,
							Name:    "444",
						},
					},
				}),
			},
			Err: nil,
		},
		{
			Caller: "0xfff0000000000000000000000000000000000000",
			Data: generateProposeData(t, CouncilExtraArgs{
				Candidates: []*CouncilMember{
					{
						Address: admin1,
						Weight:  1,
						Name:    "111",
					},
					{
						Address: admin2,
						Weight:  1,
						Name:    "222",
					},
					{
						Address: admin3,
						Weight:  1,
						Name:    "333",
					},
					{
						Address: admin4,
						Weight:  1,
						Name:    "444",
					},
				},
			}),
			Expected: vm.ExecutionResult{
				UsedGas: CouncilProposalGas,
			},
			Err: ErrNotFoundCouncilMember,
		},
	}

	for _, test := range testcases {
		cm.Reset(stateLedger)

		result, err := cm.Run(&vm.Message{
			From: types.NewAddressByStr(test.Caller).ETHAddress(),
			Data: test.Data,
		})
		assert.Equal(t, test.Err, err)

		if result != nil {
			assert.Equal(t, nil, result.Err)
			assert.Equal(t, test.Expected.UsedGas, result.UsedGas)

			expectedCouncil := &Council{}
			err = json.Unmarshal(test.Expected.ReturnData, expectedCouncil)
			assert.Nil(t, err)

			actualCouncil := &Council{}
			err = json.Unmarshal(result.ReturnData, actualCouncil)
			assert.Nil(t, err)
			assert.Equal(t, *expectedCouncil, *actualCouncil)
		}
	}
}

func TestRunForVote(t *testing.T) {
	cm := NewCouncilManager(&common.SystemContractConfig{
		Logger: logrus.New(),
	})

	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	assert.Nil(t, err)
	ld, err := leveldb.New(filepath.Join(repoRoot, "node_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())

	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()
	stateLedger.EXPECT().AddLog(gomock.Any()).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()

	err = InitCouncilMembers(stateLedger, []*repo.Admin{
		{
			Address: admin1,
			Weight:  1,
			Name:    "111",
		},
		{
			Address: admin2,
			Weight:  1,
			Name:    "222",
		},
		{
			Address: admin3,
			Weight:  1,
			Name:    "333",
		},
		{
			Address: admin4,
			Weight:  1,
			Name:    "444",
		},
	}, "10000000")
	assert.Nil(t, err)

	cm.Reset(stateLedger)

	cm.propose(types.NewAddressByStr(admin1).ETHAddress(), &CouncilProposalArgs{
		BaseProposalArgs: BaseProposalArgs{
			ProposalType: uint8(CouncilElect),
			Title:        "council elect",
			Desc:         "desc",
			BlockNumber:  2,
		},
		CouncilExtraArgs: CouncilExtraArgs{
			Candidates: []*CouncilMember{
				{
					Address: admin1,
					Weight:  2,
					Name:    "111",
				},
				{
					Address: admin2,
					Weight:  2,
					Name:    "222",
				},
				{
					Address: admin3,
					Weight:  2,
					Name:    "333",
				},
				{
					Address: admin4,
					Weight:  2,
					Name:    "444",
				},
			},
		},
	})

	testcases := []struct {
		Caller   string
		Data     []byte
		Expected vm.ExecutionResult
		Err      error
	}{
		{
			Caller: admin2,
			Data:   generateVoteData(t, cm.proposalID.GetID()-1, Pass),
			Expected: vm.ExecutionResult{
				UsedGas: CouncilVoteGas,
				ReturnData: generateReturnData(t, &TestCouncilProposal{
					ID:          1,
					Type:        CouncilElect,
					Proposer:    admin1,
					TotalVotes:  3,
					PassVotes:   []string{admin1, admin2},
					RejectVotes: []string{},
					Status:      Voting,
					Candidates: []*CouncilMember{
						{
							Address: admin1,
							Weight:  2,
							Name:    "111",
						},
						{
							Address: admin2,
							Weight:  2,
							Name:    "222",
						},
						{
							Address: admin3,
							Weight:  2,
							Name:    "333",
						},
						{
							Address: admin4,
							Weight:  2,
							Name:    "444",
						},
					},
				}),
			},
			Err: nil,
		},
		{
			Caller: admin3,
			Data:   generateVoteData(t, cm.proposalID.GetID()-1, Pass),
			Expected: vm.ExecutionResult{
				UsedGas: CouncilVoteGas,
				ReturnData: generateReturnData(t, &TestCouncilProposal{
					ID:          1,
					Type:        CouncilElect,
					Proposer:    admin1,
					TotalVotes:  3,
					PassVotes:   []string{admin1, admin2, admin3},
					RejectVotes: []string{},
					Status:      Approved,
					Candidates: []*CouncilMember{
						{
							Address: admin1,
							Weight:  2,
							Name:    "111",
						},
						{
							Address: admin2,
							Weight:  2,
							Name:    "222",
						},
						{
							Address: admin3,
							Weight:  2,
							Name:    "333",
						},
						{
							Address: admin4,
							Weight:  2,
							Name:    "444",
						},
					},
				}),
			},
			Err: nil,
		},
		{
			Caller:   "0xfff0000000000000000000000000000000000000",
			Data:     generateVoteData(t, cm.proposalID.GetID()-1, Pass),
			Expected: vm.ExecutionResult{},
			Err:      ErrNotFoundCouncilMember,
		},
	}

	for _, test := range testcases {
		cm.Reset(stateLedger)

		result, err := cm.Run(&vm.Message{
			From: types.NewAddressByStr(test.Caller).ETHAddress(),
			Data: test.Data,
		})
		assert.Equal(t, test.Err, err)

		if result != nil {
			assert.Equal(t, nil, result.Err)
			assert.Equal(t, test.Expected.UsedGas, result.UsedGas)
			expectedCouncil := &Council{}
			err = json.Unmarshal(test.Expected.ReturnData, expectedCouncil)
			assert.Nil(t, err)

			actualCouncil := &Council{}
			err = json.Unmarshal(result.ReturnData, actualCouncil)
			assert.Nil(t, err)
			assert.Equal(t, *expectedCouncil, *actualCouncil)
		}
	}
}

func TestEstimateGas(t *testing.T) {
	cm := NewCouncilManager(&common.SystemContractConfig{
		Logger: logrus.New(),
	})

	from := types.NewAddressByStr(admin1).ETHAddress()
	to := types.NewAddressByStr(common.CouncilManagerContractAddr).ETHAddress()
	data := hexutil.Bytes(generateProposeData(t, CouncilExtraArgs{
		Candidates: []*CouncilMember{
			{
				Address: admin1,
				Weight:  1,
			},
		},
	}))
	// test propose
	gas, err := cm.EstimateGas(&types.CallArgs{
		From: &from,
		To:   &to,
		Data: &data,
	})
	assert.Nil(t, err)
	assert.Equal(t, CouncilProposalGas, gas)

	// test vote
	data = hexutil.Bytes(generateVoteData(t, 1, Pass))
	gas, err = cm.EstimateGas(&types.CallArgs{
		From: &from,
		To:   &to,
		Data: &data,
	})
	assert.Nil(t, err)
	assert.Equal(t, CouncilVoteGas, gas)
}

func generateProposeData(t *testing.T, extraArgs CouncilExtraArgs) []byte {
	gabi, err := GetABI()

	title := "title"
	desc := "desc"
	blockNumber := uint64(1000)
	extra, err := json.Marshal(extraArgs)
	assert.Nil(t, err)
	data, err := gabi.Pack(ProposeMethod, uint8(CouncilElect), title, desc, blockNumber, extra)
	assert.Nil(t, err)

	return data
}

func generateVoteData(t *testing.T, proposalID uint64, voteResult VoteResult) []byte {
	gabi, err := GetABI()

	data, err := gabi.Pack(VoteMethod, proposalID, voteResult, []byte(""))
	assert.Nil(t, err)

	return data
}

func generateReturnData(t *testing.T, testProposal *TestCouncilProposal) []byte {
	proposal := &CouncilProposal{
		BaseProposal: BaseProposal{
			ID:          testProposal.ID,
			Type:        testProposal.Type,
			Strategy:    NowProposalStrategy,
			Proposer:    testProposal.Proposer,
			Title:       "title",
			Desc:        "desc",
			BlockNumber: uint64(1000),
			TotalVotes:  testProposal.TotalVotes,
			PassVotes:   testProposal.PassVotes,
			RejectVotes: testProposal.RejectVotes,
			Status:      testProposal.Status,
		},
		Candidates: testProposal.Candidates,
	}

	b, err := json.Marshal(proposal)
	assert.Nil(t, err)

	return b
}
