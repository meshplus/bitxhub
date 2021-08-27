package contracts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var caller = "0x3f9d18f7c3a6e5e4c0b877fe3e688ab08840b997"

func TestAppchainManager_Appchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addr0 := types.NewAddress([]byte{0}).String()
	addr1 := types.NewAddress([]byte{1}).String()

	chain := &appchainMgr.Appchain{
		ID:            addr0,
		Name:          "appchain A",
		Validators:    "",
		ConsensusType: "",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "11111",
	}

	data, err := json.Marshal(chain)
	assert.Nil(t, err)

	o1 := mockStub.EXPECT().Caller().Return(addr0)
	o2 := mockStub.EXPECT().Caller().Return(addr1)
	gomock.InOrder(o1, o2)
	mockStub.EXPECT().Get("appchain-"+addr0).Return(true, data)
	mockStub.EXPECT().Get("appchain-"+addr1).Return(false, nil)

	am := &AppchainManager{
		Stub: mockStub,
	}

	res := am.Appchain()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, data, res.Result)

	res = am.Appchain()
	assert.Equal(t, false, res.Ok)
}

func TestAppchainManager_Appchains(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	var chains []*appchainMgr.Appchain
	var chainsData [][]byte
	for i := 0; i < 2; i++ {
		addr := types.NewAddress([]byte{byte(i)}).String()

		chain := &appchainMgr.Appchain{
			Status:        appchainMgr.AppchainAvailable,
			ID:            addr,
			Name:          "appchain" + addr,
			Validators:    "",
			ConsensusType: "",
			ChainType:     "fabric",
			Desc:          "",
			Version:       "",
			PublicKey:     "pubkey" + addr,
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
	}

	logger := log.NewWithModule("contracts")

	am := &AppchainManager{
		Stub: mockStub,
	}
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()

	// test for register
	mockStub.EXPECT().Get(gomock.Any()).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))
	mockStub.EXPECT().Has(AppchainKey(caller)).Return(false).MaxTimes(3)
	am.Register(chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)

	appchainsReq1 := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(true, chainsData)
	appchainReq2 := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(false, nil)
	counterAppchainReq := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(true, chainsData)
	gomock.InOrder(appchainsReq1, appchainReq2, counterAppchainReq)
	res := am.Appchains()
	assert.Equal(t, true, res.Ok)

	var appchains []*appchainMgr.Appchain
	err := json.Unmarshal(res.Result, &appchains)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(chains))
	assert.Equal(t, chains[0], appchains[0])
	assert.Equal(t, chains[1], appchains[1])

	res = am.Appchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, []byte(nil), res.Result)

	// counter chains
	res = am.CountAppchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "2", string(res.Result))

	// test GetAppchain
	res = am.GetAppchain(caller)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, chainsData[0], res.Result)
}

func TestAppchainManager_Register(t *testing.T) {
	am, mockStub, chains, chainsData := prepare(t)

	logger := log.NewWithModule("contracts")

	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			chain := ret.(*appchainMgr.Appchain)
			chain.ID = chains[0].ID
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))
	mockStub.EXPECT().Has(AppchainKey(caller)).Return(false).Times(1)
	mockStub.EXPECT().Has(AppchainKey(caller)).Return(true).AnyTimes()
	res := am.Register(chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)
	assert.True(t, res.Ok)

	// test for repeated register
	am.Register(chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)
	assert.True(t, res.Ok)
}

func TestAppchainManager_Manager(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	am := &AppchainManager{
		Stub: mockStub,
	}

	chain := &appchainMgr.Appchain{
		Status:        appchainMgr.AppchainUpdating,
		ID:            "addr",
		Name:          "appchain A",
		Validators:    "",
		ConsensusType: "",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "11111",
	}
	data, err := json.Marshal(chain)
	assert.Nil(t, err)

	chain1 := &appchainMgr.Appchain{
		Status:        appchainMgr.AppchainUpdating,
		ID:            "addr1",
		Name:          "appchain A",
		Validators:    "",
		ConsensusType: "",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "11111",
	}
	data1, err := json.Marshal(chain1)
	assert.Nil(t, err)

	mockStub.EXPECT().Get(AppchainKey("addr")).Return(true, data).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey("addr1")).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Has(AppchainKey("addr")).Return(true).AnyTimes()
	mockStub.EXPECT().Has(AppchainKey("addr1")).Return(false).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("addrNotAdmin").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test without permission
	res := am.Manager(appchainMgr.EventUpdate, string(APPOVED), data)
	assert.False(t, res.Ok)
	// test with permission
	res = am.Manager(appchainMgr.EventUpdate, string(APPOVED), data1)
	assert.False(t, res.Ok)
	res = am.Manager(appchainMgr.EventUpdate, string(REJECTED), data1)
	assert.False(t, res.Ok)
	res = am.Manager(appchainMgr.EventUpdate, string(APPOVED), data)
	assert.True(t, res.Ok, string(res.Result))
	res = am.Manager(appchainMgr.EventUpdate, string(REJECTED), data)
	assert.True(t, res.Ok, string(res.Result))

	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "Register", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "Register", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	res = am.Manager(appchainMgr.EventRegister, string(APPOVED), data)
	assert.False(t, res.Ok)
	res = am.Manager(appchainMgr.EventRegister, string(APPOVED), data)
	assert.True(t, res.Ok, string(res.Result))
}

