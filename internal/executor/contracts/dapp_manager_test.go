package contracts

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

const (
	ownerAddr  = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f1"
	ownerAddr1 = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f2"
	dappID     = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f1-0"
	conAddr1   = "0x12342e0b3189132D815b8eb325bE17285AC898f1"
	conAddr2   = "0x12342e0b3189132D815b8eb325bE17285AC898f2"
)

func TestDappManager_Manage(t *testing.T) {
	dm, mockStub, dapps, dappsData := dappPrepare(t)

	mockStub.EXPECT().CurrentCaller().Return("addrNoPermission").Times(1)
	mockStub.EXPECT().CurrentCaller().Return(constant.GovernanceContractAddr.Address().String()).AnyTimes()
	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).SetArg(1, *dapps[0]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(DappKey(dapps[3].DappID), gomock.Any()).SetArg(1, *dapps[3]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(DappKey(dapps[4].DappID), gomock.Any()).SetArg(1, *dapps[4]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(DappKey(dapps[5].DappID), gomock.Any()).SetArg(1, *dapps[5]).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(OwnerKey(ownerAddr), gomock.Any()).SetArg(1, map[string]struct{}{}).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(OwnerKey(ownerAddr1), gomock.Any()).SetArg(1, map[string]struct{}{}).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Delete(gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()

	// test without permission
	res := dm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), dapps[0].DappID, dappsData[0])
	assert.False(t, res.Ok, string(res.Result))
	// test changestatus error
	res = dm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), dapps[0].DappID, dappsData[0])
	assert.False(t, res.Ok, string(res.Result))

	// test register
	res = dm.Manage(string(governance.EventRegister), string(APPROVED), string(governance.GovernanceUnavailable), dapps[3].DappID, dappsData[3])
	assert.True(t, res.Ok, string(res.Result))

	// test update
	updateInfo := &UpdateDappInfo{
		DappName: UpdateInfo{
			OldInfo: dapps[0].Name,
			NewInfo: dapps[1].Name,
			IsEdit:  true,
		},
		Desc: UpdateInfo{
			OldInfo: dapps[0].Desc,
			NewInfo: dapps[1].Desc,
			IsEdit:  true,
		},
		Url: UpdateInfo{
			OldInfo: dapps[0].Url,
			NewInfo: dapps[1].Url,
			IsEdit:  true,
		},
		ContractAddr: UpdateMapInfo{
			OldInfo: dapps[0].ContractAddr,
			NewInfo: map[string]struct{}{conAddr2: {}},
			IsEdit:  true,
		},
		Permission: UpdateMapInfo{
			OldInfo: dapps[0].Permission,
			NewInfo: dapps[0].Permission,
			IsEdit:  false,
		},
	}
	updateInfoData, err := json.Marshal(updateInfo)
	assert.Nil(t, err)
	res = dm.Manage(string(governance.EventUpdate), string(APPROVED), string(governance.GovernanceAvailable), dapps[4].DappID, updateInfoData)
	assert.True(t, res.Ok, string(res.Result))

	// test transer
	transData, err := json.Marshal(dapps[5].TransferRecords[len(dapps[5].TransferRecords)-1])
	assert.Nil(t, err)
	res = dm.Manage(string(governance.EventTransfer), string(APPROVED), string(governance.GovernanceAvailable), dapps[5].DappID, transData)
	assert.True(t, res.Ok, string(res.Result))
}

