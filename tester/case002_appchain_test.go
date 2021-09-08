package tester

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
	rule_mgr "github.com/meshplus/bitxhub-core/rule-mgr"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

const (
	appchainMethod           = "did:bitxhub:appchain1:."
	appchainAdminDIDPrefix   = "did:bitxhub:appchain"
	relaychainAdminDIDPrefix = "did:bitxhub:relayroot"
	relayAdminDID            = "did:bitxhub:relay:0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013"
	docAddr                  = "/ipfs/QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
	docHash                  = "QmQVxzUqN2Yv2UHUQXYwH8dSNkM8ReJ9qPqwJsf8zzoNUi"
)

type RegisterAppchain struct {
	suite.Suite
	api api.CoreAPI
}

func (suite *RegisterAppchain) SetupSuite() {

}

// Appchain registers in bitxhub
func (suite *RegisterAppchain) TestRegisterAppchain() {
	path1 := "./test_data/config/node1/key.json"
	path2 := "./test_data/config/node2/key.json"
	path3 := "./test_data/config/node3/key.json"
	keyPath1 := filepath.Join(path1)
	keyPath2 := filepath.Join(path2)
	keyPath3 := filepath.Join(path3)
	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	suite.Require().Nil(err)
	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	suite.Require().Nil(err)
	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	suite.Require().Nil(err)
	fromAdmin1, err := priAdmin1.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	suite.Require().Nil(err)
	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())

	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)

	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++

	// register rule
	chainID1 := fmt.Sprintf("appchain%s", from1.String())
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()),
		pb.String("ruleUrl"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	// register appchain
	args := []*pb.Arg{
		pb.String(chainID1),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(ruleAddr1.Address),
		pb.String("reason"),
	}
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId := gRet.ProposalID
	fmt.Printf("========%s", proposalId)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	chainInfo := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, chainInfo)
	suite.Require().Nil(err)
	suite.Require().Equal("desc", chainInfo.Desc)
	suite.Require().Equal(governance.GovernanceRegisting, chainInfo.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetRuleByAddr",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	ruleInfo := &rule_mgr.Rule{}
	err = json.Unmarshal(ret.Ret, ruleInfo)
	suite.Require().Nil(err)
	suite.Require().Equal(governance.GovernanceBinding, ruleInfo.Status)

	suite.vote(proposalId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalId, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, chainInfo)
	suite.Require().Nil(err)
	suite.Require().Equal("desc", chainInfo.Desc)
	suite.Require().Equal(governance.GovernanceAvailable, chainInfo.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetRuleByAddr",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, ruleInfo)
	suite.Require().Nil(err)
	suite.Require().Equal(governance.GovernanceAvailable, ruleInfo.Status)
}

func (suite *RegisterAppchain) TestFetchAppchains() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	from2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
	k2Nonce := suite.api.Broker().GetPendingNonceByAccount(from2.String())

	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))

	id1 := fmt.Sprintf("appchain%s", addr1.String())
	args := []*pb.Arg{
		pb.String(id1),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(validator.FabricRuleAddr),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	args = []*pb.Arg{
		pb.String(fmt.Sprintf("appchain%s", addr2.String())),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(validator.SimFabricRuleAddr),
		pb.String("reason"),
	}
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Nil(err)
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Appchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	k2Nonce++

	rec, err := invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "CountAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err := strconv.Atoi(string(rec.Ret))
	suite.Require().Nil(err)
	result := gjson.Parse(string(ret.Ret))
	suite.Require().GreaterOrEqual(num, len(result.Array()))
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "CountAvailableAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err = strconv.Atoi(string(ret.Ret))
	suite.Require().Nil(err)
	suite.Require().EqualValues(0, num)
	k2Nonce++

	//GetAppchain
	ret2, err := invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
	suite.Require().Nil(err)
	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
	appchain := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret2.Ret, appchain)
	suite.Require().Nil(err)
	suite.Require().Equal("desc", appchain.Desc)
	k2Nonce++
}