func TestAppchainManager_IsAvailable(t *testing.T) {
	am, mockStub, chains, chainsData := prepare(t)
	mockStub.EXPECT().Get(AppchainKey(chains[0].ID)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(chains[1].ID)).Return(true, chainsData[1]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey("errId")).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey("unmarshalErrId")).Return(true, []byte("1")).AnyTimes()

	res := am.IsAvailable(chains[0].ID)
	assert.Equal(t, true, res.Ok, string(res.Result))
	res = am.IsAvailable(chains[1].ID)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = am.IsAvailable("errId")
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = am.IsAvailable("unmarshalErrId")
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestManageChain(t *testing.T) {
	am, mockStub, chains, chainsData := prepare(t)
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(caller).AnyTimes()
	mockStub.EXPECT().Has(gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(caller)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey("freezingChain")).Return(true, chainsData[1]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test FreezeAppchain
	res = am.FreezeAppchain(caller)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain("freezingChain")
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test LogoutAppchain
	res = am.LogoutAppchain()
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestManageChain_WithoutPermission(t *testing.T) {
	am, mockStub, _, _ := prepare(t)
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(caller).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).AnyTimes()

	// test FreezeAppchain
	res := am.FreezeAppchain("addr")
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = am.FreezeAppchain("addr")
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain("addr")
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestManageChain_Error(t *testing.T) {
	am, mockStub, chains, _ := prepare(t)
	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(caller).AnyTimes()
	mockStub.EXPECT().Has(gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, nil).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test FreezeAppchain
	res = am.FreezeAppchain(caller)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain(caller)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test LogoutAppchain
	res = am.LogoutAppchain()
	assert.Equal(t, false, res.Ok, string(res.Result))
}

func TestCountApprovedAppchains(t *testing.T) {
	am, mockStub, _, chainsData := prepare(t)

	logger := log.NewWithModule("contracts")
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	// test for CountApprovedAppchains
	mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(true, chainsData)
	res := am.CountAvailableAppchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "1", string(res.Result))
}

func TestDeleteAppchain(t *testing.T) {
	am, mockStub, _, _ := prepare(t)

	approveRes := &boltvm.Response{
		Ok:     true,
		Result: []byte("true"),
	}
	logger := log.NewWithModule("contracts")
	// test for DeleteAppchain
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", gomock.Any()).Return(boltvm.Success(nil)).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", gomock.Any()).Return(boltvm.Success([]byte(strconv.FormatBool(false)))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", gomock.Any()).Return(approveRes).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "DeleteInterchain",
		gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "DeleteInterchain",
		gomock.Any()).Return(approveRes).AnyTimes()
	mockStub.EXPECT().Delete(AppchainKey(caller)).Return()

	// judge caller type error
	res := am.DeleteAppchain(caller)
	assert.Equal(t, false, res.Ok)
	// caller is not an admin account
	res = am.DeleteAppchain(caller)
	assert.Equal(t, false, res.Ok)
	// CrossInvoke DeleteInterchain error
	res = am.DeleteAppchain(caller)
	assert.Equal(t, false, res.Ok)

	res = am.DeleteAppchain(caller)
	assert.Equal(t, true, res.Ok)
}

func TestGetPubKeyByChainID(t *testing.T) {
	am, mockStub, chains, _ := prepare(t)
	// test for GetPubKeyByChainID
	mockStub.EXPECT().Has(AppchainKey(caller)).Return(true)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			chain := ret.(*appchainMgr.Appchain)
			chain.Status = appchainMgr.AppchainAvailable
			chain.PublicKey = chains[0].PublicKey
			assert.Equal(t, key, AppchainKey(caller))
			fmt.Printf("chain is %v", chain)
			return true
		}).AnyTimes()
	res := am.GetPubKeyByChainID(caller)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, chains[0].PublicKey, string(res.Result))
}

func prepare(t *testing.T) (*AppchainManager, *mock_stub.MockStub, []*appchainMgr.Appchain, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	am := &AppchainManager{
		Stub: mockStub,
	}

	var chains []*appchainMgr.Appchain
	var chainsData [][]byte
	chainType := []string{string(appchainMgr.AppchainAvailable), string(appchainMgr.AppchainFrozen)}
	for i := 0; i < 2; i++ {
		addr := types.NewAddress([]byte{byte(i)}).String()

		chain := &appchainMgr.Appchain{
			Status:        appchainMgr.AppchainStatus(chainType[i]),
			ID:            addr,
			Name:          "appchain" + addr,
			Validators:    "",
			ConsensusType: "",
			ChainType:     "fabric",
			Desc:          "",
			Version:       "",
			PublicKey:     "pubkey" + addr,
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
	}

	return am, mockStub, chains, chainsData
}

func TestInterchainManager_Register(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addr := types.NewAddress([]byte{0}).String()
	//mockStub.EXPECT().Caller().Return(addr).AnyTimes()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	o1 := mockStub.EXPECT().Get(appchainMgr.PREFIX+addr).Return(false, nil)

	interchain := pb.Interchain{
		ID:                   addr,
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[addr] = 1
	interchain.ReceiptCounter[addr] = 1
	interchain.SourceReceiptCounter[addr] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)
	o2 := mockStub.EXPECT().Get(appchainMgr.PREFIX+addr).Return(true, data0)

	interchain = pb.Interchain{
		ID:                   addr,
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	data1, err := interchain.Marshal()
	assert.Nil(t, err)
	o3 := mockStub.EXPECT().Get(appchainMgr.PREFIX+addr).Return(true, data1)
	gomock.InOrder(o1, o2, o3)

	im := &InterchainManager{mockStub}

	res := im.Register(addr)
	assert.Equal(t, true, res.Ok)

	ic := &pb.Interchain{}
	err = ic.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, addr, ic.ID)
	assert.Equal(t, 0, len(ic.InterchainCounter))
	assert.Equal(t, 0, len(ic.ReceiptCounter))
	assert.Equal(t, 0, len(ic.SourceReceiptCounter))

	res = im.Register(addr)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, data0, res.Result)

	res = im.Register(addr)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, data1, res.Result)
}

func TestInterchainManager_Interchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addr := types.NewAddress([]byte{0}).String()
	mockStub.EXPECT().Caller().Return(addr).AnyTimes()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	o1 := mockStub.EXPECT().Get(appchainMgr.PREFIX+addr).Return(false, nil)

	interchain := pb.Interchain{
		ID:                   addr,
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[addr] = 1
	interchain.ReceiptCounter[addr] = 1
	interchain.SourceReceiptCounter[addr] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)
	o2 := mockStub.EXPECT().Get(appchainMgr.PREFIX+addr).Return(true, data0)
	gomock.InOrder(o1, o2)

	im := &InterchainManager{mockStub}

	res := im.Interchain()
	assert.False(t, res.Ok)

	res = im.Interchain()
	assert.True(t, res.Ok)
	assert.Equal(t, data0, res.Result)
}

func TestInterchainManager_GetInterchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addr := types.NewAddress([]byte{0}).String()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	o1 := mockStub.EXPECT().Get(appchainMgr.PREFIX+addr).Return(false, nil)

	interchain := pb.Interchain{
		ID:                   addr,
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[addr] = 1
	interchain.ReceiptCounter[addr] = 1
	interchain.SourceReceiptCounter[addr] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)
	o2 := mockStub.EXPECT().Get(appchainMgr.PREFIX+addr).Return(true, data0)
	gomock.InOrder(o1, o2)

	im := &InterchainManager{mockStub}

	res := im.GetInterchain(addr)
	assert.False(t, res.Ok)

	res = im.GetInterchain(addr)
	assert.True(t, res.Ok)
	assert.Equal(t, data0, res.Result)
}

