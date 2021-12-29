package contracts

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

func TestGovStrategy_UpdateProposalStrategy(t *testing.T) {
	g, mockStub, strategies := proposalStrategyPrepare(t)
	logger := log.NewWithModule("contracts")

	data := make([][]byte, 0)
	for _, strategy := range strategies {
		d, err := json.Marshal(strategy)
		assert.Nil(t, err)
		data = append(data, d)
	}

	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, data).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8").Times(1)
	mockStub.EXPECT().Caller().Return("0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8").Times(1)
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *strategies[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(appchainAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("SubmitProposal"),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Eq(constant.GovernanceContractAddr.Address().String()), gomock.Eq("ZeroPermission"),
		gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().PostEvent(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()

	res := g.GetAllProposalStrategy()
	assert.True(t, res.Ok)
	ps1 := make([]*ProposalStrategy, 0)
	err := json.Unmarshal(res.Result, &ps1)
	assert.Nil(t, err)
	assert.True(t, len(ps1) == len(strategies))

	res = g.UpdateProposalStrategy(strategies[0].Module, string(SimpleMajority), 0.5, "123")
	assert.True(t, res.Ok, string(res.Result))
}

func proposalStrategyPrepare(t *testing.T) (*GovStrategy, *mock_stub.MockStub, []*ProposalStrategy) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	g := &GovStrategy{
		Stub: mockStub,
	}
	strategies := make([]*ProposalStrategy, 0)
	for i := 0; i < 5; i++ {
		ps := &ProposalStrategy{
			Module:               repo.AppchainMgr,
			Typ:                  ZeroPermission,
			Status:               governance.GovernanceAvailable,
			ParticipateThreshold: 0,
		}
		strategies = append(strategies, ps)
	}
	return g, mockStub, strategies
}
