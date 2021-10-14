package tester

import (
	"crypto/sha256"
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

type Interchain struct {
	suite.Suite
	api         api.CoreAPI
	priAdmin1   crypto.PrivateKey
	priAdmin2   crypto.PrivateKey
	priAdmin3   crypto.PrivateKey
	adminNonce1 uint64
	adminNonce2 uint64
	adminNonce3 uint64

	k1 crypto.PrivateKey
	k2 crypto.PrivateKey

	chainID1   string
	chainID2   string
	serviceID1 string
	serviceID2 string
	serviceID3 string

	ibtpNonce uint64
}

func (suite *Interchain) SetupSuite() {
	suite.ibtpNonce = 1
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
	suite.priAdmin1 = priAdmin1
	suite.priAdmin2 = priAdmin2
	suite.priAdmin3 = priAdmin3

	fromAdmin1, err := priAdmin1.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	suite.Require().Nil(err)

	suite.adminNonce1 = suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
	suite.adminNonce2 = suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	suite.adminNonce3 = suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())

	suite.k1, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	suite.k2, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	suite.Require().Nil(err)
	addr1, err := suite.k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := suite.k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))

	chainName1 := "应用链1case003"
	chainName2 := "应用链2case003"
	suite.chainID1 = suite.registerAppchain(suite.k1, chainName1, validator.HappyRuleAddr, appchainMgr.ChainTypeETH, addr1.String())
	suite.chainID2 = suite.registerAppchain(suite.k2, chainName2, validator.HappyRuleAddr, appchainMgr.ChainTypeETH, addr2.String())

	suite.serviceID1 = "service1"
	suite.serviceID2 = "service2"
	suite.serviceID3 = "service3"

	fullServiceID1 := fmt.Sprintf("1356:%s:%s", suite.chainID1, suite.serviceID1)
	fullServiceID2 := fmt.Sprintf("1356:%s:%s", suite.chainID2, suite.serviceID2)
	fullServiceID3 := fmt.Sprintf("1356:%s:%s", suite.chainID1, suite.serviceID3)

	suite.registerService(suite.k1, suite.chainID1, suite.serviceID1, "服务1case003", "")
	suite.registerService(suite.k1, suite.chainID1, suite.serviceID3, "服务2case003", "")
	suite.registerService(suite.k2, suite.chainID2, suite.serviceID2, "服务3case003", fullServiceID3)

	proof := []byte("true")
	proofHash := sha256.Sum256(proof)
	ib := &pb.IBTP{From: fullServiceID1, To: fullServiceID2, Index: suite.ibtpNonce, TimeoutHeight: 10, Proof: proofHash[:]}
	ret, err := suite.sendIBTPTx(suite.k1, ib, proof)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.ibtpNonce++

	ibRec := &pb.IBTP{From: fullServiceID1, To: fullServiceID2, Index: suite.ibtpNonce - 1, TimeoutHeight: 10, Proof: proofHash[:], Type: pb.IBTP_RECEIPT_SUCCESS}
	ret, err = suite.sendIBTPTx(suite.k2, ibRec, proof)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}