func TestInterchainManager_HandleIBTP(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	from := types.NewAddress([]byte{0}).String()
	to := types.NewAddress([]byte{1}).String()

	unexistChain := types.NewAddress([]byte{2}).String()
	unavailableChain := types.NewAddress([]byte{3}).String()

	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxTimestamp().Return(time.Now().UnixNano()).AnyTimes()

	interchain := pb.Interchain{
		ID:                   from,
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[to] = 1
	interchain.ReceiptCounter[to] = 1
	interchain.SourceReceiptCounter[to] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)

	mockStub.EXPECT().Get(appchainMgr.PREFIX+from).Return(true, data0).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.PREFIX+to).Return(true, data0).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.PREFIX+unexistChain).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.PREFIX+unavailableChain).Return(false, nil).AnyTimes()

	appchain := &appchainMgr.Appchain{
		ID:            "",
		Name:          "Relay1",
		Validators:    "",
		ConsensusType: "",
		Status:        appchainMgr.AppchainAvailable,
		ChainType:     "appchain",
		Desc:          "Relay1",
		Version:       "1",
		PublicKey:     "",
	}
	appchainData, err := json.Marshal(appchain)
	assert.Nil(t, err)

	unavailableAppchain := &appchainMgr.Appchain{
		ID:            unavailableChain,
		Name:          "Relay1",
		Validators:    "",
		ConsensusType: "",
		Status:        appchainMgr.AppchainFrozen,
		ChainType:     "appchain",
		Desc:          "Relay1",
		Version:       "1",
		PublicKey:     "",
	}
	unavailableAppchainData, err := json.Marshal(unavailableAppchain)
	assert.Nil(t, err)

	// mockStub.EXPECT().IsRelayIBTP(gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Eq("GetAppchain"), pb.String(from)).Return(boltvm.Success(appchainData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Eq("GetAppchain"), pb.String(to)).Return(boltvm.Success(appchainData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Eq("GetAppchain"), pb.String(unavailableChain)).Return(boltvm.Success(unavailableAppchainData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Eq("GetAppchain"), pb.String(unexistChain)).Return(boltvm.Error("")).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Not("GetAppchain"), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxIndex().Return(uint64(1)).AnyTimes()
	mockStub.EXPECT().PostInterchainEvent(gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxHash().Return(&types.Hash{}).AnyTimes()
	metas := []*InterchainMeta{}
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-send-interchain-%s", from), gomock.Any()).SetArg(1, metas).Return(true)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-receipt-interchain-%s", unexistChain), gomock.Any()).SetArg(1, metas).Return(false)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-receipt-interchain-%s", unavailableChain), gomock.Any()).SetArg(1, metas).Return(false)

	im := &InterchainManager{mockStub}

	ibtp := &pb.IBTP{}

	res := im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))

	ibtp.From = from
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))

	ibtp.From = unexistChain
	ibtp.To = to
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), CurAppchainNotAvailable))

	ibtp.From = unavailableChain
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), CurAppchainNotAvailable))

	mockStub.EXPECT().Caller().Return(to).MaxTimes(1)
	ibtp.From = from
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))

	mockStub.EXPECT().Caller().Return(from).MaxTimes(5)
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexExist))

	ibtp.Index = 3
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexWrong))

	ibtp.Index = 1
	ibtp.To = unexistChain
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), TargetAppchainNotAvailable))

	mockStub.EXPECT().GetObject(fmt.Sprintf("index-send-interchain-%s", from), gomock.Any()).SetArg(1, metas).Return(true)
	ibtp.To = unavailableChain
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), TargetAppchainNotAvailable))

	ibtp.Type = pb.IBTP_RECEIPT_SUCCESS
	ibtp.From = unexistChain
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))

	ibtp.From = from
	ibtp.To = unexistChain
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), CurAppchainNotAvailable))

	ibtp.To = to
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), InvalidIBTP))

	mockStub.EXPECT().Caller().Return(to).AnyTimes()
	ibtp.Index = 1
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexExist))

	ibtp.Index = 3
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, true, strings.Contains(string(res.Result), ibtpIndexWrong))

	ibtp.Index = 2
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_RECEIPT_FAILURE
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)
}

func TestInterchainManager_GetIBTPByID(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	from := types.NewAddress([]byte{0}).String()
	to := types.NewAddress([]byte{1}).String()

	validID := fmt.Sprintf("%s-%s-1", from, to)

	mockStub.EXPECT().Caller().Return(from).AnyTimes()
	im := &InterchainManager{mockStub}

	res := im.GetIBTPByID("a")
	assert.False(t, res.Ok)
	assert.Equal(t, "wrong ibtp id", string(res.Result))

	res = im.GetIBTPByID("abc-def-10")
	assert.False(t, res.Ok)
	assert.Equal(t, "The caller does not have access to this ibtp", string(res.Result))

	unexistId := fmt.Sprintf("%s-%s-10", from, to)
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-tx-%s", unexistId), gomock.Any()).Return(false)

	res = im.GetIBTPByID(unexistId)
	assert.False(t, res.Ok)
	assert.Equal(t, "this id is not existed", string(res.Result))

	mockStub.EXPECT().GetObject(fmt.Sprintf("index-tx-%s", validID), gomock.Any()).Return(true)
	res = im.GetIBTPByID(validID)
	assert.True(t, res.Ok)
}

