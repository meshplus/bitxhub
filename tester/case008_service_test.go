package tester

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
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

type Service struct {
	suite.Suite
	api api.CoreAPI
}

func (suite *Service) SetupSuite() {
}

func (suite *Service) TestRegisterService() {
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

	fabricBroker := appchainMgr.FabricBroker{
		ChannelID:     "1",
		ChaincodeID:   "2",
		BrokerVersion: "3",
	}
	fabricBrokerData, err := json.Marshal(fabricBroker)
	suite.Require().Nil(err)
	chainName1 := "应用链1case008"
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
	serviceID1 := "service1"
	chainServiceID1 := fmt.Sprintf("%s:%s", chainID1, serviceID1)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	appchain := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "RegisterService",
		pb.String(chainID1),
		pb.String(serviceID1),
		pb.String("服务1case008"),
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

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "UpdateService",
		pb.String(chainServiceID1),
		pb.String("name1"),
		pb.String("intro"),
		pb.String(""),
		pb.String("details"),
		pb.String("raeson"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	//gRet = &governance.GovernanceResult{}
	//err = json.Unmarshal(ret.Ret, gRet)
	//suite.Require().Nil(err)
	//proposalServiceId2 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceUpdating, service.Status)

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.AppchainMgrContractAddr.Address(), "FreezeAppchain",
		pb.String(chainID1),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalFreezeAppchainId := gRet.ProposalID

	suite.vote(proposalFreezeAppchainId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalFreezeAppchainId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalFreezeAppchainId, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

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

	// activate
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "ActivateAppchain",
		pb.String(chainID1),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalActivateAppchainId := gRet.ProposalID

	suite.vote(proposalActivateAppchainId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalActivateAppchainId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalActivateAppchainId, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceUpdating, service.Status)
}

func (suite *Service) TestLogoutAppchainPauseService() {
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

	fabricBroker := appchainMgr.FabricBroker{
		ChannelID:     "1",
		ChaincodeID:   "2",
		BrokerVersion: "3",
	}
	fabricBrokerData, err := json.Marshal(fabricBroker)
	suite.Require().Nil(err)
	chainName1 := "应用链2case008"
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
	proposalRegisterAppchainId := gRet.ProposalID

	suite.vote(proposalRegisterAppchainId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalRegisterAppchainId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalRegisterAppchainId, priAdmin3, adminNonce3, string(contracts.APPROVED))
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
	serviceID1 := "service1"
	chainServiceID1 := fmt.Sprintf("%s:%s", chainID1, serviceID1)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	appchain := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "RegisterService",
		pb.String(chainID1),
		pb.String(serviceID1),
		pb.String("服务2case008"),
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
	proposalServiceId := gRet.ProposalID

	suite.vote(proposalServiceId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalServiceId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalServiceId, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	service := &service_mgr.Service{}
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, service.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "LogoutAppchain",
		pb.String(chainID1),
		pb.String("raeson"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet = &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalLogoutAppchainID := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernancePause, service.Status)

	suite.vote(proposalLogoutAppchainID, priAdmin1, adminNonce1, string(contracts.REJECTED))
	adminNonce1++

	suite.vote(proposalLogoutAppchainID, priAdmin2, adminNonce2, string(contracts.REJECTED))
	adminNonce2++

	suite.vote(proposalLogoutAppchainID, priAdmin3, adminNonce3, string(contracts.REJECTED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, service.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "LogoutAppchain",
		pb.String(chainID1),
		pb.String("raeson"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet = &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalLogoutAppchainID2 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernancePause, service.Status)

	suite.vote(proposalLogoutAppchainID2, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalLogoutAppchainID2, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalLogoutAppchainID2, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceForbidden, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernancePause, service.Status)

}

func (suite *Service) TestLogoutService() {
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

	fabricBroker := appchainMgr.FabricBroker{
		ChannelID:     "1",
		ChaincodeID:   "2",
		BrokerVersion: "3",
	}
	fabricBrokerData, err := json.Marshal(fabricBroker)
	suite.Require().Nil(err)
	chainName1 := "应用链3case008"
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
	proposalRegisterAppchainId := gRet.ProposalID

	suite.vote(proposalRegisterAppchainId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalRegisterAppchainId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalRegisterAppchainId, priAdmin3, adminNonce3, string(contracts.APPROVED))
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
	serviceID1 := "service1"
	chainServiceID1 := fmt.Sprintf("%s:%s", chainID1, serviceID1)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	appchain := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "RegisterService",
		pb.String(chainID1),
		pb.String(serviceID1),
		pb.String("服务3case008"),
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
	proposalServiceId := gRet.ProposalID

	suite.vote(proposalServiceId, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalServiceId, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalServiceId, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	service := &service_mgr.Service{}
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceAvailable, service.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "LogoutService",
		pb.String(chainServiceID1),
		pb.String("raeson"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet = &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalLogoutServiceID := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "LogoutAppchain",
		pb.String(chainID1),
		pb.String("raeson"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet = &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalLogoutAppchainID := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceLogouting, service.Status)

	suite.vote(proposalLogoutAppchainID, priAdmin1, adminNonce1, string(contracts.APPROVED))
	adminNonce1++

	suite.vote(proposalLogoutAppchainID, priAdmin2, adminNonce2, string(contracts.APPROVED))
	adminNonce2++

	suite.vote(proposalLogoutAppchainID, priAdmin3, adminNonce3, string(contracts.APPROVED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, appchain)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceForbidden, appchain.Status)

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernanceLogouting, service.Status)

	suite.vote(proposalLogoutServiceID, priAdmin1, adminNonce1, string(contracts.REJECTED))
	adminNonce1++

	suite.vote(proposalLogoutServiceID, priAdmin2, adminNonce2, string(contracts.REJECTED))
	adminNonce2++

	suite.vote(proposalLogoutServiceID, priAdmin3, adminNonce3, string(contracts.REJECTED))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	err = json.Unmarshal(ret.Ret, service)
	suite.Require().Nil(err)
	suite.Equal(governance.GovernancePause, service.Status)
}

func (suite *Service) vote(proposalId string, adminKey crypto.PrivateKey, adminNonce uint64, info string) {
	ret, err := invokeBVMContract(suite.api, adminKey, adminNonce, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId),
		pb.String(info),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}
