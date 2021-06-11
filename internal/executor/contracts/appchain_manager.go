package contracts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/hexutil"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/repo"
)

// todo: get this from config file
const relayRootPrefix = "did:bitxhub:relayroot:"

type AppchainManager struct {
	boltvm.Stub
	appchainMgr.AppchainManager
}

// extra: appchainMgr.Appchain
func (am *AppchainManager) Manage(eventTyp string, proposalResult, lastStatus string, extra []byte) *boltvm.Response {
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

	ok, errData := am.AppchainManager.ChangeStatus(chain.ID, proposalResult, lastStatus, nil)
	if !ok {
		return boltvm.Error(string(errData))
	}

	if proposalResult == string(APPOVED) {
		//relaychainAdmin := relayRootPrefix + am.Caller()
		switch eventTyp {
		case string(governance.EventRegister):
			// When applying a new method for appchain is successful
			// 1. notify InterchainContract
			// 2. notify MethodRegistryContract to auditApply this method, then register appchain info
			res = am.CrossInvoke(constant.InterchainContractAddr.String(), "Register", pb.String(chain.ID))
			if !res.Ok {
				return res
			}
			if err = am.chainDefaultConfig(chain); err != nil {
				return boltvm.Error("chain default config error:" + err.Error())
			}
			//res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "AuditApply",
			//	pb.String(relaychainAdmin), pb.String(chain.ID), pb.Int32(1), pb.Bytes(nil))
			//if !res.Ok {
			//	return res
			//}
			//return am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Register",
			//	pb.String(relaychainAdmin), pb.String(chain.ID),
			//	pb.String(chain.DidDocAddr), pb.Bytes([]byte(chain.DidDocHash)), pb.Bytes(nil))
		case string(governance.EventUpdate):
			res := responseWrapper(am.AppchainManager.Update(extra))
			if !res.Ok {
				return res
			}
		}
	} else {
		//relaychainAdmin := relayRootPrefix + am.Caller()
		//switch eventTyp {
		//case string(governance.EventRegister):
		//	res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Audit",
		//		pb.String(relaychainAdmin), pb.String(chain.ID), pb.String(string(bitxid.Initial)), pb.Bytes(nil))
		//	if !res.Ok {
		//		return res
		//	}
		//
		//}
	}

	return boltvm.Success(nil)
}

func (am *AppchainManager) chainDefaultConfig(chain *appchainMgr.Appchain) error {
	if chain.ChainType == appchainMgr.FabricType {
		res := am.CrossInvoke(constant.RuleManagerContractAddr.String(), "DefaultRule", pb.String(chain.ID), pb.String(validator.FabricRuleAddr))
		if !res.Ok {
			return fmt.Errorf(string(res.Result))
		}
		res = am.CrossInvoke(constant.RuleManagerContractAddr.String(), "DefaultRule", pb.String(chain.ID), pb.String(validator.SimFabricRuleAddr))
		if !res.Ok {
			return fmt.Errorf(string(res.Result))
		}
	}
	return nil
}

// Register registers appchain info
// caller is the appchain manager address
// return appchain id, proposal id and error
func (am *AppchainManager) Register(method string, docAddr, docHash, validators string,
	consensusType, chainType, name, desc, version, pubkey string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	//res := am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Apply",
	//	pb.String(appchainAdminDID), pb.String(appchainMethod), pb.Bytes(nil))
	//if !res.Ok {
	//	return res
	//}

	addr, err := getAddr(pubkey)
	if err != nil {
		return boltvm.Error(fmt.Sprintf("get addr from public key: %v", err))
	}

	appchainAdminDID := fmt.Sprintf("%s:%s:%s", repo.BitxhubRootPrefix, method, addr)
	appchainDID := fmt.Sprintf("%s:%s:.", repo.BitxhubRootPrefix, method)

	chain := &appchainMgr.Appchain{
		ID:            appchainDID,
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		ChainType:     chainType,
		Status:        governance.GovernanceRegisting,
		Desc:          desc,
		Version:       version,
		PublicKey:     pubkey,
		DidDocAddr:    docAddr,
		DidDocHash:    docHash,
		OwnerDID:      appchainAdminDID,
	}
	chainData, err := json.Marshal(chain)
	if err != nil {
		return boltvm.Error("marshal chain error:" + err.Error())
	}

	ok, data := am.AppchainManager.Register(chainData)
	if !ok {
		return boltvm.Error("register error: " + string(data))
	}

	registerRes := &governance.RegisterResult{}
	if err := json.Unmarshal(data, registerRes); err != nil {
		return boltvm.Error("register error: " + string(data))
	}
	if registerRes.IsRegistered {
		return boltvm.Error("appchain has registered, chain id: " + registerRes.ID)
	}

	res := am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventRegister)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(appchainDID),
		pb.String(string(governance.GovernanceUnavailable)),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return res
	}

	return getGovernanceRet(string(res.Result), []byte(appchainDID))
}