func TestInterchainManager_HandleUnionIBTP(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	from := types.NewAddress([]byte{0}).String()
	to := types.NewAddress([]byte{1}).String()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().Has(gomock.Any()).Return(true).AnyTimes()

	interchain := pb.Interchain{
		ID:                   from,
		InterchainCounter:    make(map[string]uint64),
		ReceiptCounter:       make(map[string]uint64),
		SourceReceiptCounter: make(map[string]uint64),
	}
	interchain.InterchainCounter[to] = 1
	interchain.ReceiptCounter[to] = 1
	interchain.SourceReceiptCounter[to] = 1
	data0, err := interchain.Marshal()
	assert.Nil(t, err)

	relayChain := &appchainMgr.Appchain{
		Status:        appchainMgr.AppchainAvailable,
		ID:            from,
		Name:          "appchain" + from,
		Validators:    "",
		ConsensusType: "",
		ChainType:     "fabric",
		Desc:          "",
		Version:       "",
		PublicKey:     "pubkey",
	}

	keys := make([]crypto.PrivateKey, 0, 4)
	var bv BxhValidators
	addrs := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		keyPair, err := asym.GenerateKeyPair(crypto.Secp256k1)
		require.Nil(t, err)
		keys = append(keys, keyPair)
		address, err := keyPair.PublicKey().Address()
		require.Nil(t, err)
		addrs = append(addrs, address.String())
	}

	bv.Addresses = addrs
	addrsData, err := json.Marshal(bv)
	require.Nil(t, err)
	relayChain.Validators = string(addrsData)

	data, err := json.Marshal(relayChain)
	assert.Nil(t, err)

	mockStub.EXPECT().Get(appchainMgr.PREFIX+from).Return(true, data0).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.PREFIX+from+"-"+from).Return(true, data0).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(data)).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxIndex().Return(uint64(1)).AnyTimes()
	mockStub.EXPECT().PostInterchainEvent(gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxHash().Return(&types.Hash{}).AnyTimes()
	mockStub.EXPECT().GetTxTimestamp().Return(time.Now().UnixNano()).AnyTimes()

	im := &InterchainManager{mockStub}

	ibtp := &pb.IBTP{
		From:      from + "-" + from,
		To:        to,
		Index:     0,
		Type:      pb.IBTP_INTERCHAIN,
		Timestamp: 0,
		Proof:     nil,
		Payload:   nil,
		Version:   "",
		Extra:     nil,
	}

	mockStub.EXPECT().Caller().Return(ibtp.From).AnyTimes()
	metas := []*InterchainMeta{}
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-send-interchain-%s", ibtp.From), gomock.Any()).SetArg(1, metas).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(fmt.Sprintf("index-receipt-interchain-%s", ibtp.To), gomock.Any()).SetArg(1, metas).Return(true).AnyTimes()

	res := im.handleUnionIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, "wrong index, required 2, but 0", string(res.Result))

	ibtp.Index = 2

	ibtpHash := ibtp.Hash()
	hash := sha256.Sum256([]byte(ibtpHash.String()))
	sign := &pb.SignResponse{Sign: make(map[string][]byte)}
	for _, key := range keys {
		signData, err := key.Sign(hash[:])
		require.Nil(t, err)

		address, err := key.PublicKey().Address()
		require.Nil(t, err)
		ok, err := asym.Verify(crypto.Secp256k1, signData[:], hash[:], *address)
		require.Nil(t, err)
		require.True(t, ok)
		sign.Sign[address.String()] = signData
	}
	signData, err := sign.Marshal()
	require.Nil(t, err)
	ibtp.Proof = signData

	res = im.handleUnionIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_ASSET_EXCHANGE_INIT
	ibtpHash = ibtp.Hash()
	hash = sha256.Sum256([]byte(ibtpHash.String()))
	sign = &pb.SignResponse{Sign: make(map[string][]byte)}
	for _, key := range keys {
		signData, err := key.Sign(hash[:])
		require.Nil(t, err)

		address, err := key.PublicKey().Address()
		require.Nil(t, err)
		ok, err := asym.Verify(crypto.Secp256k1, signData[:], hash[:], *address)
		require.Nil(t, err)
		require.True(t, ok)
		sign.Sign[address.String()] = signData
	}
	signData, err = sign.Marshal()
	require.Nil(t, err)
	ibtp.Proof = signData
	res = im.handleUnionIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_ASSET_EXCHANGE_REFUND

	ibtpHash = ibtp.Hash()
	hash = sha256.Sum256([]byte(ibtpHash.String()))
	sign = &pb.SignResponse{Sign: make(map[string][]byte)}
	for _, key := range keys {
		signData, err := key.Sign(hash[:])
		require.Nil(t, err)

		address, err := key.PublicKey().Address()
		require.Nil(t, err)
		ok, err := asym.Verify(crypto.Secp256k1, signData[:], hash[:], *address)
		require.Nil(t, err)
		require.True(t, ok)
		sign.Sign[address.String()] = signData
	}
	signData, err = sign.Marshal()
	require.Nil(t, err)
	ibtp.Proof = signData

	res = im.handleUnionIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_ASSET_EXCHANGE_REDEEM

	ibtpHash = ibtp.Hash()
	hash = sha256.Sum256([]byte(ibtpHash.String()))
	sign = &pb.SignResponse{Sign: make(map[string][]byte)}
	for _, key := range keys {
		signData, err := key.Sign(hash[:])
		require.Nil(t, err)

		address, err := key.PublicKey().Address()
		require.Nil(t, err)
		ok, err := asym.Verify(crypto.Secp256k1, signData[:], hash[:], *address)
		require.Nil(t, err)
		require.True(t, ok)
		sign.Sign[address.String()] = signData
	}
	signData, err = sign.Marshal()
	require.Nil(t, err)
	ibtp.Proof = signData

	res = im.handleUnionIBTP(ibtp)
	assert.True(t, res.Ok)
}

func TestRole_GetRole(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()
	mockStub.EXPECT().Caller().Return(admins[0].Address)

	im := &Role{mockStub}

	res := im.GetRole()
	assert.True(t, res.Ok)
	assert.Equal(t, "admin", string(res.Result))

	mockStub.EXPECT().Caller().Return(types.NewAddress([]byte{2}).String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error(""))

	res = im.GetRole()
	assert.True(t, res.Ok)
	assert.Equal(t, "none", string(res.Result))

	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))

	res = im.GetRole()
	assert.True(t, res.Ok)
	assert.Equal(t, "appchain_admin", string(res.Result))
}

func TestRole_IsAdmin(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.IsAdmin(admins[0].Address)
	assert.True(t, res.Ok)
	assert.Equal(t, "true", string(res.Result))

	res = im.IsAdmin(types.NewAddress([]byte{2}).String())
	assert.True(t, res.Ok)
	assert.Equal(t, "false", string(res.Result))
}

func TestRole_GetAdminRoles(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
		&repo.Admin{
			Address: "0x79a1215469FaB6f9c63c1816b45183AD3624bE34",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.GetAdminRoles()
	assert.True(t, res.Ok)

	var as []*repo.Admin
	err := json.Unmarshal(res.Result, &as)
	assert.Nil(t, err)
	assert.Equal(t, len(admins), len(as))
	for i, admin := range admins {
		assert.Equal(t, admin.Address, as[i].Address)
	}
}

func TestRole_SetAdminRoles(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addrs := []string{types.NewAddress([]byte{0}).String(), types.NewAddress([]byte{1}).String()}
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()

	im := &Role{mockStub}

	data, err := json.Marshal(addrs)
	assert.Nil(t, err)

	res := im.SetAdminRoles(string(data))
	assert.True(t, res.Ok)
}

func TestRole_GetRoleWeight(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.GetRoleWeight(admins[0].Address)
	assert.True(t, res.Ok)
	w, err := strconv.Atoi(string(res.Result))
	assert.Nil(t, err)
	assert.Equal(t, admins[0].Weight, uint64(w))

	res = im.GetRoleWeight("")
	assert.False(t, res.Ok)
}

func TestRole_CheckPermission(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
			Weight:  1,
		},
	}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, admins).AnyTimes()

	im := &Role{mockStub}

	res := im.CheckPermission(string(PermissionAdmin), "", admins[0].Address, nil)
	assert.True(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionAdmin), "", types.NewAddress([]byte{2}).String(), nil)
	assert.False(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionSelfAdmin), "", admins[0].Address, nil)
	assert.True(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionSelfAdmin), "", types.NewAddress([]byte{2}).String(), nil)
	assert.False(t, res.Ok, string(res.Result))

	addrData, err := json.Marshal([]string{admins[0].Address})
	assert.Nil(t, err)
	res = im.CheckPermission(string(PermissionSpecific), "", admins[0].Address, addrData)
	assert.True(t, res.Ok, string(res.Result))
	res = im.CheckPermission(string(PermissionSpecific), "", "", addrData)
	assert.False(t, res.Ok, string(res.Result))
}

