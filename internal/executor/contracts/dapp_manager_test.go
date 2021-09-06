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
	"github.com/stretchr/testify/assert"
)

const (
	ownerAddr  = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f1"
	ownerAddr1 = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f2"
	dappID     = "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f1-0"
)

func TestDappManager_RegisterDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

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
	res := dm.RegisterDapp(dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "-", reason)
	assert.Equal(t, false, res.Ok)
	// 2. governancePre error
	res = dm.RegisterDapp(dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, false, res.Ok)
	// 3. submit error
	res = dm.RegisterDapp(dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, false, res.Ok)

	res = dm.RegisterDapp(dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, true, res.Ok)
}

func TestDappManager_UpdateDapp(t *testing.T) {
	dm, mockStub, dapps, _ := dappPrepare(t)

	governancePreErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(false).Times(1)
	checkPermissionErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	checkInfoErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	submitErrReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	changeStatusErrReq1 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(1)
	changeStatusErrReq2 := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[2]).Return(true).Times(1)
	okReq := mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).SetArg(1, *dapps[0]).Return(true).Times(2)
	gomock.InOrder(governancePreErrReq, checkPermissionErrReq, checkInfoErrReq, submitErrReq, changeStatusErrReq1, changeStatusErrReq2, okReq)

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
	res := dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, false, res.Ok)
	// 2. check permision error
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, false, res.Ok)
	// 3. check info error
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "-", reason)
	assert.Equal(t, false, res.Ok)
	// 4. submit error
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, false, res.Ok)
	// 5. change status error
	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, false, res.Ok)

	res = dm.UpdateDapp(dapps[0].DappID, dapps[0].Name, dapps[0].Type, dapps[0].Desc, "", "", reason)
	assert.Equal(t, true, res.Ok)
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
	res := dm.TransferDapp(dapps[0].DappID, reason, "1")
	assert.Equal(t, false, res.Ok)
	// 2. governancePre error
	res = dm.TransferDapp(dapps[0].DappID, reason, ownerAddr1)
	assert.Equal(t, false, res.Ok)
	// 3. check permision error
	res = dm.TransferDapp(dapps[0].DappID, reason, ownerAddr1)
	assert.Equal(t, false, res.Ok)
	// 4. submit error
	res = dm.TransferDapp(dapps[0].DappID, reason, ownerAddr1)
	assert.Equal(t, false, res.Ok)
	// 5. change status error
	res = dm.TransferDapp(dapps[0].DappID, reason, ownerAddr1)
	assert.Equal(t, false, res.Ok)

	res = dm.TransferDapp(dapps[0].DappID, reason, ownerAddr1)
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

	mockStub.EXPECT().Query(DAPPPREFIX).Return(true, dappsData).AnyTimes()

	res = dm.GetAllDapps()
	assert.Equal(t, true, res.Ok)
	theDapps := []*Dapp{}
	err := json.Unmarshal(res.Result, &theDapps)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(theDapps))

	res = dm.GetPermissionDapps()
	assert.Equal(t, true, res.Ok)
	err = json.Unmarshal(res.Result, &theDapps)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(theDapps))

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
	statusType := []governance.GovernanceStatus{governance.GovernanceAvailable, governance.GovernanceFrozen, governance.GovernanceUnavailable}

	for i := 0; i < 3; i++ {
		dapp := &Dapp{
			DappID:       dappID,
			Name:         "name",
			Type:         "type",
			Desc:         "desc",
			ContractAddr: nil,
			Permission:   nil,
			OwnerAddr:    ownerAddr,
			Status:       statusType[i],
			Score:        1,
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

		data, err := json.Marshal(dapp)
		assert.Nil(t, err)

		dapps = append(dapps, dapp)
		dappsData = append(dappsData, data)
	}

	return dm, mockStub, dapps, dappsData
}