func (suite *RegisterAppchain) TestUpdateAppchains() {
	path1 := "./test_data/config/node1/key.json"
	path2 := "./test_data/config/node2/key.json"
	path3 := "./test_data/config/node3/key.json"
	keyPath1 := filepath.Join(path1)
	keyPath2 := filepath.Join(path2)
	keyPath3 := filepath.Join(path3)
	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	suite.Require().Nil(err)
	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	suite.Require().Nil(err)
	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	suite.Require().Nil(err)
	fromAdmin1, err := priAdmin1.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	suite.Require().Nil(err)
	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())

	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))

	id1 := fmt.Sprintf("appchain%s", from1.String())
	args := []*pb.Arg{
		pb.String(id1),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(validator.FabricRuleAddr),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	proposalId := gjson.Get(string(ret.Ret), "proposal_id").String()

	suite.vote(proposalId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalId, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	//GetAppchain
	ret2, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
	suite.Require().Nil(err)
	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
	appchain := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret2.Ret, appchain)
	suite.Require().Nil(err)
	suite.Require().Equal(uint64(0), appchain.Version)
	k1Nonce++

	//UpdateAppchain
	args = []*pb.Arg{
		pb.String(id1),
		pb.String("desc1"),
	}
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "UpdateAppchain", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	k1Nonce++

	//GetAppchain
	ret2, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
	suite.Require().Nil(err)
	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
	err = json.Unmarshal(ret2.Ret, appchain)
	suite.Require().Nil(err)
	suite.Require().Equal(uint64(1), appchain.Version)
	k1Nonce++
}