func TestRuleManager_RegisterRule(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := types.NewAddress([]byte{0}).String()
	id1 := types.NewAddress([]byte{1}).String()

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(id0)).Return(boltvm.Success(nil))
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "IsAvailable", pb.String(id1)).Return(boltvm.Error(""))
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()

	im := &RuleManager{mockStub}

	addr := types.NewAddress([]byte{2}).String()
	res := im.RegisterRule(id0, addr)
	assert.True(t, res.Ok)

	res = im.RegisterRule(id1, addr)
	assert.False(t, res.Ok)
}

func TestRuleManager_GetRuleAddress(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := types.NewAddress([]byte{0}).String()
	id1 := types.NewAddress([]byte{1}).String()
	rule := Rule{
		Address: "123",
		Status:  1,
	}

	mockStub.EXPECT().GetObject(RuleKey(id0), gomock.Any()).SetArg(1, rule).Return(true)
	mockStub.EXPECT().GetObject(RuleKey(id1), gomock.Any()).Return(false).MaxTimes(5)

	im := &RuleManager{mockStub}

	res := im.GetRuleAddress(id0, "")
	assert.True(t, res.Ok)
	assert.Equal(t, rule.Address, string(res.Result))

	res = im.GetRuleAddress(id1, "fabric")
	assert.True(t, res.Ok)
	assert.Equal(t, validator.FabricRuleAddr, string(res.Result))

	res = im.GetRuleAddress(id1, "hyperchain")
	assert.False(t, res.Ok)
	assert.Equal(t, "", string(res.Result))
}

func TestRuleManager_Audit(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := types.NewAddress([]byte{0}).String()
	id1 := types.NewAddress([]byte{1}).String()
	rule := Rule{
		Address: "123",
		Status:  1,
	}

	mockStub.EXPECT().GetObject(RuleKey(id0), gomock.Any()).SetArg(1, rule).Return(true)
	mockStub.EXPECT().GetObject(RuleKey(id1), gomock.Any()).Return(false)
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject("audit-record-"+id0, gomock.Any()).SetArg(1, []*ruleRecord{}).Return(true).AnyTimes()

	im := &RuleManager{mockStub}

	res := im.Audit(id1, 0, "")
	assert.False(t, res.Ok)
	assert.Equal(t, "this rule does not exist", string(res.Result))

	res = im.Audit(id0, 1, "approve")
	assert.True(t, res.Ok)
}

func TestStore_Get(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	key0 := "1"
	val0 := "10"
	key1 := "1"

	mockStub.EXPECT().GetObject(key0, gomock.Any()).SetArg(1, val0).Return(true)
	mockStub.EXPECT().GetObject(key1, gomock.Any()).Return(false)

	im := &Store{mockStub}

	res := im.Get(key0)
	assert.True(t, res.Ok)
	assert.Equal(t, val0, string(res.Result))

	res = im.Get(key1)
	assert.False(t, res.Ok)
	assert.Equal(t, "there is not exist key", string(res.Result))
}

func TestStore_Set(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any())

	im := &Store{mockStub}

	res := im.Set("1", "2")
	assert.True(t, res.Ok)
}

func TestTransactionManager_Begin(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id := types.NewHash([]byte{0}).String()
	mockStub.EXPECT().AddObject(fmt.Sprintf("%s-%s", PREFIX, id), pb.TransactionStatus_BEGIN)

	im := &TransactionManager{mockStub}

	res := im.Begin(id)
	assert.True(t, res.Ok)
}

func TestTransactionManager_Report(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := "id0"
	id1 := "id1"
	txInfoKey := fmt.Sprintf("%s-%s", PREFIX, id0)

	im := &TransactionManager{mockStub}

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, pb.TransactionStatus_SUCCESS).Return(true)
	res := im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction with Id %s is finished", id0), string(res.Result))

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, pb.TransactionStatus_BEGIN).Return(true)
	mockStub.EXPECT().SetObject(txInfoKey, pb.TransactionStatus_SUCCESS)
	res = im.Report(id0, 0)
	assert.True(t, res.Ok)

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, pb.TransactionStatus_BEGIN).Return(true)
	mockStub.EXPECT().SetObject(txInfoKey, pb.TransactionStatus_FAILURE)
	res = im.Report(id0, 1)
	assert.True(t, res.Ok)

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).Return(false).AnyTimes()

	mockStub.EXPECT().Get(id0).Return(false, nil)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("cannot get global id of child tx id %s", id0), string(res.Result))

	globalId := "globalId"
	globalInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, globalId)
	mockStub.EXPECT().Get(id0).Return(true, []byte(globalId)).AnyTimes()
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).Return(false)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction global id %s does not exist", globalId), string(res.Result))

	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_SUCCESS,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction with global Id %s is finished", globalId), string(res.Result))

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("%s is not in transaction %s, %v", id0, globalId, txInfo), string(res.Result))

	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_SUCCESS
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.Report(id0, 0)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("%s has already reported result", id0), string(res.Result))

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_BEGIN
	expTxInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	expTxInfo.ChildTxInfo[id0] = pb.TransactionStatus_SUCCESS
	expTxInfo.ChildTxInfo[id1] = pb.TransactionStatus_BEGIN
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	mockStub.EXPECT().SetObject(globalInfoKey, expTxInfo).MaxTimes(1)
	res = im.Report(id0, 0)
	assert.True(t, res.Ok)

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	expTxInfo.GlobalState = pb.TransactionStatus_SUCCESS
	expTxInfo.ChildTxInfo[id0] = pb.TransactionStatus_SUCCESS
	expTxInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	mockStub.EXPECT().SetObject(globalInfoKey, expTxInfo).MaxTimes(1)
	res = im.Report(id0, 0)
	assert.True(t, res.Ok)

	txInfo.GlobalState = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	expTxInfo.GlobalState = pb.TransactionStatus_FAILURE
	expTxInfo.ChildTxInfo[id0] = pb.TransactionStatus_FAILURE
	expTxInfo.ChildTxInfo[id1] = pb.TransactionStatus_SUCCESS
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	mockStub.EXPECT().SetObject(globalInfoKey, expTxInfo).MaxTimes(1)
	res = im.Report(id0, 1)
	assert.True(t, res.Ok)
}

