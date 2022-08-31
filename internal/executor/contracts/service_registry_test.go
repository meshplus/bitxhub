package contracts

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/stretchr/testify/assert"
)

func TestSetPriceLevel(t *testing.T) {

	a := big.NewInt(1000000000000000000)
	b := big.NewInt(2678744073709551616)
	fmt.Println(a.Cmp(b))

	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))
	mockStub.EXPECT().Caller().Return("0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997")
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	sr := &ServiceRegistry{mockStub}

	res := sr.SetPriceLevel(1, 2, 3, 4, 5)
	assert.True(t, res.Ok)
}

func TestGetPriceLevel(t *testing.T) {

	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	priceLevel := PriceLevel{
		Price1Letter: 1,
		Price2Letter: 2,
		Price3Letter: 3,
		Price4Letter: 4,
		Price5Letter: 5,
	}
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, priceLevel).Return(true)
	sr := &ServiceRegistry{mockStub}

	var data PriceLevel
	res := sr.GetPriceLevel()
	json.Unmarshal(res.Result, &data)
	assert.Equal(t, data.Price1Letter, uint64(1))
}

func TestSetTokenPrice(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	price := uint64(100)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, price).Return(true)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))
	mockStub.EXPECT().Caller().Return("0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997")

	sr := &ServiceRegistry{mockStub}
	res := sr.SetTokenPrice(price)
	assert.True(t, res.Ok)

	res = sr.GetTokenPrice()
	priceRes := binary.BigEndian.Uint64(res.Result)
	assert.Equal(t, priceRes, uint64(price))

}

func TestGetDomainExpires(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	servDomain := make(map[string]uint64)
	servDomain["appchain.hub"] = 10000
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, servDomain).Return(true)

	sr := &ServiceRegistry{mockStub}
	res := sr.GetDomainExpires("appchain.hub")
	expire := binary.BigEndian.Uint64(res.Result)
	assert.Equal(t, expire, uint64(10000))
}

func TestRecordExists(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().Has("appchain.hub").Return(true)
	sr := &ServiceRegistry{mockStub}
	res := sr.RecordExists("appchain.hub")
	exist := res.Result
	assert.Equal(t, exist, []byte(strconv.FormatBool(true)))
}

func TestOwner(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	serviceDomainRec := &ServDomainRec{
		Owner:     "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997",
		Resolver:  "0x0000000000000000000000000000000000000023",
		SubDomain: make(map[string]bool),
	}

	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *serviceDomainRec).Return(true)
	mockStub.EXPECT().Has(gomock.Any()).Return(true)
	sr := &ServiceRegistry{mockStub}
	res := sr.Owner("appchain.hub")
	assert.True(t, res.Ok)

	assert.Equal(t, res.Result, []byte("0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"))
}

func TestResolver(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	serviceDomainRec := &ServDomainRec{
		Owner:     "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997",
		Resolver:  "0x0000000000000000000000000000000000000023",
		SubDomain: make(map[string]bool),
	}
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *serviceDomainRec).Return(true)
	mockStub.EXPECT().Has(gomock.Any()).Return(true)
	sr := &ServiceRegistry{mockStub}
	res := sr.Resolver("appchain.hub")
	assert.True(t, res.Ok)
	assert.Equal(t, res.Result, []byte("0x0000000000000000000000000000000000000023"))
}

//todo: an occasional slice bug need to fix
//func TestGetSubDomain(t *testing.T) {
//	mockCtl := gomock.NewController(t)
//	mockStub := mock_stub.NewMockStub(mockCtl)
//	servDomain := make(map[string]uint64)
//	servDomain["a.hub"] = 100000
//	servDomain["b.hub"] = 100000
//	mockStub.EXPECT().GetObject(Level1Domain, gomock.Any()).SetArg(1, servDomain).Return(true)
//
//	subDomain := make(map[string]bool)
//	subDomain["a.a.hub"] = true
//	subDomain["b.a.hub"] = true
//	serviceDomainRec := &ServDomainRec{
//		Owner:     "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997",
//		Resolver:  "0x0000000000000000000000000000000000000023",
//		SubDomain: subDomain,
//	}
//	mockStub.EXPECT().GetObject("a.hub", gomock.Any()).SetArg(1, *serviceDomainRec).Return(true)
//	sr := &ServiceRegistry{mockStub}
//	res := sr.GetSubDomain("hub")
//	assert.True(t, res.Ok)
//	var one []string
//	json.Unmarshal(res.Result, &one)
//	assert.Equal(t, one, []string{"a.hub", "b.hub"})
//
//	res = sr.GetSubDomain("a.hub")
//	var two []string
//	json.Unmarshal(res.Result, &two)
//	assert.Equal(t, two, []string{"a.a.hub", "b.a.hub"})
//
//}

func TestIsApproved(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	permissionController := make(map[string]map[string]bool)
	permissionController["0x0000000000000000000000000000000000000022"] = make(map[string]bool)
	permissionController["0x0000000000000000000000000000000000000022"]["0x0000000000000000000000000000000000000023"] = true
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, permissionController).Return(true)

	sr := &ServiceRegistry{mockStub}
	res := sr.IsApproved("0x0000000000000000000000000000000000000022", "0x0000000000000000000000000000000000000023")
	assert.True(t, res.Ok)

}