//func (suite *RegisterAppchain) TestActivateAppchains() {
//	path1 := "./test_data/config/node1/key.json"
//	path2 := "./test_data/config/node2/key.json"
//	path3 := "./test_data/config/node3/key.json"
//	keyPath1 := filepath.Join(path1)
//	keyPath2 := filepath.Join(path2)
//	keyPath3 := filepath.Join(path3)
//	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
//	suite.Require().Nil(err)
//	fromAdmin1, err := priAdmin1.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin2, err := priAdmin2.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin3, err := priAdmin3.PublicKey().Address()
//	suite.Require().Nil(err)
//	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
//	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
//	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())
//
//	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
//	suite.Require().Nil(err)
//	from1, err := k1.PublicKey().Address()
//	suite.Require().Nil(err)
//	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
//	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))
//
//	id1 := fmt.Sprintf("appchain%s", from1.String())
//	args := []*pb.Arg{
//		pb.String(id1),
//		pb.Bytes(nil),
//		pb.String("broker"),
//		pb.String("desc"),
//		pb.String(validator.FabricRuleAddr),
//		pb.String("reason"),
//	}
//	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	proposalId := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	suite.vote(proposalId, priAdmin1, adminNonce1, string(contracts.APPROVED))
//	adminNonce1++
//
//	suite.vote(proposalId, priAdmin2, adminNonce2, string(contracts.APPROVED))
//	adminNonce2++
//
//	suite.vote(proposalId, priAdmin3, adminNonce3, string(contracts.APPROVED))
//	adminNonce3++
//
//	//GetAppchain
//	ret2, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
//	suite.Require().Nil(err)
//	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
//	appchain := &appchainMgr.Appchain{}
//	err = json.Unmarshal(ret2.Ret, appchain)
//	suite.Require().Nil(err)
//	suite.Equal(governance.GovernanceAvailable, appchain.Status)
//	k1Nonce++
//
//	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
//	suite.Require().Nil(err)
//
//	// register rule
//	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
//	suite.Require().Nil(err)
//	k1Nonce++
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
//		pb.String(id1),
//		pb.String(ruleAddr1.String()),
//		pb.String("url"),
//	)
//	suite.Assert().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//
//	//UpdateMasterRule
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "UpdateMasterRule", pb.String(id1), pb.String(ruleAddr1.String()), pb.String("reason"))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	proposalId2 := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	//GetAppchain
//	ret2, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
//	suite.Require().Nil(err)
//	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
//	err = json.Unmarshal(ret2.Ret, appchain)
//	suite.Require().Nil(err)
//	suite.Equal(governance.GovernanceFrozen, appchain.Status)
//	k1Nonce++
//
//	// activate appchain
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "ActivateAppchain", pb.String(id1), pb.String("reason"))
//	suite.Require().Nil(err)
//	suite.Require().False(ret.IsSuccess())
//	k1Nonce++
//
//	suite.vote(proposalId2, priAdmin1, adminNonce1, string(contracts.APPROVED))
//	adminNonce1++
//
//	suite.vote(proposalId2, priAdmin2, adminNonce2, string(contracts.APPROVED))
//	adminNonce2++
//
//	suite.vote(proposalId2, priAdmin3, adminNonce3, string(contracts.APPROVED))
//	adminNonce3++
//
//	//GetAppchain
//	ret2, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
//	suite.Require().Nil(err)
//	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
//	err = json.Unmarshal(ret2.Ret, appchain)
//	suite.Require().Nil(err)
//	suite.Equal(governance.GovernanceAvailable, appchain.Status)
//	k1Nonce++
//}
//
//func (suite *RegisterAppchain) TestLogoutAppchain() {
//	path1 := "./test_data/config/node1/key.json"
//	path2 := "./test_data/config/node2/key.json"
//	path3 := "./test_data/config/node3/key.json"
//	keyPath1 := filepath.Join(path1)
//	keyPath2 := filepath.Join(path2)
//	keyPath3 := filepath.Join(path3)
//	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
//	suite.Require().Nil(err)
//	fromAdmin1, err := priAdmin1.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin2, err := priAdmin2.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin3, err := priAdmin3.PublicKey().Address()
//	suite.Require().Nil(err)
//	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
//	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
//	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())
//
//	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
//	suite.Require().Nil(err)
//	from1, err := k1.PublicKey().Address()
//	suite.Require().Nil(err)
//	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
//	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))
//
//	id1 := fmt.Sprintf("appchain%s", from1.String())
//	args := []*pb.Arg{
//		pb.String(id1),
//		pb.Bytes(nil),
//		pb.String("broker"),
//		pb.String("desc"),
//		pb.String(validator.FabricRuleAddr),
//		pb.String("reason"),
//	}
//	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	proposalRegisterAppchainId := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	suite.vote(proposalRegisterAppchainId, priAdmin1, adminNonce1, string(contracts.APPROVED))
//	adminNonce1++
//
//	suite.vote(proposalRegisterAppchainId, priAdmin2, adminNonce2, string(contracts.APPROVED))
//	adminNonce2++
//
//	suite.vote(proposalRegisterAppchainId, priAdmin3, adminNonce3, string(contracts.APPROVED))
//	adminNonce3++
//
//	//GetAppchain
//	ret2, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
//	suite.Require().Nil(err)
//	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
//	appchain := &appchainMgr.Appchain{}
//	err = json.Unmarshal(ret2.Ret, appchain)
//	suite.Require().Nil(err)
//	suite.Equal(governance.GovernanceAvailable, appchain.Status)
//	k1Nonce++
//
//	// logout appchain
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "LogoutAppchain", pb.String(id1), pb.String("reason"))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess())
//	k1Nonce++
//	proposalLogoutId := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	suite.vote(proposalLogoutId, priAdmin1, adminNonce1, string(contracts.REJECTED))
//	adminNonce1++
//
//	suite.vote(proposalLogoutId, priAdmin2, adminNonce2, string(contracts.REJECTED))
//	adminNonce2++
//
//	suite.vote(proposalLogoutId, priAdmin3, adminNonce3, string(contracts.REJECTED))
//	adminNonce3++
//
//	//GetAppchain
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	err = json.Unmarshal(ret.Ret, appchain)
//	suite.Require().Nil(err)
//	suite.Equal(governance.GovernanceAvailable, appchain.Status)
//	k1Nonce++
//
//	// logout appchain
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "LogoutAppchain", pb.String(id1), pb.String("reason"))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess())
//	k1Nonce++
//	proposalLogoutId2 := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	suite.vote(proposalLogoutId2, priAdmin1, adminNonce1, string(contracts.APPROVED))
//	adminNonce1++
//
//	suite.vote(proposalLogoutId2, priAdmin2, adminNonce2, string(contracts.APPROVED))
//	adminNonce2++
//
//	suite.vote(proposalLogoutId2, priAdmin3, adminNonce3, string(contracts.APPROVED))
//	adminNonce3++
//
//	//GetAppchain
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	err = json.Unmarshal(ret.Ret, appchain)
//	suite.Require().Nil(err)
//	suite.Equal(governance.GovernanceForbidden, appchain.Status)
//	k1Nonce++
//}