func TestTransactionManager_GetStatus(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id := "id"
	txInfoKey := fmt.Sprintf("%s-%s", PREFIX, id)
	globalInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, id)

	im := &TransactionManager{mockStub}

	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).SetArg(1, pb.TransactionStatus_SUCCESS).Return(true).MaxTimes(1)
	res := im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "1", string(res.Result))

	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	mockStub.EXPECT().GetObject(txInfoKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "0", string(res.Result))

	mockStub.EXPECT().GetObject(globalInfoKey, gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Get(id).Return(false, nil).MaxTimes(1)
	res = im.GetStatus(id)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("cannot get global id of child tx id %s", id), string(res.Result))

	globalId := "globalId"
	globalIdInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, globalId)
	mockStub.EXPECT().Get(id).Return(true, []byte(globalId)).AnyTimes()
	mockStub.EXPECT().GetObject(globalIdInfoKey, gomock.Any()).Return(false).MaxTimes(1)
	res = im.GetStatus(id)
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("transaction info for global id %s does not exist", globalId), string(res.Result))

	mockStub.EXPECT().GetObject(globalIdInfoKey, gomock.Any()).SetArg(1, txInfo).Return(true).MaxTimes(1)
	res = im.GetStatus(id)
	assert.True(t, res.Ok)
	assert.Equal(t, "0", string(res.Result))
}

func TestTransactionManager_BeginMultiTXs(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := "id0"
	id1 := "id1"
	globalId := "globalId"
	txInfoKey := fmt.Sprintf("%s-%s", PREFIX, globalId)
	globalInfoKey := fmt.Sprintf("global-%s-%s", PREFIX, globalId)

	im := &TransactionManager{mockStub}

	mockStub.EXPECT().Has(txInfoKey).Return(true).MaxTimes(1)
	res := im.BeginMultiTXs(globalId, id0, id1)
	assert.False(t, res.Ok)
	assert.Equal(t, "Transaction id already exists", string(res.Result))

	mockStub.EXPECT().Has(txInfoKey).Return(false).AnyTimes()
	mockStub.EXPECT().Set(id0, []byte(globalId))
	mockStub.EXPECT().Set(id1, []byte(globalId))
	txInfo := TransactionInfo{
		GlobalState: pb.TransactionStatus_BEGIN,
		ChildTxInfo: make(map[string]pb.TransactionStatus),
	}
	txInfo.ChildTxInfo[id0] = pb.TransactionStatus_BEGIN
	txInfo.ChildTxInfo[id1] = pb.TransactionStatus_BEGIN
	mockStub.EXPECT().SetObject(globalInfoKey, txInfo)
	res = im.BeginMultiTXs(globalId, id0, id1)
	assert.True(t, res.Ok)
}

func TestAssetExchange_Init(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	from := types.NewAddress([]byte{0}).String()
	to := types.NewAddress([]byte{1}).String()

	ae := &AssetExchange{mockStub}

	res := ae.Init(from, to, []byte{1})
	assert.False(t, res.Ok)

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    0,
	}
	info, err := aei.Marshal()
	assert.Nil(t, err)

	mockStub.EXPECT().Has(AssetExchangeKey(aei.Id)).Return(true).MaxTimes(1)
	res = ae.Init(from, to, info)
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exhcange id already exists", string(res.Result))

	mockStub.EXPECT().Has(AssetExchangeKey(aei.Id)).Return(false).AnyTimes()
	res = ae.Init(from, to, info)
	assert.False(t, res.Ok)
	assert.Equal(t, "illegal asset exchange info", string(res.Result))

	aei.AssetOnDst = 100
	info, err = aei.Marshal()
	assert.Nil(t, err)

	aer := AssetExchangeRecord{
		Chain0: from,
		Chain1: to,
		Status: AssetExchangeInit,
		Info:   aei,
	}
	mockStub.EXPECT().SetObject(AssetExchangeKey(aei.Id), aer)
	res = ae.Init(from, to, info)
	assert.True(t, res.Ok)
}

func TestAssetExchange_Redeem(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	from := types.NewAddress([]byte{0}).String()
	to := types.NewAddress([]byte{1}).String()

	ae := &AssetExchange{mockStub}

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    100,
	}

	aer := AssetExchangeRecord{
		Chain0: from,
		Chain1: to,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).Return(false).MaxTimes(1)
	res := ae.Redeem(from, to, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange record does not exist", string(res.Result))

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).MaxTimes(1)
	res = ae.Redeem(from, to, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange status for this id is not 'Init'", string(res.Result))

	aer.Status = AssetExchangeInit
	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).AnyTimes()
	res = ae.Redeem(to, to, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("invalid participator of asset exchange id %s", aei.Id), string(res.Result))

	expAer := AssetExchangeRecord{
		Chain0: from,
		Chain1: to,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}
	mockStub.EXPECT().SetObject(AssetExchangeKey(aei.Id), expAer)
	res = ae.Redeem(from, to, []byte(aei.Id))
	assert.True(t, res.Ok)
}

func TestAssetExchange_Refund(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	from := types.NewAddress([]byte{0}).String()
	to := types.NewAddress([]byte{1}).String()

	ae := &AssetExchange{mockStub}

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    100,
	}

	aer := AssetExchangeRecord{
		Chain0: from,
		Chain1: to,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).Return(false).MaxTimes(1)
	res := ae.Refund(from, to, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange record does not exist", string(res.Result))

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).MaxTimes(1)
	res = ae.Refund(from, to, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange status for this id is not 'Init'", string(res.Result))

	aer.Status = AssetExchangeInit
	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).AnyTimes()
	res = ae.Refund(to, to, []byte(aei.Id))
	assert.False(t, res.Ok)
	assert.Equal(t, fmt.Sprintf("invalid participator of asset exchange id %s", aei.Id), string(res.Result))

	expAer := AssetExchangeRecord{
		Chain0: from,
		Chain1: to,
		Status: AssetExchangeRefund,
		Info:   aei,
	}
	mockStub.EXPECT().SetObject(AssetExchangeKey(aei.Id), expAer)
	res = ae.Refund(from, to, []byte(aei.Id))
	assert.True(t, res.Ok)
}

