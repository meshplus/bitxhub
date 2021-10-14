package tester

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
	ruleMgr "github.com/meshplus/bitxhub-core/rule-mgr"
	service_mgr "github.com/meshplus/bitxhub-core/service-mgr"
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

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++

	fabricBroker := appchainMgr.FabricBroker{
		ChannelID:     "1",
		ChaincodeID:   "2",
		BrokerVersion: "3",
	}
	fabricBrokerData, err := json.Marshal(fabricBroker)
	suite.Require().Nil(err)
	chainName1 := "应用链2case007"
	args := []*pb.Arg{
		pb.String(chainName1),
		pb.String(appchainMgr.ChainTypeFabric1_4_3),
		pb.Bytes(nil),
		pb.Bytes(fabricBrokerData),
		pb.String("desc"),
		pb.String(validator.FabricRuleAddr),
		pb.String("url"),
		pb.String(addr1.String()),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain",
		args...,
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId1 := gRet.ProposalID

	suite.vote(proposalId1, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalId1, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalId1, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchainByName", pb.String(chainName1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	chainInfo := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, chainInfo)
	suite.Require().Nil(err)
	suite.Require().Equal("desc", chainInfo.Desc)
	suite.Require().Equal(governance.GovernanceAvailable, chainInfo.Status)
	chainID1 := chainInfo.ID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RoleContractAddr.Address(), "GetRole")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Equal(string(contracts.AppchainAdmin), string(ret.Ret))
	k1Nonce++

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

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++

	// register appchain
	chainName1 := "应用链1case007"
	args := []*pb.Arg{
		pb.String(chainName1),
		pb.String(appchainMgr.ChainTypeHyperchain1_8_6),
		pb.Bytes(nil),
		pb.Bytes([]byte("123")),
		pb.String("desc"),
		pb.String(ruleAddr1.String()),
		pb.String("url"),
		pb.String(addr1.String()),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain",
		args...,
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId1 := gRet.ProposalID

	suite.vote(proposalId1, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalId1, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalId1, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchainByName", pb.String(chainName1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	chainInfo := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, chainInfo)
	suite.Require().Nil(err)
	suite.Require().Equal("desc", chainInfo.Desc)
	suite.Require().Equal(governance.GovernanceAvailable, chainInfo.Status)
	chainID1 := chainInfo.ID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
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

	serviceID1 := "service1"
	chainServiceID1 := fmt.Sprintf("%s:%s", chainID1, serviceID1)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "RegisterService",
		pb.String(chainID1),
		pb.String(serviceID1),
		pb.String("服务1case007"),
		pb.String(string(service_mgr.ServiceCallContract)),
		pb.String("intro"),
		pb.Bool(true),
		pb.String(""),
		pb.String("details"),
		pb.String("raeson"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet = &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalServiceId1 := gRet.ProposalID

	suite.vote(proposalServiceId1, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalServiceId1, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalServiceId1, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	service := &service_mgr.Service{}
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, service.Status)

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

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernancePause, service.Status)

	suite.vote(proposalId2, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalId2, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalId2, priAdmin3, adminNonce3, string(contracts.APPROVED))
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

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, service.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "UpdateMasterRule",
		pb.String(chainID1),
		pb.String(ruleAddr1.String()),
		pb.String("reason"))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId3 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceFrozen, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernancePause, service.Status)

	suite.vote(proposalId3, priAdmin1, adminNonce1, string(contracts.REJECTED))
	adminNonce1++

	suite.vote(proposalId3, priAdmin2, adminNonce2, string(contracts.REJECTED))
	adminNonce2++

	suite.vote(proposalId3, priAdmin3, adminNonce3, string(contracts.REJECTED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceFrozen, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetMasterRule",
		pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, rule)
	suite.Require().Nil(err)
	suite.Require().Equal(ruleAddr2.String(), rule.Address)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernancePause, service.Status)
}

func (suite *Rule) vote(proposalId string, adminKey crypto.PrivateKey, adminNonce uint64, info string) {
	ret, err := invokeBVMContract(suite.api, adminKey, adminNonce, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId),
		pb.String(info),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}