// UpdateAppchain updates available appchain
func (am *AppchainManager) UpdateAppchain(id, docAddr, docHash, validators string, consensusType, chainType,
	name, desc, version, pubkey string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, data := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(data))
	}

	oldChainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(data, oldChainInfo); err != nil {
		return boltvm.Error(err.Error())
	}

	addr, err := getAddr(oldChainInfo.PublicKey)
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

	// pre update
	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventUpdate, nil); !ok {
		return boltvm.Error("update prepare error: " + string(data))
	}

	chain := &appchainMgr.Appchain{
		ID:            id,
		Name:          name,
		Validators:    validators,
		ConsensusType: consensusType,
		Status:        governance.GovernanceAvailable,
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

	if oldChainInfo.PublicKey != chain.PublicKey {
		return boltvm.Error("pubkey can not be updated")
	}

	if oldChainInfo.Validators == chain.Validators {
		res := responseWrapper(am.AppchainManager.Update(data))
		if !res.Ok {
			return res
		} else {
			return getGovernanceRet("", nil)
		}
	}

	res = am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventUpdate)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(oldChainInfo.Status)),
		pb.Bytes(data),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventUpdate), string(chain.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// FreezeAppchain freezes available appchain
func (am *AppchainManager) FreezeAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	ok, chainData := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(chainData))
	}
	chainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(chainData, chainInfo); err != nil {
		return boltvm.Error(err.Error())
	}

	addr, err := getAddr(chainInfo.PublicKey)
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

	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventFreeze, nil); !ok {
		return boltvm.Error("freeze prepare error: " + string(data))
	}

	res = am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventFreeze)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventFreeze), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// ActivateAppchain updates freezing appchain
func (am *AppchainManager) ActivateAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, chainData := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(chainData))
	}
	chainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(chainData, chainInfo); err != nil {
		return boltvm.Error(err.Error())
	}
	addr, err := getAddr(chainInfo.PublicKey)
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

	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventActivate, nil); !ok {
		return boltvm.Error("activate prepare error: " + string(data))
	}

	am.AppchainManager.Persister = am.Stub

	res = am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventActivate)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventActivate), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// LogoutAppchain updates available appchain
func (am *AppchainManager) LogoutAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	ok, chainData := am.AppchainManager.QueryById(id, nil)
	if !ok {
		return boltvm.Error(string(chainData))
	}
	chainInfo := &appchainMgr.Appchain{}
	if err := json.Unmarshal(chainData, chainInfo); err != nil {
		return boltvm.Error(err.Error())
	}
	addr, err := getAddr(chainInfo.PublicKey)
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

	if ok, data := am.AppchainManager.GovernancePre(id, governance.EventLogout, nil); !ok {
		return boltvm.Error("logout prepare error: " + string(data))
	}

	res = am.CrossInvoke(constant.GovernanceContractAddr.String(), "SubmitProposal",
		pb.String(am.Caller()),
		pb.String(string(governance.EventLogout)),
		pb.String(""),
		pb.String(string(AppchainMgr)),
		pb.String(id),
		pb.String(string(chainInfo.Status)),
		pb.Bytes(chainData),
	)
	if !res.Ok {
		return boltvm.Error("submit proposal error:" + string(res.Result))
	}

	if ok, data := am.AppchainManager.ChangeStatus(id, string(governance.EventLogout), string(chainInfo.Status), nil); !ok {
		return boltvm.Error(string(data))
	}

	return getGovernanceRet(string(res.Result), nil)
}

// CountAvailableAppchains counts all available appchains
func (am *AppchainManager) CountAvailableAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAvailable(nil))
}

// CountAppchains counts all appchains including approved, rejected or registered
func (am *AppchainManager) CountAppchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.CountAll(nil))
}

// Appchains returns all appchains
func (am *AppchainManager) Appchains() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.All(nil))
}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) IsAppchainAdmin() *boltvm.Response {
	am.AppchainManager.Persister = am.Stub

	addr := am.Caller()

	ok, data := am.AppchainManager.All(nil)
	if !ok {
		return boltvm.Error(string(data))
	}
	chains := make([]*appchainMgr.Appchain, 0)
	err := json.Unmarshal(data, &chains)
	if err != nil {
		return boltvm.Error(err.Error())
	}

	for _, chain := range chains {
		tmpAddr, err := getAddr(chain.PublicKey)
		if err != nil {
			return boltvm.Error("get addr error: " + err.Error())
		}
		if tmpAddr == addr {
			return boltvm.Success(nil)
		}
	}

	return boltvm.Error("not found the appchain admin")
}

// GetAppchain returns appchain info by appchain id
func (am *AppchainManager) GetAppchain(id string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.QueryById(id, nil))
}

func (am *AppchainManager) GetIdByAddr(addr string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.GetIdByAddr(addr))
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
	//relayAdminDID := relayRootPrefix + am.Caller()
	//res = am.CrossInvoke(constant.MethodRegistryContractAddr.String(), "Delete", pb.String(relayAdminDID), pb.String(toDeleteMethod), nil)
	//if !res.Ok {
	//	return res
	//}
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
	is, data := am.AppchainManager.QueryById(chainId, nil)

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
	var pubKeyBytes []byte
	var pubKey crypto.PublicKey
	pubKeyBytes = hexutil.Decode(pubKeyStr)
	pubKey, err := ecdsa.UnmarshalPublicKey(pubKeyBytes, crypto.Secp256k1)
	if err != nil {
		pubKeyBytes, err = base64.StdEncoding.DecodeString(pubKeyStr)
		if err != nil {
			return "", fmt.Errorf("decode error: %w", err)
		}
		pubKey, err = ecdsa.UnmarshalPublicKey(pubKeyBytes, crypto.Secp256k1)
		if err != nil {
			return "", fmt.Errorf("decrypt registerd public key error: %w", err)
		}
		//return "", fmt.Errorf("decrypt registerd public key error: %w", err)
	}
	addr, err := pubKey.Address()
	if err != nil {
		return "", fmt.Errorf("decrypt registerd public key error: %w", err)
	}

	return addr.String(), nil
}