func TestAssetExchange_GetStatus(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	aei := pb.AssetExchangeInfo{
		Id:            "123",
		SenderOnSrc:   "aliceSrc",
		ReceiverOnSrc: "bobSrc",
		AssetOnSrc:    10,
		SenderOnDst:   "bobDst",
		ReceiverOnDst: "aliceDst",
		AssetOnDst:    100,
	}

	from := types.NewAddress([]byte{0}).String()
	to := types.NewAddress([]byte{1}).String()
	aer := AssetExchangeRecord{
		Chain0: from,
		Chain1: to,
		Status: AssetExchangeRedeem,
		Info:   aei,
	}

	ae := &AssetExchange{mockStub}

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).Return(false).MaxTimes(1)
	res := ae.GetStatus(aei.Id)
	assert.False(t, res.Ok)
	assert.Equal(t, "asset exchange record does not exist", string(res.Result))

	mockStub.EXPECT().GetObject(AssetExchangeKey(aei.Id), gomock.Any()).SetArg(1, aer).Return(true).MaxTimes(1)
	res = ae.GetStatus(aei.Id)
	assert.True(t, res.Ok)
	assert.Equal(t, "1", string(res.Result))
}

func TestInterRelayBroker_InvokeInterRelayContract(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&boltvm.Response{
		Ok:     true,
		Result: nil,
	}).AnyTimes()

	realArgs := [2][]byte{[]byte("123"), []byte("123454")}
	args, err := json.Marshal(realArgs)
	require.Nil(t, err)

	interRelayBroker := InterRelayBroker{mockStub}
	res := interRelayBroker.InvokeInterRelayContract(constant.DIDRegistryContractAddr.String(), "", args)
	require.False(t, res.Ok)

	res = interRelayBroker.InvokeInterRelayContract(constant.MethodRegistryContractAddr.String(), "Synchronize", args)
	require.True(t, res.Ok)

	res = interRelayBroker.GetInCouterMap()
	require.True(t, res.Ok)

	res = interRelayBroker.GetOutCouterMap()
	require.True(t, res.Ok)

	res = interRelayBroker.GetOutMessageMap()
	require.True(t, res.Ok)

	ibtp := &pb.IBTP{
		From: "123",
		To:   "123",
		Type: pb.IBTP_INTERCHAIN,
	}

	ibtps := &pb.IBTPs{Ibtps: []*pb.IBTP{ibtp}}
	data, err := ibtps.Marshal()
	require.Nil(t, err)

	res = interRelayBroker.RecordIBTPs(data)
	require.True(t, res.Ok)

	res = interRelayBroker.GetOutMessage("123", 1)
	require.True(t, res.Ok)
}

