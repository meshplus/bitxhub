package contracts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

func TestAppchainManager_Appchain(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addr0 := types.NewAddress([]byte{0}).String()
	addr1 := types.NewAddress([]byte{1}).String()

	chain := &appchainMgr.Appchain{
		ID:            addr0,
		Name:          "appchain A",
		Validators:    "",
		ConsensusType: 0,
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
			ID:            addr,
			Name:          "appchain" + addr,
			Validators:    "",
			ConsensusType: int32(i),
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

	o1 := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(true, chainsData)
	o2 := mockStub.EXPECT().Query(appchainMgr.PREFIX).Return(false, nil)
	gomock.InOrder(o1, o2)

	am := &AppchainManager{
		Stub: mockStub,
	}

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
}

func TestInterchainManager_Register(t *testing.T) {
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

	res := im.Register()
	assert.Equal(t, true, res.Ok)

	ic := &pb.Interchain{}
	err = ic.Unmarshal(res.Result)
	assert.Nil(t, err)
	assert.Equal(t, addr, ic.ID)
	assert.Equal(t, 0, len(ic.InterchainCounter))
	assert.Equal(t, 0, len(ic.ReceiptCounter))
	assert.Equal(t, 0, len(ic.SourceReceiptCounter))

	res = im.Register()
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, data0, res.Result)

	res = im.Register()
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
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()
	f1 := mockStub.EXPECT().Get(appchainMgr.PREFIX+from).Return(false, nil)

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

	f2 := mockStub.EXPECT().Get(appchainMgr.PREFIX+from).Return(true, data0).AnyTimes()
	mockStub.EXPECT().Get(appchainMgr.PREFIX+to).Return(true, data0).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().AddObject(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxIndex().Return(uint64(1)).AnyTimes()
	mockStub.EXPECT().PostInterchainEvent(gomock.Any()).AnyTimes()
	mockStub.EXPECT().GetTxHash().Return(&types.Hash{}).AnyTimes()
	gomock.InOrder(f1, f2)

	im := &InterchainManager{mockStub}

	ibtp := &pb.IBTP{
		From: from,
	}

	res := im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, "this appchain does not exist", string(res.Result))

	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, "empty destination chain id", string(res.Result))

	ibtp = &pb.IBTP{
		From:      from,
		To:        to,
		Index:     0,
		Type:      pb.IBTP_INTERCHAIN,
		Timestamp: 0,
		Proof:     nil,
		Payload:   nil,
		Version:   "",
		Extra:     nil,
	}
	mockStub.EXPECT().Caller().Return(ibtp.To)

	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, "ibtp from != caller", string(res.Result))

	mockStub.EXPECT().Caller().Return(ibtp.From).MaxTimes(6)

	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, "index already exists, required 2, but 0", string(res.Result))

	ibtp.Index = 2
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_ASSET_EXCHANGE_INIT
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_ASSET_EXCHANGE_REFUND
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_ASSET_EXCHANGE_REDEEM
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_RECEIPT_SUCCESS
	res = im.HandleIBTP(ibtp)
	assert.False(t, res.Ok)
	assert.Equal(t, "ibtp to != caller", string(res.Result))

	mockStub.EXPECT().Caller().Return(ibtp.To).AnyTimes()

	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_RECEIPT_FAILURE
	res = im.HandleIBTP(ibtp)
	assert.True(t, res.Ok)

	ibtp.Type = pb.IBTP_ASSET_EXCHANGE_RECEIPT
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

func TestRole_GetRole(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addrs := []string{types.NewAddress([]byte{0}).String(), types.NewAddress([]byte{1}).String()}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, addrs).AnyTimes()
	mockStub.EXPECT().Caller().Return(types.NewAddress([]byte{0}).String())

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

	addrs := []string{types.NewAddress([]byte{0}).String(), types.NewAddress([]byte{1}).String()}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, addrs).AnyTimes()

	im := &Role{mockStub}

	res := im.IsAdmin(addrs[0])
	assert.True(t, res.Ok)
	assert.Equal(t, "true", string(res.Result))

	res = im.IsAdmin(types.NewAddress([]byte{2}).String())
	assert.True(t, res.Ok)
	assert.Equal(t, "false", string(res.Result))
}

func TestRole_GetAdminRoles(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addrs := []string{types.NewAddress([]byte{0}).String(), types.NewAddress([]byte{1}).String()}

	mockStub.EXPECT().GetObject(adminRolesKey, gomock.Any()).SetArg(1, addrs).AnyTimes()

	im := &Role{mockStub}

	res := im.GetAdminRoles()
	assert.True(t, res.Ok)

	var admins []string
	err := json.Unmarshal(res.Result, &admins)
	assert.Nil(t, err)
	assert.Equal(t, len(addrs), len(admins))
	for i, addr := range addrs {
		assert.Equal(t, addr, admins[i])
	}
}

func TestRole_SetAdminRoles(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	addrs := []string{types.NewAddress([]byte{0}).String(), types.NewAddress([]byte{1}).String()}
	mockStub.EXPECT().SetObject(adminRolesKey, addrs).AnyTimes()

	im := &Role{mockStub}

	data, err := json.Marshal(addrs)
	assert.Nil(t, err)

	res := im.SetAdminRoles(string(data))
	assert.True(t, res.Ok)
}

func TestRuleManager_RegisterRule(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	id0 := types.NewAddress([]byte{0}).String()
	id1 := types.NewAddress([]byte{1}).String()

	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(id0)).Return(boltvm.Success(nil))
	mockStub.EXPECT().CrossInvoke(constant.AppchainMgrContractAddr.String(), "GetAppchain", pb.String(id1)).Return(boltvm.Error(""))
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).AnyTimes()

	im := &RuleManager{mockStub}

	addr := types.NewAddress([]byte{2}).String()
	res := im.RegisterRule(id0, addr)
	assert.True(t, res.Ok)

	res = im.RegisterRule(id1, addr)
	assert.False(t, res.Ok)
	assert.Equal(t, "this appchain does not exist", string(res.Result))
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
