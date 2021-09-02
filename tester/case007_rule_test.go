package tester

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"

	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/stretchr/testify/suite"
)

type Rule struct {
	suite.Suite
	api api.CoreAPI
}

func (suite *Rule) SetupSuite() {
}

func (suite *Rule) TestRegisterRuleAppchain() {
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
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	suite.Require().Nil(err)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(addr1.String())
	k2Nonce := suite.api.Broker().GetPendingNonceByAccount(addr2.String())

	chainID1 := fmt.Sprintf("appchain%s", addr1.String())
	//chainID2 := fmt.Sprintf("appchain%s", addr2.String())

	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)

	// k1 register rule for chain1 , k1 will be admin of chain1
	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()),
		pb.String("url"),
	)
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	// k2 register rule for chain1, does not have permission
	ruleAddr2, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr2.String()),
		pb.String("url"),
	)
	suite.Assert().Nil(err)
	suite.Require().False(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Contains(string(ret.Ret), "permission")
	fmt.Printf("k2 register rule for chain1 error: %s\n", string(ret.Ret))
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetRuleByAddr",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RoleContractAddr.Address(), "GetRole")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Equal(string(contracts.AppchainAdmin), string(ret.Ret))
	k1Nonce++

	// register appchain
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain",
		pb.String(chainID1),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(ruleAddr1.String()),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId1 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	suite.vote(proposalId1, priAdmin1, adminNonce1)
	adminNonce1++

	suite.vote(proposalId1, priAdmin2, adminNonce2)
	adminNonce2++

	suite.vote(proposalId1, priAdmin3, adminNonce3)
	adminNonce3++

	// k2 register rule for chain1, does not have permission
	ruleAddr3, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr3.String()),
		pb.String("url"),
	)
	suite.Assert().Nil(err)
	suite.Require().False(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Contains(string(ret.Ret), "permission")
	fmt.Printf("k2 register rule for chain1 error: %s\n", string(ret.Ret))
	k2Nonce++
}

func (suite *Rule) TestRegisterAppchainRule() {
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
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	suite.Require().Nil(err)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(addr1.String())
	k2Nonce := suite.api.Broker().GetPendingNonceByAccount(addr2.String())

	chainID1 := fmt.Sprintf("appchain%s", addr1.String())
	//chainID2 := fmt.Sprintf("appchain%s", addr2.String())

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++

	// k1 register chain1, k1 will be admin of chain1
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain",
		pb.String(chainID1),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(validator.FabricRuleAddr),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId1 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RoleContractAddr.Address(), "GetRole")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Equal(string(contracts.AppchainAdmin), string(ret.Ret))
	k1Nonce++

	suite.vote(proposalId1, priAdmin1, adminNonce1)
	adminNonce1++

	suite.vote(proposalId1, priAdmin2, adminNonce2)
	adminNonce2++

	suite.vote(proposalId1, priAdmin3, adminNonce3)
	adminNonce3++

	// k1 register rule for chain1
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()),
		pb.String("url"),
	)
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	// k2 register rule for chain1, does not have permission
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()),
		pb.String("url"),
	)
	suite.Assert().Nil(err)
	suite.Require().False(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Contains(string(ret.Ret), "permission")
	fmt.Printf("k2 register rule for chain1 error: %s\n", string(ret.Ret))
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetRuleByAddr",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
}

func (suite *Rule) TestUpdateMasterRule() {
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
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	suite.Require().Nil(err)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(addr1.String())

	chainID1 := fmt.Sprintf("appchain%s", addr1.String())
	//chainID2 := fmt.Sprintf("appchain%s", addr2.String())

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)

	// k1 register rule for chain1 , k1 will be admin of chain1
	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()),
		pb.String("url"),
	)
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetRuleByAddr",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RoleContractAddr.Address(), "GetRole")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Equal(string(contracts.AppchainAdmin), string(ret.Ret))
	k1Nonce++

	// register appchain
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain",
		pb.String(chainID1),
		pb.Bytes(nil),
		pb.String("broker"),
		pb.String("desc"),
		pb.String(ruleAddr1.String()),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId1 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	suite.vote(proposalId1, priAdmin1, adminNonce1)
	adminNonce1++

	suite.vote(proposalId1, priAdmin2, adminNonce2)
	adminNonce2++

	suite.vote(proposalId1, priAdmin3, adminNonce3)
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetMasterRule",
		pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	rule := &ruleMgr.Rule{}
	err = json.Unmarshal(ret.Ret, rule)
	suite.Require().Nil(err)
	suite.Require().Equal(ruleAddr1.String(), string(rule.Address))

	// k1 register rule2 for chain1
	ruleAddr2, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule",
		pb.String(chainID1),
		pb.String(ruleAddr2.String()),
		pb.String("url"),
	)
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	// update master rule
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "UpdateMasterRule",
		pb.String(chainID1),
		pb.String(ruleAddr2.String()),
		pb.String("reason"))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId2 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	appchain := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceFrozen, appchain.Status)

	suite.vote(proposalId2, priAdmin1, adminNonce1)
	adminNonce1++

	suite.vote(proposalId2, priAdmin2, adminNonce2)
	adminNonce2++

	suite.vote(proposalId2, priAdmin3, adminNonce3)
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetMasterRule",
		pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, rule)
	suite.Require().Nil(err)
	suite.Require().Equal(ruleAddr2.String(), rule.Address)
}

func (suite *Rule) vote(proposalId string, adminKey crypto.PrivateKey, adminNonce uint64) {
	ret, err := invokeBVMContract(suite.api, adminKey, adminNonce, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}