func TestDappManager_RegisterDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	mockStub.EXPECT().GetObject(DappOccupyNameKey(dapps[0].Name), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(DappOccupyNameKey(dapps[1].Name), gomock.Any()).SetArg(1, dapps[1].DappID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(DappOccupyContractKey(conAddr1), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(DappOccupyContractKey(conAddr2), gomock.Any()).SetArg(1, dapps[1].DappID).Return(true).AnyTimes()

	governancePreErrReq := mockStub.EXPECT().GetObject(DappKey(dappID), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	submitErrReq := mockStub.EXPECT().GetObject(DappKey(dappID), gomock.Any()).Return(false).Times(2)
	okReq := mockStub.EXPECT().GetObject(DappKey(dappID), gomock.Any()).Return(false).AnyTimes()
	gomock.InOrder(governancePreErrReq, submitErrReq, okReq)

	mockStub.EXPECT().GetObject(OwnerKey(ownerAddr), gomock.Any()).SetArg(1, make(map[string]struct{})).Return(true).AnyTimes()

	mockStub.EXPECT().Caller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	account := mockAccount(t)
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// 1. check info error
	// url
	res := dm.RegisterDapp("", string(dapps[0].Type), dapps[0].Desc, " ", "", "-", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// name
	res = dm.RegisterDapp("", string(dapps[0].Type), dapps[0].Desc, "url", "", "-", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = dm.RegisterDapp(dapps[1].Name, string(dapps[0].Type), dapps[0].Desc, "url", "", "-", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// contract
	res = dm.RegisterDapp(dapps[1].Name, string(dapps[0].Type), dapps[0].Desc, "url", "123", "-", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	res = dm.RegisterDapp(dapps[1].Name, string(dapps[0].Type), dapps[0].Desc, "url", conAddr2, "-", reason)
	// premission
	res = dm.RegisterDapp(dapps[1].Name, string(dapps[0].Type), dapps[0].Desc, "url", conAddr1, "-", reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	// 2. governancePre error
	res = dm.RegisterDapp(dapps[0].Name, string(dapps[0].Type), dapps[0].Desc, "url", conAddr1, caller, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 3. submit error
	res = dm.RegisterDapp(dapps[0].Name, string(dapps[0].Type), dapps[0].Desc, "url", conAddr1, caller, reason)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = dm.RegisterDapp(dapps[0].Name, string(dapps[0].Type), dapps[0].Desc, "url", conAddr1, caller, reason)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestDappManager_UpdateDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	mockStub.EXPECT().GetObject(DappOccupyNameKey("newName"), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(DappOccupyNameKey(dapps[0].Name), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(DappOccupyNameKey(dapps[1].Name), gomock.Any()).SetArg(1, dapps[1].DappID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(DappOccupyContractKey(conAddr1), gomock.Any()).Return(false).AnyTimes()
	mockStub.EXPECT().GetObject(DappOccupyContractKey(conAddr2), gomock.Any()).SetArg(1, dapps[1].DappID).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).SetArg(1, *dapps[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Caller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	account := mockAccount(t)
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// 1. governancePre error
	res := dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Desc, "url", conAddr1, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	// 2. check permision error
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Desc, "url", conAddr1, "", reason)
	assert.False(t, res.Ok, string(res.Result))

	// 3. check info error
	//  name
	res = dm.UpdateDapp(dapps[0].DappID, "", dapps[0].Desc, "url", conAddr1, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	res = dm.UpdateDapp(dapps[0].DappID, dapps[1].Name, dapps[0].Desc, "url", conAddr1, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	// contract
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Desc, "url", "-", "", reason)
	assert.False(t, res.Ok, string(res.Result))
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Desc, "url", conAddr2, "", reason)
	assert.False(t, res.Ok, string(res.Result))
	// permission
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Desc, "url", conAddr1, "-", reason)
	assert.False(t, res.Ok, string(res.Result))

	// 4. no proposal
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Desc, "url", conAddr1, "", reason)
	assert.True(t, res.Ok, string(res.Result))

	// 5. submit error
	res = dm.UpdateDapp(dapps[0].DappID, "newName", dapps[0].Desc, "url", conAddr1, caller, reason)
	assert.False(t, res.Ok, string(res.Result))

	// 7. ok
	res = dm.UpdateDapp(dapps[0].DappID, "newName", dapps[0].Desc, "url", conAddr1, caller, reason)
	assert.True(t, res.Ok, string(res.Result))
}

func TestDappManager_TransferDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	checkPermissionErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	submitErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	changeStatusErrReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	changeStatusErrReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[1]).Return(true).Times(1)
	okReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(2)
	gomock.InOrder(governancePreErrReq, checkPermissionErrReq, submitErrReq, changeStatusErrReq1, changeStatusErrReq2, okReq)

	mockStub.EXPECT().Caller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	account := mockAccount(t)
	mockStub.EXPECT().GetAccount(gomock.Any()).Return(account).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Error("submit error")).Times(1)
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	// 1. illegal new owner addr
	res := dm.TransferDapp(dapps[0].DappID, "1", reason)
	assert.Equal(t, false, res.Ok)
	// 2. governancePre error
	res = dm.TransferDapp(dapps[0].DappID, ownerAddr1, reason)
	assert.Equal(t, false, res.Ok)
	// 3. check permision error
	res = dm.TransferDapp(dapps[0].DappID, ownerAddr1, reason)
	assert.Equal(t, false, res.Ok)
	// 4. submit error
	res = dm.TransferDapp(dapps[0].DappID, ownerAddr1, reason)
	assert.Equal(t, false, res.Ok)
	// 5. change status error
	res = dm.TransferDapp(dapps[0].DappID, ownerAddr1, reason)
	assert.Equal(t, false, res.Ok)

	res = dm.TransferDapp(dapps[0].DappID, ownerAddr1, reason)
	assert.Equal(t, true, res.Ok)
}

func TestDappManager_FreezeDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	mockStub.EXPECT().Caller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(appchainAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	res := dm.FreezeDapp(dapps[0].DappID, reason)
	assert.Equal(t, true, res.Ok)
}

func TestDappManager_ActivateDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	mockStub.EXPECT().Caller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[1]).Return(true).AnyTimes()
	mockStub.EXPECT().CurrentCaller().Return(appchainAdminAddr).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.RoleContractAddr.Address().String(), "IsAnyAvailableAdmin", pb.String(appchainAdminAddr), pb.String(string(GovernanceAdmin))).Return(boltvm.Success([]byte(TRUE))).AnyTimes()
	mockStub.EXPECT().CrossInvoke(constant.GovernanceContractAddr.Address().String(), "SubmitProposal", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success(nil)).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().Logger().Return(log.NewWithModule("contracts")).AnyTimes()

	res := dm.ActivateDapp(dapps[1].DappID, reason)
	assert.Equal(t, true, res.Ok)
}

func TestDappManager_ConfirmTransfer(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	mockStub.EXPECT().CurrentCaller().Return(noAdminAddr).Times(1)
	mockStub.EXPECT().CurrentCaller().Return(ownerAddr).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	// 1. get dapp error
	res := dm.ConfirmTransfer(dapps[0].DappID)
	assert.Equal(t, false, res.Ok)

	// 2. check permission error
	res = dm.ConfirmTransfer(dapps[0].DappID)
	assert.Equal(t, false, res.Ok)

	res = dm.ConfirmTransfer(dapps[0].DappID)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestDappManager_EvaluateDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	mockStub.EXPECT().GetTxTimeStamp().Return(int64(0)).AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).AnyTimes()
	mockStub.EXPECT().Caller().Return(ownerAddr).Times(1)
	mockStub.EXPECT().Caller().Return(ownerAddr1).AnyTimes()
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()

	// 1. illegal score
	res := dm.EvaluateDapp(dapps[0].DappID, dapps[0].Desc, 6)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 2. get dapp error
	res = dm.EvaluateDapp(dapps[0].DappID, dapps[0].Desc, 4)
	assert.Equal(t, false, res.Ok, string(res.Result))
	// 3. has evaluated
	res = dm.EvaluateDapp(dapps[0].DappID, dapps[0].Desc, 4)
	assert.Equal(t, false, res.Ok, string(res.Result))

	res = dm.EvaluateDapp(dapps[0].DappID, dapps[0].Desc, 4)
	assert.Equal(t, true, res.Ok, string(res.Result))
}

func TestDappManager_Query(t *testing.T) {
	dm, mockStub, dapps, dappsData := dappPrepare(t)
	mockStub.EXPECT().Caller().Return(ownerAddr).AnyTimes()

	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).Return(false).Times(1)
	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	// 1. get error
	res := dm.GetDapp(dapps[0].DappID)
	assert.Equal(t, false, res.Ok)
	// 2. ok
	res = dm.GetDapp(dapps[0].DappID)
	assert.Equal(t, true, res.Ok)

	mockStub.EXPECT().Query(DappPrefix).Return(true, dappsData).AnyTimes()

	res = dm.GetAllDapps()
	assert.Equal(t, true, res.Ok)
	theDapps := []*Dapp{}
	err := json.Unmarshal(res.Result, &theDapps)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(theDapps))

	res = dm.GetPermissionDapps(ownerAddr)
	assert.Equal(t, true, res.Ok)
	err = json.Unmarshal(res.Result, &theDapps)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(theDapps))

	res = dm.GetPermissionAvailableDapps(ownerAddr)
	assert.Equal(t, true, res.Ok)
	err = json.Unmarshal(res.Result, &theDapps)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(theDapps))

	mockStub.EXPECT().GetObject(OwnerKey(ownerAddr), gomock.Any()).SetArg(1, map[string]struct{}{
		dapps[0].DappID: struct{}{},
	}).Return(true).Times(1)
	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	res = dm.GetDappsByOwner(ownerAddr)
	assert.Equal(t, true, res.Ok)
	err = json.Unmarshal(res.Result, &theDapps)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(theDapps))

	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	mockStub.EXPECT().GetObject(DappKey(dapps[0].DappID), gomock.Any()).SetArg(1, *dapps[1]).Return(true).Times(1)
	res = dm.IsAvailable(dapps[0].DappID)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "true", string(res.Result))
	res = dm.IsAvailable(dapps[0].DappID)
	assert.Equal(t, true, res.Ok)
	assert.Equal(t, "false", string(res.Result))
}

