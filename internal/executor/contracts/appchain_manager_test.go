package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/stretchr/testify/assert"
)

const (
	method = "appchain1"
)

func TestAppchainManager_IsAppchainAdmin(t *testing.T) {
	am, mockStub, chains, chainsData := prepare(t)

	addr, err := getAddr(chains[0].PublicKey)
	assert.Nil(t, err)
	mockStub.EXPECT().Caller().Return(addr).AnyTimes()
	mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(true, chainsData).AnyTimes()

	res := am.IsAppchainAdmin()
	assert.Equal(t, true, res.Ok)
}

func TestAppchainManager_Appchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	chain := &appchainMgr.Appchain{
		ID:            appchainMethod,
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

	mockStub.EXPECT().Get("appchain-"+appchainMethod).Return(true, data)
	mockStub.EXPECT().Get("appchain-"+appchainMethod2).Return(false, nil)

	am := &AppchainManager{
		Stub: mockStub,
	}

	res := am.GetAppchain(appchainMethod)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, data, res.Result)

	res = am.GetAppchain(appchainMethod2)
	assert.Equal(t, false, res.Ok)
}

func TestAppchainManager_Appchains(t *testing.T) {
	am, mockStub, chains, chainsData := prepare(t)

	logger := log.NewWithModule("contracts")
	//applyResponse := &boltvm.Response{
	//	Ok:     true,
	//	Result: []byte("OK"),
	//}

	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))
	//mockStub.EXPECT().CrossInvoke(constant.MethodRegistryContractAddr.String(), "Apply",
	//	gomock.Any(), gomock.Any(), gomock.Any()).Return(applyResponse)
	mockStub.EXPECT().Has(AppchainKey(appchainMethod)).Return(false).MaxTimes(3)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			chain := ret.(*appchainMgr.Appchain)
			chain.ID = chains[2].ID
			chain.Status = chains[2].Status
			return true
		}).Return(true).Times(1)

	am.Register(method, docAddr, docHash,
		chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
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
	assert.Equal(t, 3, len(chains))
	assert.Equal(t, chains[0], appchains[0])
	assert.Equal(t, chains[1], appchains[1])

	res = am.Appchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, []byte(nil), res.Result)

	// counter chains
	res = am.CountAppchains()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "3", string(res.Result))

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
			chain.ID = chains[2].ID
			chain.Status = chains[2].Status
			return true
		}).Return(true).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			chain := ret.(*appchainMgr.Appchain)
			chain.ID = chains[2].ID
			chain.Status = chains[0].Status
			return true
		}).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(logger).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("").AnyTimes()
	//mockStub.EXPECT().CrossInvoke(constant.MethodRegistryContractAddr.String(), "Apply",
	//	gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	res := am.Register(method, docAddr, docHash,
		chains[2].Validators, chains[2].ConsensusType, chains[2].ChainType,
		chains[2].Name, chains[2].Desc, chains[2].Version, chains[2].PublicKey)
	assert.True(t, res.Ok)

	// test for repeated register
	res = am.Register(method, docAddr, docHash,
		chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)
	assert.False(t, res.Ok)
}