func (suite *Interchain) TestHandleIBTP() {
	fullServiceID1 := fmt.Sprintf("1356:%s:%s", suite.chainID1, suite.serviceID1)
	fullServiceID2 := fmt.Sprintf("1356:%s:%s", suite.chainID2, suite.serviceID2)
	fullServiceID3 := fmt.Sprintf("1356:%s:%s", suite.chainID1, suite.serviceID3)
	unregisterBxhServiceID := fmt.Sprintf("1357:%s:%s", suite.chainID2, suite.serviceID2)

	keyAddr, err := suite.k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(keyAddr.String())

	keyAddr, err = suite.k2.PublicKey().Address()
	suite.Require().Nil(err)
	k2Nonce := suite.api.Broker().GetPendingNonceByAccount(keyAddr.String())

	proof := []byte("true")
	proofHash := sha256.Sum256(proof)
	ib := &pb.IBTP{From: fullServiceID1, To: fullServiceID2, Index: suite.ibtpNonce, TimeoutHeight: 10, Proof: proofHash[:]}
	tx, err := genIBTPTransaction(suite.k1, ib, k1Nonce)
	suite.Require().Nil(err)
	k1Nonce++

	tx.Extra = proof
	ret, err := sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.ibtpNonce++

	ibRec := &pb.IBTP{From: fullServiceID1, To: fullServiceID2, Index: suite.ibtpNonce - 1, TimeoutHeight: 10, Proof: proofHash[:], Type: pb.IBTP_RECEIPT_SUCCESS}
	tx1, err := genIBTPTransaction(suite.k2, ibRec, k2Nonce)
	suite.Require().Nil(err)
	k2Nonce++

	tx1.Extra = proof
	ret, err = sendTransactionWithReceipt(suite.api, tx1)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	ib1 := &pb.IBTP{From: fullServiceID1, To: unregisterBxhServiceID, Index: 1, TimeoutHeight: 10, Proof: proofHash[:]}
	tx, err = genIBTPTransaction(suite.k1, ib1, k1Nonce)
	suite.Require().Nil(err)
	k1Nonce++

	tx.Extra = proof
	ret, err = sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	ib2 := &pb.IBTP{From: fullServiceID3, To: fullServiceID2, Index: 1, TimeoutHeight: 10, Proof: proofHash[:]}
	tx2, err := genIBTPTransaction(suite.k1, ib2, k1Nonce)
	suite.Require().Nil(err)
	k1Nonce++

	tx2.Extra = proof
	ret, err = sendTransactionWithReceipt(suite.api, tx2)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	addr, sign, err := suite.api.Broker().GetSign(ib.ID(), pb.GetMultiSignsRequest_IBTP_REQUEST)
	suite.Require().Nil(err)
	suite.Require().Equal(65, len(sign), fmt.Sprintf("signature's length is %d", len(sign)))
	suite.Require().NotEqual("", addr, fmt.Sprintf("signer is %s", addr))

	signM := suite.api.Broker().FetchSignsFromOtherPeers(ib.ID(), pb.GetMultiSignsRequest_IBTP_REQUEST)
	suite.Require().Equal(3, len(signM), fmt.Sprintf("signM's size is %d", len(signM)))

	for addr, sign := range signM {
		suite.Require().Equal(65, len(sign), fmt.Sprintf("signature's length is %d", len(sign)))
		suite.Require().NotEqual("", addr, fmt.Sprintf("signer is %s", addr))
	}
}

func (suite *Interchain) TestGetIBTPByID() {
	fullServiceID1 := fmt.Sprintf("1356:%s:%s", suite.chainID1, suite.serviceID1)
	fullServiceID2 := fmt.Sprintf("1356:%s:%s", suite.chainID2, suite.serviceID2)

	keyAddr, err := suite.k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(keyAddr.String())

	ib := &pb.IBTP{From: fullServiceID1, To: fullServiceID2, Index: 1}
	ret, err := invokeBVMContract(suite.api, suite.k1, k1Nonce, constant.InterchainContractAddr.Address(), "GetIBTPByID", pb.String(ib.ID()), pb.Bool(true))
	suite.Assert().Nil(err)
	suite.Assert().Equal(true, ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret1, err := invokeBVMContract(suite.api, suite.k1, k1Nonce, constant.InterchainContractAddr.Address(), "GetIBTPByID", pb.String(ib.ID()), pb.Bool(false))
	suite.Assert().Nil(err)
	suite.Assert().Equal(true, ret1.IsSuccess(), string(ret.Ret))
	k1Nonce++
}

func (suite *Interchain) TestInterchain() {
	fullServiceID := fmt.Sprintf("1356:%s:%s", suite.chainID1, suite.serviceID1)

	keyAddr, err := suite.k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(keyAddr.String())

	ret, err := invokeBVMContract(suite.api, suite.k1, k1Nonce, constant.InterchainContractAddr.Address(),
		"GetInterchain", pb.String(fullServiceID))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	ic := &pb.Interchain{}
	err = ic.Unmarshal(ret.Ret)
	suite.Require().Nil(err)
	suite.Require().Equal(fullServiceID, ic.ID)
	suite.Require().GreaterOrEqual(len(ic.InterchainCounter), 1)
	suite.Require().GreaterOrEqual(len(ic.ReceiptCounter), 1)
	suite.Require().Equal(0, len(ic.SourceReceiptCounter))
	k1Nonce++
}

func (suite *Interchain) TestRegister() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.InterchainContractAddr.Address(), "Register", pb.String(from1.Address))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
}