//func (suite *RegisterAppchain) TestLogoutAppchainsWhenUpdateRule() {
//	path1 := "./test_data/config/node1/key.json"
//	path2 := "./test_data/config/node2/key.json"
//	path3 := "./test_data/config/node3/key.json"
//	keyPath1 := filepath.Join(path1)
//	keyPath2 := filepath.Join(path2)
//	keyPath3 := filepath.Join(path3)
//	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
//	suite.Require().Nil(err)
//	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
//	suite.Require().Nil(err)
//	fromAdmin1, err := priAdmin1.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin2, err := priAdmin2.PublicKey().Address()
//	suite.Require().Nil(err)
//	fromAdmin3, err := priAdmin3.PublicKey().Address()
//	suite.Require().Nil(err)
//	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
//	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
//	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())
//
//	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
//	suite.Require().Nil(err)
//	from1, err := k1.PublicKey().Address()
//	suite.Require().Nil(err)
//	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
//	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))
//
//	id1 := fmt.Sprintf("appchain%s", from1.String())
//	args := []*pb.Arg{
//		pb.String(id1),
//		pb.Bytes(nil),
//		pb.String("broker"),
//		pb.String("desc"),
//		pb.String(validator.FabricRuleAddr),
//		pb.String("reason"),
//	}
//	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain", args...)
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	proposalRegisterAppchainId := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	suite.vote(proposalRegisterAppchainId, priAdmin1, adminNonce1, string(contracts.APPROVED))
//	adminNonce1++
//
//	suite.vote(proposalRegisterAppchainId, priAdmin2, adminNonce2, string(contracts.APPROVED))
//	adminNonce2++
//
//	suite.vote(proposalRegisterAppchainId, priAdmin3, adminNonce3, string(contracts.APPROVED))
//	adminNonce3++
//
//	//GetAppchain
//	ret2, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
//	suite.Require().Nil(err)
//	suite.Require().True(ret2.IsSuccess(), string(ret2.Ret))
//	appchain := &appchainMgr.Appchain{}
//	err = json.Unmarshal(ret2.Ret, appchain)
//	suite.Require().Nil(err)
//	suite.Equal(governance.GovernanceAvailable, appchain.Status)
//	k1Nonce++
//
//	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
//	suite.Require().Nil(err)
//
//	// register rule
//	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
//	suite.Require().Nil(err)
//	k1Nonce++
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
//		pb.String(id1),
//		pb.String(ruleAddr1.String()),
//		pb.String("url"),
//	)
//	suite.Assert().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//
//	//UpdateMasterRule
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "UpdateMasterRule", pb.String(id1), pb.String(ruleAddr1.String()), pb.String("reason"))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	proposalUpdateRuleId := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetRuleByAddr",
//		pb.String(id1),
//		pb.String(ruleAddr1.String()))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	ruleInfo := &rule_mgr.Rule{}
//	err = json.Unmarshal(ret.Ret, ruleInfo)
//	suite.Require().Nil(err)
//	suite.Require().Equal(governance.GovernanceBinding, ruleInfo.Status)
//
//	// logout appchain
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "LogoutAppchain", pb.String(id1), pb.String("reason"))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess())
//	k1Nonce++
//	proposalLogoutId := gjson.Get(string(ret.Ret), "proposal_id").String()
//
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.GovernanceContractAddr.Address(), "GetProposal",
//		pb.String(proposalUpdateRuleId))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	proposalInfo := &contracts.Proposal{}
//	err = json.Unmarshal(ret.Ret, proposalInfo)
//	suite.Require().Nil(err)
//	suite.Require().Equal(contracts.PAUSED, proposalInfo.Status)
//
//	suite.vote(proposalLogoutId, priAdmin1, adminNonce1, string(contracts.REJECTED))
//	adminNonce1++
//
//	suite.vote(proposalLogoutId, priAdmin2, adminNonce2, string(contracts.REJECTED))
//	adminNonce2++
//
//	suite.vote(proposalLogoutId, priAdmin3, adminNonce3, string(contracts.REJECTED))
//	adminNonce3++
//
//	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.GovernanceContractAddr.Address(), "GetProposal",
//		pb.String(proposalUpdateRuleId))
//	suite.Require().Nil(err)
//	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
//	k1Nonce++
//	err = json.Unmarshal(ret.Ret, proposalInfo)
//	suite.Require().Nil(err)
//	suite.Require().Equal(contracts.PROPOSED, proposalInfo.Status)
//}

func (suite *RegisterAppchain) vote(proposalId string, adminKey crypto.PrivateKey, adminNonce uint64, info string) {
	ret, err := invokeBVMContract(suite.api, adminKey, adminNonce, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId),
		pb.String(info),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}
