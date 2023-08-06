package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/ledger"
	vm "github.com/axiomesh/eth-kit/evm"
	ethledger "github.com/axiomesh/eth-kit/ledger"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	admin1 = "0x1210000000000000000000000000000000000000"
	admin2 = "0x1220000000000000000000000000000000000000"
	admin3 = "0x1230000000000000000000000000000000000000"
	admin4 = "0x1240000000000000000000000000000000000000"
)

func initializeCouncil(t *testing.T, lg ethledger.StateLedger, admins []*CouncilMember) {
	council := &Council{}
	council.Members = admins
	account := lg.GetOrCreateAccount(types.NewAddressByStr(common.CouncilManagerContractAddr))
	b, err := json.Marshal(council)
	assert.Nil(t, err)
	account.SetState([]byte(CouncilKey), b)
}

func TestRunForPropose(t *testing.T) {
	logger := logrus.New()
	cm := NewCouncilManager(logger)

	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot, err := os.MkdirTemp("", "leveldb_tmp")
	assert.Nil(t, err)
	ld, err := leveldb.New(filepath.Join(repoRoot, "node_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())

	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()

	initializeCouncil(t, stateLedger, []*CouncilMember{
		{
			Address: admin1,
			Weight:  1,
		},
		{
			Address: admin2,
			Weight:  1,
		},
		{
			Address: admin3,
			Weight:  1,
		},
		{
			Address: admin4,
			Weight:  1,
		},
	})

	cm.Reset(stateLedger)

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
					},
					{
						Address: admin2,
						Weight:  1,
					},
					{
						Address: admin3,
						Weight:  1,
					},
				},
			}),
			Expected: vm.ExecutionResult{
				UsedGas: CouncilProposalGas,
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
					},
					{
						Address: admin2,
						Weight:  1,
					},
					{
						Address: admin3,
						Weight:  1,
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
		result, err := cm.Run(&vm.Message{
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

func TestRunForVote(t *testing.T) {
	logger := logrus.New()
	cm := NewCouncilManager(logger)

	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot, err := os.MkdirTemp("", "leveldb_tmp")
	assert.Nil(t, err)
	ld, err := leveldb.New(filepath.Join(repoRoot, "node_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())

	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()

	initializeCouncil(t, stateLedger, []*CouncilMember{
		{
			Address: admin1,
			Weight:  1,
		},
		{
			Address: admin2,
			Weight:  1,
		},
		{
			Address: admin3,
			Weight:  1,
		},
	})

	cm.Reset(stateLedger)
	cm.propose(types.NewAddressByStr(admin1).ETHAddress(), &CouncilProposalArgs{
		BaseProposalArgs: BaseProposalArgs{
			ProposalType: uint8(CouncilElect),
			Title:        "council elect",
			Desc:         "desc",
			BlockNumber:  1,
		},
		CouncilExtraArgs: CouncilExtraArgs{
			Candidates: []*CouncilMember{
				{
					Address: admin1,
					Weight:  1,
				},
				{
					Address: admin2,
					Weight:  1,
				},
				{
					Address: admin3,
					Weight:  1,
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
			Caller: admin1,
			Data:   generateVoteData(t, globalProposalID.GetID()-1, Pass),
			Expected: vm.ExecutionResult{
				UsedGas: CouncilVoteGas,
			},
			Err: nil,
		},
		{
			Caller: admin2,
			Data:   generateVoteData(t, globalProposalID.GetID()-1, Pass),
			Expected: vm.ExecutionResult{
				UsedGas: CouncilVoteGas,
			},
			Err: nil,
		},
	}

	for _, test := range testcases {
		result, err := cm.Run(&vm.Message{
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
