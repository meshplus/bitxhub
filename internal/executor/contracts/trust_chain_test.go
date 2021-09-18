package contracts

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/stretchr/testify/require"
)

func TestTrustChain_AddTrustMeta(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	tc := &TrustChain{
		Stub: mockStub,
	}

	trustMeta := &TrustMeta{
		TrustContractAddr: "TrustContractAddr",
	}
	trustMetaData, err := json.Marshal(trustMeta)
	require.Nil(t, err)

	trustMeta1 := &TrustMeta{}
	trustMetaData1, err := json.Marshal(trustMeta1)
	require.Nil(t, err)

	mockStub.EXPECT().CrossInvoke(trustMeta.TrustContractAddr, gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()

	res := tc.AddTrustMeta(trustMetaData)
	require.True(t, res.Ok, string(res.Result))
	res = tc.AddTrustMeta(trustMetaData1)
	require.True(t, res.Ok, string(res.Result))
}

func TestTrustChain_GetTrustMeta(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	tc := &TrustChain{
		Stub: mockStub,
	}

	trustMeta := &TrustMeta{
		TrustContractAddr: "TrustContractAddr",
	}
	trustMetaData, err := json.Marshal(trustMeta)
	require.Nil(t, err)
	trustMeta1 := &TrustMeta{}
	trustMetaData1, err := json.Marshal(trustMeta1)
	require.Nil(t, err)

	mockStub.EXPECT().CrossInvoke(trustMeta.TrustContractAddr, gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).Times(1)
	mockStub.EXPECT().Get(gomock.Any()).Return(true, nil).AnyTimes()

	res := tc.GetTrustMeta(trustMetaData)
	require.True(t, res.Ok, string(res.Result))
	res = tc.GetTrustMeta(trustMetaData1)
	require.False(t, res.Ok, string(res.Result))
	res = tc.GetTrustMeta(trustMetaData1)
	require.True(t, res.Ok, string(res.Result))
}
