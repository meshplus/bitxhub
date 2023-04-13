package contracts

import (
	"encoding/json"
	"testing"

	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
)

func TestGetServDomainData(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	addr := make(map[uint64]string)
	addr[1] = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

	key0 := "appchain.hub"
	value0 := &ServDomainData{
		Addr:        addr,
		ServiceName: "name1",
		Des:         "des1",
	}
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *value0).Return(true)
	mockStub.EXPECT().CrossInvoke(constant.ServiceRegistryContractAddr.Address().String(), "Owner",
		pb.String("appchain.hub")).Return(boltvm.Success([]byte("0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"))).Times(1)
	sr := &ServiceResolver{mockStub}

	var data ServDomainData
	res := sr.GetServDomainData(key0)
	err := json.Unmarshal(res.Result, &data)
	assert.Nil(t, err)
	assert.True(t, res.Ok)
	assert.Equal(t, data.Addr[1], "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997")

}

func TestSetAddr(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	addr := make(map[uint64]string)
	addr[1] = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

	value0 := ServDomainData{
		Addr:        addr,
		ServiceName: "name1",
		Des:         "des1",
	}
	mockStub.EXPECT().GetObject("appchain.hub", gomock.Any()).SetArg(1, value0).Return(true)
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Caller().Return("0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997")
	mockStub.EXPECT().CurrentCaller().Return("0x0000000000000000000000000000000000000022")
	mockStub.EXPECT().CrossInvoke(constant.ServiceRegistryContractAddr.Address().String(), "Owner",
		pb.String("appchain.hub")).Return(boltvm.Success([]byte("0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.ServiceRegistryContractAddr.Address().String(), "IsApproved",
		pb.String(string(constant.ServiceRegistryContractAddr)), pb.String(string(constant.ServiceResolverContractAddr))).Return(
		boltvm.Success([]byte("true"))).Times(1)
	reverseName := make(map[string][]string)
	reverseName["1356:xxx:xxxx"] = []string{"appchain.hub"}

	mockStub.EXPECT().GetObject(ReverseMap, gomock.Any()).SetArg(1, reverseName).Return(true).AnyTimes()

	sr := &ServiceResolver{mockStub}

	res := sr.SetServiceName("appchain.hub", "1356:xxx:xxxx", 1)
	assert.True(t, res.Ok)
}

func TestGetReverseName(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	reverseName := make(map[string][]string)
	reverseName["1356:xxx:xxxx"] = []string{"appchain.hub"}
	mockStub.EXPECT().GetObject("reverseMap", gomock.Any()).SetArg(1, reverseName).Return(true).AnyTimes()
	sr := &ServiceResolver{mockStub}
	res := sr.GetReverseName("1356:xxx:xxxx")
	assert.True(t, res.Ok)

	res = sr.GetReverseName("1356:xxx:xx")
	assert.Equal(t, string(res.Result), "null")
}