func TestAppchainManager_Manager(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	am := &AppchainManager{
		Stub: mockStub,
	}

	chain := &appchainMgr.Appchain{
		Status:        governance.GovernanceUpdating,
		ID:            appchainMethod,
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
		Status:        governance.GovernanceUpdating,
		ID:            appchainMethod2,
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

	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(appchainMethod)).Return(true, data).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(appchainMethod2)).Return(false, nil).AnyTimes()
	mockStub.EXPECT().Has(AppchainKey(appchainMethod)).Return(true).AnyTimes()
	mockStub.EXPECT().Has(AppchainKey(appchainMethod2)).Return(false).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return("addrNotAdmin").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.String()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RuleManagerContractAddr.String(), "DefaultRule", gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	//mockStub.EXPECT().CrossInvoke(constant.MethodRegistryContractAddr.String(), "AuditApply",
	//	gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))
	//mockStub.EXPECT().CrossInvoke(constant.MethodRegistryContractAddr.String(), "Register",
	//	gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))

	// test without permission
	res := am.Manage(string(governance.EventUpdate), string(APPOVED), string(governance.GovernanceAvailable), data)
	assert.False(t, res.Ok)
	// test with permission
	res = am.Manage(string(governance.EventUpdate), string(APPOVED), string(governance.GovernanceAvailable), data1)
	assert.False(t, res.Ok)
	res = am.Manage(string(governance.EventUpdate), string(REJECTED), string(governance.GovernanceAvailable), data1)
	assert.False(t, res.Ok)
	res = am.Manage(string(governance.EventUpdate), string(APPOVED), string(governance.GovernanceAvailable), data)
	assert.True(t, res.Ok, string(res.Result))
	res = am.Manage(string(governance.EventUpdate), string(REJECTED), string(governance.GovernanceAvailable), data)
	assert.True(t, res.Ok, string(res.Result))

	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "Register", gomock.Any()).Return(boltvm.Error("")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.InterchainContractAddr.String(), "Register", gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	res = am.Manage(string(governance.EventRegister), string(APPOVED), string(governance.GovernanceUnavailable), data)
	assert.False(t, res.Ok)
	res = am.Manage(string(governance.EventRegister), string(APPOVED), string(governance.GovernanceUnavailable), data)
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
	mockStub.EXPECT().Get(AppchainKey(appchainMethod)).Return(true, chainsData[0]).AnyTimes()
	mockStub.EXPECT().Get(AppchainKey(appchainMethod2)).Return(true, chainsData[1]).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()

	availableChain := chains[0]
	availableChain.Status = governance.GovernanceAvailable
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *availableChain).Return(true).Times(2)
	frozenChain := chains[1]
	frozenChain.Status = governance.GovernanceFrozen
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *frozenChain).Return(true).AnyTimes()

	// test UpdateAppchain
	res := am.UpdateAppchain(appchainMethod, docAddr, docHash,
		chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test FreezeAppchain
	res = am.FreezeAppchain(appchainMethod)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain(appchainMethod2)
	assert.Equal(t, true, res.Ok, string(res.Result))
	// test LogoutAppchain
	res = am.LogoutAppchain(appchainMethod)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestManageChain_WithoutPermission(t *testing.T) {
	am, mockStub, _, chainsData := prepare(t)
	mockStub.EXPECT().Caller().Return(caller).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(caller).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("")).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(true, chainsData[0]).AnyTimes()

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
	res := am.UpdateAppchain(appchainMethod, docAddr, docHash,
		chains[0].Validators, chains[0].ConsensusType, chains[0].ChainType,
		chains[0].Name, chains[0].Desc, chains[0].Version, chains[0].PublicKey)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test FreezeAppchain
	res = am.FreezeAppchain(caller)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test ActivateAppchain
	res = am.ActivateAppchain(caller)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// test LogoutAppchain
	res = am.LogoutAppchain(caller)
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
	//mockStub.EXPECT().CrossInvoke(constant.MethodRegistryContractAddr.String(), "Delete",
	//	gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil))

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
	mockStub.EXPECT().Has(AppchainKey(appchainMethod)).Return(true)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Do(
		func(key string, ret interface{}) bool {
			chain := ret.(*appchainMgr.Appchain)
			chain.Status = governance.GovernanceAvailable
			chain.PublicKey = chains[0].PublicKey
			assert.Equal(t, key, AppchainKey(appchainMethod))
			fmt.Printf("chain is %v", chain)
			return true
		}).AnyTimes()
	res := am.GetPubKeyByChainID(appchainMethod)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, chains[0].PublicKey, string(res.Result))
}

func TestAppchainManager_GetIdByAddr(t *testing.T) {
	am, mockStub, _, _ := prepare(t)
	// test for GetIdByAddr
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true)

	res := am.GetIdByAddr("")
	assert.Equal(t, true, res.Ok)
}

func prepare(t *testing.T) (*AppchainManager, *mock_stub.MockStub, []*appchainMgr.Appchain, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	am := &AppchainManager{
		Stub: mockStub,
	}

	var chains []*appchainMgr.Appchain
	var chainsData [][]byte
	chainType := []string{string(governance.GovernanceAvailable), string(governance.GovernanceFrozen), string(governance.GovernanceUnavailable)}

	chainAdminKeyPath, err := repo.PathRootWithDefault("../../../tester/test_data/appchain1.json")
	assert.Nil(t, err)
	pubKey, err := getPubKey(chainAdminKeyPath)
	assert.Nil(t, err)

	for i := 0; i < 3; i++ {
		addr := appchainMethod + types.NewAddress([]byte{byte(i)}).String()

		chain := &appchainMgr.Appchain{
			Status:        governance.GovernanceStatus(chainType[i]),
			ID:            addr,
			Name:          "appchain" + addr,
			Validators:    "",
			ConsensusType: "",
			ChainType:     "fabric",
			Desc:          "",
			Version:       "",
			PublicKey:     pubKey,
		}

		data, err := json.Marshal(chain)
		assert.Nil(t, err)

		chainsData = append(chainsData, data)
		chains = append(chains, chain)
	}

	return am, mockStub, chains, chainsData
}