func TestGovernance_SubmitProposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}
	proposalExistent := &Proposal{
		Id:         idExistent,
		Des:        "des",
		Typ:        AppchainMgr,
		Status:     PROPOSED,
		BallotMap:  map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum: 1,
		AgainstNum: 1,
	}
	pData, err := json.Marshal(proposalExistent)
	assert.Nil(t, err)
	pDatas := make([][]byte, 0)
	pDatas = append(pDatas, pData)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "addr1",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr2",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr3",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr4",
			Weight:  1,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)
	adminsErrorData := make([]byte, 0)

	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsErrorData)).Times(2)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()

	// check permission error
	res := g.SubmitProposal("", "des", string(AppchainMgr), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles error
	res = g.SubmitProposal(idExistent, "des", string(AppchainMgr), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	// GetAdminRoles unmarshal error
	res = g.SubmitProposal(idExistent, "des", string(AppchainMgr), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, "des", "", []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.SubmitProposal(idExistent, "des", string(AppchainMgr), []byte{})
	assert.True(t, res.Ok, string(res.Result))

}
func TestGovernance_Proposal(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}

	idExistent := "idExistent-1"
	idNonexistent := "idNonexistent-2"
	idClosed := "idClosed-3"
	idNotReachThreshold := "idNotReachThreshold-4"
	idSuperMajorityApprove := "idSuperMajorityApprove-5"
	idSuperMajorityAgainst := "idSuperMajorityAgainst-6"
	idUnupportedType := "idUnsupportedType-7"
	addrApproved := "addrApproved"
	addrAganisted := "addrAganisted"
	addrNotVoted := "addrNotVoted"
	addrNotVoted1 := "addrNotVoted1"
	addrNotVoted2 := "addrNotVoted2"
	addrNotVoted3 := "addrNotVoted3"
	addrNotVoted4 := "addrNotVoted4"
	addrNotVoted5 := "addrNotVoted5"
	addrNotVoted6 := "addrNotVoted6"

	approveBallot := Ballot{
		VoterAddr: addrApproved,
		Approve:   BallotApprove,
		Num:       1,
		Reason:    "",
	}
	againstBallot := Ballot{
		VoterAddr: addrAganisted,
		Approve:   BallotReject,
		Num:       1,
		Reason:    "",
	}
	proposalExistent := Proposal{
		Id:            idExistent,
		Des:           "des",
		Typ:           AppchainMgr,
		Status:        PROPOSED,
		BallotMap:     map[string]Ballot{addrApproved: approveBallot, addrAganisted: againstBallot},
		ApproveNum:    1,
		AgainstNum:    1,
		ElectorateNum: 4,
		ThresholdNum:  3,
	}

	pData, err := json.Marshal(proposalExistent)
	assert.Nil(t, err)
	pDatas := make([][]byte, 0)
	pDatas = append(pDatas, pData)

	admins := []*repo.Admin{
		&repo.Admin{
			Address: "addr1",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr2",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr3",
			Weight:  1,
		},
		&repo.Admin{
			Address: "addr4",
			Weight:  1,
		},
	}
	adminsData, err := json.Marshal(admins)
	assert.Nil(t, err)

	mockStub.EXPECT().Has(ProposalKey(idExistent)).Return(true).AnyTimes()
	mockStub.EXPECT().Has(ProposalKey(idNonexistent)).Return(false).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idExistent), gomock.Any()).SetArg(1, proposalExistent).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idClosed), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idClosed
			pro.Des = proposalExistent.Des
			pro.Typ = proposalExistent.Typ
			pro.Status = APPOVED
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNonexistent), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idNotReachThreshold), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idNotReachThreshold
			pro.Des = proposalExistent.Des
			pro.Typ = RuleMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = 4
			pro.ThresholdNum = 4
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idSuperMajorityApprove), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idSuperMajorityApprove
			pro.Des = proposalExistent.Des
			pro.Typ = NodeMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = proposalExistent.ElectorateNum
			pro.ThresholdNum = proposalExistent.ThresholdNum
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idSuperMajorityAgainst), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idSuperMajorityAgainst
			pro.Des = proposalExistent.Des
			pro.Typ = NodeMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = proposalExistent.ElectorateNum
			pro.ThresholdNum = proposalExistent.ThresholdNum
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProposalKey(idUnupportedType), gomock.Any()).Do(
		func(id string, ret interface{}) bool {
			pro := ret.(*Proposal)
			pro.Id = idUnupportedType
			pro.Des = proposalExistent.Des
			pro.Typ = ServiceMgr
			pro.Status = proposalExistent.Status
			pro.BallotMap = proposalExistent.BallotMap
			pro.ApproveNum = proposalExistent.ApproveNum
			pro.AgainstNum = proposalExistent.AgainstNum
			pro.ElectorateNum = proposalExistent.ElectorateNum
			pro.ThresholdNum = proposalExistent.ThresholdNum
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(string(AppchainMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(RuleMgr), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			proStrategy := ret.(*ProposalStrategy)
			proStrategy.Typ = SimpleMajority
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(string(NodeMgr), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			proStrategy := ret.(*ProposalStrategy)
			proStrategy.Typ = SuperMajorityApprove
			return true
		}).Return(true).Times(1)
	mockStub.EXPECT().GetObject(string(NodeMgr), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			proStrategy := ret.(*ProposalStrategy)
			proStrategy.Typ = SuperMajorityAgainst
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(string(ServiceMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().Query(gomock.Any()).Return(true, pDatas).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRoleWeight", gomock.Any()).Return(boltvm.Error("get role weight")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRoleWeight", gomock.Any()).Return(boltvm.Success([]byte(""))).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetRoleWeight", gomock.Any()).Return(boltvm.Success([]byte(strconv.Itoa(1)))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "GetAdminRoles").Return(boltvm.Success(adminsData)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manager", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "Manager", gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	res := g.ModifyProposal(idExistent, "des", string(AppchainMgr), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.ModifyProposal(idExistent, "des", string(AppchainMgr), []byte{})
	assert.True(t, res.Ok, string(res.Result))
	res = g.ModifyProposal(idNonexistent, "des", string(AppchainMgr), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.ModifyProposal("", "des", string(AppchainMgr), []byte{})
	assert.False(t, res.Ok, string(res.Result))
	res = g.ModifyProposal(idExistent, "des", "", []byte{})
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposal(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposal(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalsByFrom("idExistent")
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByTyp("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalsByTyp(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetProposalsByStatus("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalsByStatus(string((PROPOSED)))
	assert.True(t, res.Ok, string(res.Result))

	res = g.GetDes(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetDes(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetTyp(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetTyp(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetStatus(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetStatus(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetApprove(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetApprove(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetAgainst(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetAgainst(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetVotedNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetVotedNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetVoted(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetVoted(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetApproveNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetApproveNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetAgainstNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetAgainstNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetElectorateNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetElectorateNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetThresholdNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetThresholdNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))

	var v = &Ballot{}
	res = g.GetBallot(addrApproved, idNonexistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetBallot(addrNotVoted, idExistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetBallot(addrApproved, idExistent)
	assert.True(t, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, v)
	assert.Nil(t, err)
	assert.Equal(t, BallotApprove, v.Approve)
	assert.Equal(t, uint64(1), v.Num)
	res = g.GetBallot(addrAganisted, idExistent)
	assert.True(t, res.Ok, string(res.Result))
	err = json.Unmarshal(res.Result, v)
	assert.Nil(t, err)
	assert.Equal(t, BallotReject, v.Approve)
	assert.Equal(t, uint64(1), v.Num)

	res = g.GetUnvote(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetUnvote(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetUnvoteNum(idNonexistent)
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetUnvoteNum(idExistent)
	assert.True(t, res.Ok, string(res.Result))
	mockStub.EXPECT().Caller().Return(addrApproved).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrApproved).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted1).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted2).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted3).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted4).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted5).Times(1)
	mockStub.EXPECT().Caller().Return(addrNotVoted6).Times(1)

	// nonexistent error
	res = g.Vote(idNonexistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// closed error
	res = g.Vote(idClosed, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// has voted error
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))

	// get weight error
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// get weight parse int error
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))

	// not reach threshold (approve:1)
	res = g.Vote(idNotReachThreshold, BallotApprove, "")
	assert.True(t, res.Ok, string(res.Result))
	// SuperMajorityApprove (reject:1)
	res = g.Vote(idSuperMajorityApprove, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// SuperMajorityAgainst (approve:2)
	res = g.Vote(idSuperMajorityAgainst, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// UnupportedType (reject:2)
	res = g.Vote(idUnupportedType, BallotReject, "")
	assert.False(t, res.Ok, string(res.Result))
	// Manager error (approve:3)
	res = g.Vote(idExistent, BallotApprove, "")
	assert.False(t, res.Ok, string(res.Result))
	// reject (reject:3)
	res = g.Vote(idExistent, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
	// approve (approve:4)
	res = g.Vote(idExistent, BallotReject, "")
	assert.True(t, res.Ok, string(res.Result))
}

func TestGovernance_ProposalStrategy(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	g := Governance{mockStub}
	ps := &ProposalStrategy{
		Typ:                  SimpleMajority,
		ParticipateThreshold: 0.5,
	}
	psData, err := json.Marshal(ps)
	assert.Nil(t, err)

	psError := &ProposalStrategy{
		Typ:                  SimpleMajority,
		ParticipateThreshold: 1.5,
	}
	psErrorData, err := json.Marshal(psError)
	assert.Nil(t, err)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetObject(string(RuleMgr), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(string(AppchainMgr), gomock.Any()).Return(true).AnyTimes()

	res := g.NewProposalStrategy(string(SimpleMajority), 0.5, []byte{})
	assert.True(t, res.Ok, string(res.Result))
	res = g.NewProposalStrategy("", 0.5, []byte{})
	assert.False(t, res.Ok, string(res.Result))

	res = g.SetProposalStrategy(string(AppchainMgr), psData)
	assert.True(t, res.Ok, string(res.Result))
	res = g.SetProposalStrategy("", psData)
	assert.False(t, res.Ok, string(res.Result))
	res = g.SetProposalStrategy(string(AppchainMgr), psErrorData)
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalStrategy(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategy("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategy(string(RuleMgr))
	assert.False(t, res.Ok, string(res.Result))

	res = g.GetProposalStrategyType(string(AppchainMgr))
	assert.True(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategyType("")
	assert.False(t, res.Ok, string(res.Result))
	res = g.GetProposalStrategyType(string(RuleMgr))
	assert.False(t, res.Ok, string(res.Result))
}
