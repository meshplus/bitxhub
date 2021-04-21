package contracts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/tidwall/gjson"
)

// todo: get this from config file
const relayRootPrefix = "did:bitxhub:relayroot:"

type AppchainManager struct {
	boltvm.Stub
	appchainMgr.AppchainManager
}

type RegisterResult struct {
	ChainID    string `json:"chain_id"`
	ProposalID string `json:"proposal_id"`
}

func (am *AppchainManager) Manage(eventTyp string, proposalResult string, extra []byte) *boltvm.Response {
	specificAddrs := []string{constant.GovernanceContractAddr.Address().String()}
	addrsData, err := json.Marshal(specificAddrs)
	if err != nil {
		return boltvm.Error("marshal specificAddrs error:" + err.Error())
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
		switch eventTyp {
		case string(governance.EventRegister):
			// When applying a new method for appchain is successful
			// 1. notify InterchainContract
			// 2. notify MethodRegistryContract to auditApply this method, then register appchain info
			res = am.CrossInvoke(constant.InterchainContractAddr.String(), "Register", pb.String(chain.ID))
			if !res.Ok {
				return res
			}

			relaychainAdmin := relayRootPrefix + am.Caller()
			res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "AuditApply",
				pb.String(relaychainAdmin), pb.String(chain.ID), pb.Int32(1), pb.Bytes(nil))
			if !res.Ok {
				return res
			}

			return am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Register",
				pb.String(relaychainAdmin), pb.String(chain.ID),
				pb.String(chain.DidDocAddr), pb.Bytes([]byte(chain.DidDocHash)), pb.Bytes(nil))
		case string(governance.EventUpdate):
			return responseWrapper(am.AppchainManager.UpdateAppchain(chain.ID, chain.OwnerDID,
				chain.DidDocAddr, chain.DidDocHash, chain.Validators, chain.ConsensusType,
				chain.ChainType, chain.Name, chain.Desc, chain.Version, chain.PublicKey))
		}
	}

	return boltvm.Success(nil)
}

// Register appchain managers registers appchain info caller is the appchain
// manager address return appchain id and error
func (am *AppchainManager) Register(appchainAdminDID, appchainMethod string, docAddr, docHash, validators string,
	consensusType, chainType, name, desc, version, pubkey string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	res := am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Apply",
		pb.String(appchainAdminDID), pb.String(appchainMethod), pb.Bytes(nil))
	if !res.Ok {
		return res
	}
	ok, idData := am.AppchainManager.Register(appchainMethod, appchainAdminDID, docAddr, docHash, validators, consensusType,
		chainType, name, desc, version, pubkey)
	if ok {
		return boltvm.Error("appchain has registered, chain id: " + string(idData))
	}

	ok, data := am.AppchainManager.GetAppchain(string(idData))
	if !ok {
		return boltvm.Error("get appchain error: " + string(data))
	}

	res = am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventRegister)),
		pb.String("des"),
		pb.String(string(AppchainMgr)),
		pb.String(appchainMethod),
		pb.Bytes(data),
	)
	if !res.Ok {
		return res
	}
	res1 := RegisterResult{
		ChainID:    appchainMethod,
		ProposalID: string(res.Result),
	}
	resData, err := json.Marshal(res1)
	if err != nil {
		return boltvm.Error(err.Error())
	}
	return boltvm.Success(resData)
}

// UpdateAppchain updates available appchain
func (am *AppchainManager) UpdateAppchain(id, docAddr, docHash, validators string, consensusType, chainType,
	name, desc, version, pubkey string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, data := am.AppchainManager.GetAppchain(id)
	if !ok {
		return boltvm.Error(string(data))
	}
	pubKeyStr := gjson.Get(string(data), "public_key").String()
	addr, err := getAddr(pubKeyStr)
	if err != nil {
		return boltvm.Error("get addr error: " + err.Error())
	}

	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelf)),
		pb.String(addr),
		pb.String(am.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventUpdate)); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID:            id,
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		Status:        governance.GovernanceUpdating,
		ChainType:     chainType,
		Desc:          desc,
		Version:       version,
		PublicKey:     pubkey,
		DidDocAddr:    docAddr,
		DidDocHash:    docHash,
	}
	data, err = json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventUpdate)),
		pb.String("des"),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.Bytes(data),
	)
}