func dappPrepare(t *testing.T) (*DappManager, *mock_stub.MockStub, []*Dapp, [][]byte) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	dm := &DappManager{
		Stub: mockStub,
	}

	var dapps []*Dapp
	var dappsData [][]byte
	statusType := []governance.GovernanceStatus{
		governance.GovernanceAvailable,
		governance.GovernanceFrozen,
		governance.GovernanceUnavailable,
		governance.GovernanceRegisting,
		governance.GovernanceUpdating,
		governance.GovernanceTransferring,
	}

	for i := 0; i < 6; i++ {
		dappID1 := fmt.Sprintf("%s%d", dappID[:len(dappID)-2], i)
		dapp := &Dapp{
			DappID: dappID1,
			Name:   fmt.Sprintf("name%d", i),
			Type:   "tool",
			Desc:   "desc",
			Url:    "url",
			ContractAddr: map[string]struct{}{
				conAddr1: struct{}{},
			},
			Permission: make(map[string]struct{}),
			OwnerAddr:  ownerAddr,
			Status:     statusType[i],
			Score:      1,
			TransferRecords: []*TransferRecord{
				{
					From:       ownerAddr1,
					To:         ownerAddr,
					Reason:     reason,
					Confirm:    false,
					CreateTime: 0,
				},
			},
			EvaluationRecords: map[string]*governance.EvaluationRecord{
				ownerAddr: &governance.EvaluationRecord{
					Addr:       ownerAddr,
					Score:      1,
					Desc:       "",
					CreateTime: 0,
				},
			},
		}

		if dapp.Status == governance.GovernanceTransferring {
			dapp.TransferRecords = append(dapp.TransferRecords, &TransferRecord{
				From:       ownerAddr,
				To:         ownerAddr1,
				Reason:     reason,
				Confirm:    false,
				CreateTime: 0,
			})
		}

		data, err := json.Marshal(dapp)
		assert.Nil(t, err)

		dapps = append(dapps, dapp)
		dappsData = append(dappsData, data)
	}

	return dm, mockStub, dapps, dappsData
}