func (suite *Interchain) registerAppchain(privKey crypto.PrivateKey, chainName, rule, chainType, adminAddrs string) string {
	addr, err := privKey.PublicKey().Address()
	suite.Require().Nil(err)
	keyNonce := suite.api.Broker().GetPendingNonceByAccount(addr.String())

	fabricBroker := appchainMgr.FabricBroker{
		ChannelID:     "1",
		ChaincodeID:   "2",
		BrokerVersion: "3",
	}
	fabricBrokerData, err := json.Marshal(fabricBroker)
	suite.Require().Nil(err)
	brokerData := []byte("123")
	if chainType == appchainMgr.ChainTypeFabric1_4_3 || chainType == appchainMgr.ChainTypeFabric1_4_4 {
		brokerData = fabricBrokerData
	}
	args := []*pb.Arg{
		pb.String(chainName),
		pb.String(chainType),
		pb.Bytes(nil),
		pb.Bytes(brokerData),
		pb.String("desc"),
		pb.String(rule),
		pb.String("url"),
		pb.String(adminAddrs),
		pb.String("reason"),
	}
	ret, err := invokeBVMContract(suite.api, privKey, keyNonce, constant.AppchainMgrContractAddr.Address(), "RegisterAppchain",
		args...,
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	keyNonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalId1 := gRet.ProposalID

	suite.vote(proposalId1, suite.priAdmin1, suite.adminNonce1)
	suite.adminNonce1++

	suite.vote(proposalId1, suite.priAdmin2, suite.adminNonce2)
	suite.adminNonce2++

	suite.vote(proposalId1, suite.priAdmin3, suite.adminNonce3)
	suite.adminNonce3++

	ret, err = invokeBVMContract(suite.api, privKey, keyNonce, constant.AppchainMgrContractAddr.Address(), "GetAppchainByName", pb.String(chainName))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	keyNonce++
	chainInfo := &appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, chainInfo)
	suite.Require().Nil(err)
	suite.Require().Equal(governance.GovernanceAvailable, chainInfo.Status)
	chainID := chainInfo.ID

	ret, err = invokeBVMContract(suite.api, privKey, keyNonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainID))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	keyNonce++

	return chainID
}

func (suite *Interchain) registerService(privKey crypto.PrivateKey, chainID, serviceID, serviceName, blacklist string) {
	addr, err := privKey.PublicKey().Address()
	suite.Require().Nil(err)
	keyNonce := suite.api.Broker().GetPendingNonceByAccount(addr.String())

	ret, err := invokeBVMContract(suite.api, privKey, keyNonce, constant.ServiceMgrContractAddr.Address(), "RegisterService",
		pb.String(chainID),
		pb.String(serviceID),
		pb.String(serviceName),
		pb.String(string(service_mgr.ServiceCallContract)),
		pb.String("intro"),
		pb.Bool(true),
		pb.String(blacklist),
		pb.String("details"),
		pb.String("raeson"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	keyNonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	proposalServiceId1 := gRet.ProposalID

	chainServiceID := fmt.Sprintf("%s:%s", chainID, serviceID)
	ret, err = invokeBVMContract(suite.api, privKey, keyNonce, constant.ServiceMgrContractAddr.Address(), "GetServiceInfo", pb.String(chainServiceID))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	keyNonce++

	suite.vote(proposalServiceId1, suite.priAdmin1, suite.adminNonce1)
	suite.adminNonce1++

	suite.vote(proposalServiceId1, suite.priAdmin2, suite.adminNonce2)
	suite.adminNonce2++

	suite.vote(proposalServiceId1, suite.priAdmin3, suite.adminNonce3)
	suite.adminNonce3++
}

func (suite *Interchain) vote(proposalId string, adminKey crypto.PrivateKey, adminNonce uint64) {
	ret, err := invokeBVMContract(suite.api, adminKey, adminNonce, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId),
		pb.String(string(contracts.APPROVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
}

func (suite *Interchain) sendIBTPTx(privKey crypto.PrivateKey, ibtp *pb.IBTP, proof []byte) (*pb.Receipt, error) {
	addr, err := privKey.PublicKey().Address()
	suite.Require().Nil(err)
	keyNonce := suite.api.Broker().GetPendingNonceByAccount(addr.String())

	tx, err := genIBTPTransaction(privKey, ibtp, keyNonce)
	suite.Require().Nil(err)

	tx.Extra = proof

	return sendTransactionWithReceipt(suite.api, tx)
}