// FreezeAppchain freezes available appchain
func (am *AppchainManager) FreezeAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	ok, data := am.AppchainManager.GetAppchain(id)
	if !ok {
		return boltvm.Error(string(data))
	}

	pubKeyStr := gjson.Get(string(data), "public_key").String()
	addr, err := getAddr(pubKeyStr)
	if err != nil {
		return boltvm.Error("get addr error: " + err.Error())
	}

	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelfAdmin)),
		pb.String(addr),
		pb.String(am.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventFreeze)); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID: id,
	}
	chainData, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventFreeze)),
		pb.String("des"),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.Bytes(chainData),
	)
}

// ActivateAppchain updates freezing appchain
func (am *AppchainManager) ActivateAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, data := am.AppchainManager.GetAppchain(id)
	if !ok {
		return boltvm.Error(string(data))
	}
	pubKeyStr := gjson.Get(string(data), "public_key").String()
	addr, err := getAddr(pubKeyStr)
	if err != nil {
		return boltvm.Error("get addr error: " + err.Error())
	}

	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelfAdmin)),
		pb.String(addr),
		pb.String(am.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	am.AppchainManager.Persister = am.Stub
	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventActivate)); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID: id,
	}
	data, err = json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventActivate)),
		pb.String("des"),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.Bytes(data),
	)
}

// LogoutAppchain updates available appchain
func (am *AppchainManager) LogoutAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, data := am.AppchainManager.GetAppchain(id)
	if !ok {
		return boltvm.Error(string(data))
	}
	pubKeyStr := gjson.Get(string(data), "public_key").String()
	addr, err := getAddr(pubKeyStr)
	if err != nil {
		return boltvm.Error("get addr error: " + err.Error())
	}

	res := am.CrossInvoke(constant.RoleContractAddr.String(), "CheckPermission",
		pb.String(string(PermissionSelf)),
		pb.String(addr),
		pb.String(am.CurrentCaller()),
		pb.Bytes(nil))
	if !res.Ok {
		return boltvm.Error("check permission error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventLogout)); !ok {
		return boltvm.Error(string(data))
	}

	chain := &appchainMgr.Appchain{
		ID: id,
	}
	data, err = json.Marshal(chain)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	return am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventLogout)),
		pb.String("des"),
		pb.String(string(AppchainMgr)),
		pb.String(id),
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

func (am *AppchainManager) DeleteAppchain(toDeleteMethod string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	if res := am.IsAdmin(); !res.Ok {
		return res
	}
	res := am.CrossInvoke(constant.InterchainContractAddr.String(), "DeleteInterchain", pb.String(toDeleteMethod))
	if !res.Ok {
		return res
	}
	relayAdminDID := relayRootPrefix + am.Caller()
	res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Delete", pb.String(relayAdminDID), pb.String(toDeleteMethod), nil)
	if !res.Ok {
		return res
	}
	return responseWrapper(am.AppchainManager.DeleteAppchain(toDeleteMethod))
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

	if app.Status != governance.GovernanceAvailable {
		return boltvm.Error("the appchain status is " + string(app.Status))
	}

	return boltvm.Success(nil)
}

func getAddr(pubKeyStr string) (string, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyStr)
	if err != nil {
		return "", fmt.Errorf("decode error: %w", err)
	}
	pubKey, err := ecdsa.UnmarshalPublicKey(pubKeyBytes, crypto.Secp256k1)
	if err != nil {
		return "", fmt.Errorf("decrypt registerd public key error: %w", err)
	}
	addr, err := pubKey.Address()
	if err != nil {
		return "", fmt.Errorf("decrypt registerd public key error: %w", err)
	}

	return addr.String(), nil
}
