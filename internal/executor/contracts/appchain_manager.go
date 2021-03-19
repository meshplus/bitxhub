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
	am.AppchainManager.Persister = am.Stub
	chain := &appchainMgr.Appchain{}
	if err := json.Unmarshal(extra, chain); err != nil {
		return boltvm.Error("unmarshal json error:" + err.Error())
	}

	ok, err := am.AppchainManager.ChangeStatus(chain.ID, proposalResult)
	if !ok {
		return boltvm.Error(string(err))
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
func (am *AppchainManager) Register(validators string, consensusType int32, chainType, name, desc, version, pubkey string) *boltvm.Response {
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

// UpdateAppchain updates approved appchain
func (am *AppchainManager) UpdateAppchain(validators string, consensusType int32, chainType, name, desc, version, pubkey string) *boltvm.Response {
	am.AppchainManager.Persister = am.Stub
	return responseWrapper(am.AppchainManager.UpdateAppchain(am.Caller(), validators, consensusType, chainType, name, desc, version, pubkey))
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
