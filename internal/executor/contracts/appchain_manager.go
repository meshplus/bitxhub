package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
)

type AppchainManager struct {
	boltvm.Stub
	appchainMgr.AppchainManager
}

type RegisterResult struct {
	ChainID    string `json:"chain_id"`
	ProposalID string `json:"proposal_id"`
}

func (am *AppchainManager) Manager(des string, proposalResult string, extra []byte) *boltvm.Response {
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + string(err.Error()))
	}
	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSpecific)),
		pb.String(""),
		pb.String(am.CurrentCaller()),
		pb.Bytes(addrsData))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	am.AppchainManager.Persister = am.Stub
	chain := &appchainMgr.Appchain{}
	if err := json.Unmarshal(extra, chain); err != nil {
		return boltvm.Error("unmarshal json error:" + err.Error())
	}

	ok, errData := am.AppchainManager.ChangeStatus(chain.ID, proposalResult)
	if !ok {
		return boltvm.Error(string(errData))
	}

	if proposalResult == string(APPOVED) {
		switch des {
		case appchainMgr.EventRegister:
			return am.CrossInvoke(constant.InterchainContractAddr.String(), "Register", pb.String(chain.ID))
		case appchainMgr.EventUpdate:
			return responseWrapper(am.AppchainManager.UpdateAppchain(chain.ID, chain.Validators, chain.ConsensusType, chain.ChainType, chain.Name, chain.Desc, chain.Version, chain.PublicKey))
		}
	}

	return boltvm.Success(nil)
}

// Register appchain managers registers appchain info caller is the appchain
// manager address return appchain id and error
func (am *AppchainManager) Register(validators string, consensusType, chainType, name, desc, version, pubkey string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	ok, idData := am.AppchainManager.Register(am.Caller(), validators, consensusType, chainType, name, desc, version, pubkey)
	if ok {
		return boltvm.Error("appchain has registered, chain id: " + string(idData))
	}

	ok, data := am.AppchainManager.GetAppchain(string(idData))
	if !ok {
		return boltvm.Error("get appchain error: " + string(data))
	}

	res := am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(appchainMgr.EventRegister),
		pb.String(string(AppchainMgr)),
		pb.Bytes(data),
	)

	if !res.Ok {
		return res
	}

	res1 := RegisterResult{
		ChainID:    am.Caller(),
		ProposalID: string(res.Result),
	}
	resData, err := json.Marshal(res1)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(resData)
}

// UpdateAppchain updates available appchain
func (am *AppchainManager) UpdateAppchain(validators string, consensusType, chainType, name, desc, version, pubkey string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	if ok, data := am.AppchainManager.ChangeStatus(am.Caller(), appchainMgr.EventUpdate); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID:            am.Caller(),
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		Status:        appchainMgr.AppchainUpdating,
		ChainType:     chainType,
		Desc:          desc,
		Version:       version,
		PublicKey:     pubkey,
	}
	data, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(appchainMgr.EventUpdate),
		pb.String(string(AppchainMgr)),
		pb.Bytes(data),
	)
}

// FreezeAppchain freezes available appchain
func (am *AppchainManager) FreezeAppchain(id string) *boltvm.Response {
	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelfAdmin)),
		pb.String(id),
		pb.String(am.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	am.AppchainManager.Persister = am.Stub
	if ok, data := am.AppchainManager.ChangeStatus(id, appchainMgr.EventFreeze); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID: id,
	}
	data, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(appchainMgr.EventFreeze),
		pb.String(string(AppchainMgr)),
		pb.Bytes(data),
	)
}

// ActivateAppchain updates freezing appchain
func (am *AppchainManager) ActivateAppchain(id string) *boltvm.Response {
	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelfAdmin)),
		pb.String(id),
		pb.String(am.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	am.AppchainManager.Persister = am.Stub
	if ok, data := am.AppchainManager.ChangeStatus(id, appchainMgr.EventActivate); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID: id,
	}
	data, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(appchainMgr.EventActivate),
		pb.String(string(AppchainMgr)),
		pb.Bytes(data),
	)
}

// LogoutAppchain updates available appchain
func (am *AppchainManager) LogoutAppchain() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	if ok, data := am.AppchainManager.ChangeStatus(am.Caller(), appchainMgr.EventLogout); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID: am.Caller(),
	}
	data, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(appchainMgr.EventLogout),
		pb.String(string(AppchainMgr)),
		pb.Bytes(data),
	)
}

// CountApprovedAppchains counts all approved appchains
func (am *AppchainManager) CountAvailableAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAvailableAppchains())
}

// CountAppchains counts all appchains including approved, rejected or registered
func (am *AppchainManager) CountAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAppchains())
}

// Appchains returns all appchains
func (am *AppchainManager) Appchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.Appchains())
}

// Appchain returns appchain info
func (am *AppchainManager) Appchain() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.Appchain())
}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) GetAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.GetAppchain(id))
}

// GetPubKeyByChainID can get aim chain's public key using aim chain ID
func (am *AppchainManager) GetPubKeyByChainID(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.GetPubKeyByChainID(id))
}

func (am *AppchainManager) DeleteAppchain(cid string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	if res := am.IsAdmin(); !res.Ok {
		return res
	}
	res := am.CrossInvoke(constant.InterchainContractAddr.String(), "DeleteInterchain", pb.String(cid))
	if !res.Ok {
		return res
	}
	return responseWrapper(am.AppchainManager.DeleteAppchain(cid))
}

func (am *AppchainManager) IsAdmin() *boltvm.Response {
	ret := am.CrossInvoke(constant.RoleContractAddr.String(), "IsAdmin", pb.String(am.Caller()))
	is, err := strconv.ParseBool(string(ret.Result))
	if err != nil {
		return boltvm.Error(fmt.Errorf("judge caller type: %w", err).Error())
	}

	if !is {
		return boltvm.Error("caller is not an admin account")
	}
	return boltvm.Success([]byte("1"))
}

func responseWrapper(ok bool, data []byte) *boltvm.Response {
	if ok {
		return boltvm.Success(data)
	}
	return boltvm.Error(string(data))
}

func (am *AppchainManager) IsAvailable(chainId string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	is, data := am.AppchainManager.GetAppchain(chainId)

	if !is {
		return boltvm.Error("get appchain info error: " + string(data))
	}

	app := &appchainMgr.Appchain{}
	if err := json.Unmarshal(data, app); err != nil {
		return boltvm.Error("unmarshal error: " + err.Error())
	}

	if app.Status != appchainMgr.AppchainAvailable {
		return boltvm.Error("the appchain status is " + string(app.Status))
	}

	return boltvm.Success(nil)
}
