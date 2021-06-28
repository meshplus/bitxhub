package tester

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

type Interchain struct {
	suite.Suite
	api api.CoreAPI
}

func (suite *Interchain) SetupSuite() {
}

func (suite *Interchain) TestHandleIBTP() {
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
	ibtpNonce := uint64(1)
	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	rawpub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub1 := base64.StdEncoding.EncodeToString(rawpub1)
	rawpub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2 := base64.StdEncoding.EncodeToString(rawpub2)

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(fmt.Sprintf("appchain%s", addr1.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String("rbft"),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(pub1),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id1 := string(gRet.Extra)
	proposalId1 := gRet.ProposalID

	//ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
	//suite.Require().Nil(err)
	//suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	//k1Nonce++

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(fmt.Sprintf("appchain%s", addr2.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String("rbft"),
		pb.String("fabric"),
		pb.String("税务链"),
		pb.String("fabric婚姻链"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	k2Nonce++
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id2 := string(gRet.Extra)
	proposalId2 := gRet.ProposalID
	fmt.Printf("appchain id 2 is %s\n", id2)
	fmt.Printf("proposal id is %s\n", proposalId2)

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id2))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId2),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId2),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId2),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	addr, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++

	// register rule
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(),
		"RegisterRule", pb.String(id1), pb.String(addr.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	k1Nonce++
	proposalRuleId := gjson.Get(string(ret.Ret), "proposal_id").String()

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalRuleId),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalRuleId),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalRuleId),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	proof := []byte("true")
	proofHash := sha256.Sum256(proof)
	ib := &pb.IBTP{From: id1, To: id2, Index: ibtpNonce, Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err := genIBTPTransaction(k1, ib, k1Nonce)
	suite.Require().Nil(err)
	k1Nonce++

	tx.Extra = proof
	ret, err = sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	ibtpNonce++
}

func (suite *Interchain) TestGetIBTPByID() {
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
	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())

	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k1Nonce := uint64(0)
	k2Nonce := uint64(0)
	ibtpNonce := uint64(1)

	rawpub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub1 := base64.StdEncoding.EncodeToString(rawpub1)
	rawpub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2 := base64.StdEncoding.EncodeToString(rawpub2)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	addr2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr2, 10000000000000))
	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	confByte, err := ioutil.ReadFile("./test_data/validator")
	suite.Require().Nil(err)

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(fmt.Sprintf("appchain%s", addr1.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(string(confByte)),
		pb.String("rbft"),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id1 := string(gRet.Extra)
	proposalId1 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(fmt.Sprintf("appchain%s", addr2.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String("rbft"),
		pb.String("fabric"),
		pb.String("税务链"),
		pb.String("fabric税务链"),
		pb.String("1.8"),
		pb.String(string(pub2)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id2 := string(gRet.Extra)
	proposalId2 := gRet.ProposalID

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id2))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId2),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId2),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId2),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	contractByte, err := ioutil.ReadFile("./test_data/fabric_policy.wasm")
	suite.Require().Nil(err)
	addr, err := deployContract(suite.api, k1, k1Nonce, contractByte)
	suite.Require().Nil(err)
	k1Nonce++

	// register rule
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(),
		"RegisterRule", pb.String(id1), pb.String(addr.String()))
	suite.Require().Nil(err)
	k1Nonce++
	proposalRuleId := gjson.Get(string(ret.Ret), "proposal_id").String()

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalRuleId),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalRuleId),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalRuleId),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	proof, err := ioutil.ReadFile("./test_data/proof")
	suite.Require().Nil(err)

	proofHash := sha256.Sum256(proof)
	ib := &pb.IBTP{From: id1, To: id2, Index: ibtpNonce, Payload: []byte("111"), Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err := genIBTPTransaction(k1, ib, k1Nonce)
	suite.Require().Nil(err)
	tx.Extra = proof
	receipt, err := sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().EqualValues(true, receipt.IsSuccess(), string(receipt.Ret))
	ibtpNonce++
	k1Nonce++

	ib2 := &pb.IBTP{From: id1, To: id2, Index: ibtpNonce, Payload: []byte("111"), Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err = genIBTPTransaction(k1, ib2, k1Nonce)
	suite.Require().Nil(err)
	tx.Extra = proof
	receipt, err = sendTransactionWithReceipt(suite.api, tx)
	suite.Require().Nil(err)
	suite.Require().EqualValues(true, receipt.IsSuccess(), string(receipt.Ret))
	ibtpNonce++
	k1Nonce++

	ib3 := &pb.IBTP{From: id1, To: id2, Index: ibtpNonce, Payload: []byte("111"), Timestamp: time.Now().UnixNano(), Proof: proofHash[:]}
	tx, err = genIBTPTransaction(k1, ib3, k1Nonce)
	suite.Require().Nil(err)
	tx.Extra = proof
	receipt, err = sendTransactionWithReceipt(suite.api, tx)
	suite.Assert().Nil(err)
	ibtpNonce++
	k1Nonce++

	ib.Index = 2
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.InterchainContractAddr.Address(), "GetIBTPByID", pb.String(ib.ID()))
	suite.Assert().Nil(err)
	suite.Assert().Equal(true, ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
}

func (suite *Interchain) TestInterchain() {
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
	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())

	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k1Nonce := uint64(0)

	rawpub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub1 := base64.StdEncoding.EncodeToString(rawpub1)
	addr1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr1, 10000000000000))
	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(fmt.Sprintf("appchain%s", addr1.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String("rbft"),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id1 := string(gRet.Extra)
	proposalId1 := gRet.ProposalID

	//ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(id1))
	//suite.Require().Nil(err)
	//suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	//k1Nonce++

	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.InterchainContractAddr.Address(),
		"Interchain", pb.String(id1))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	ic := &pb.Interchain{}
	err = ic.Unmarshal(ret.Ret)
	suite.Require().Nil(err)
	suite.Require().Equal(id1, ic.ID)
	suite.Require().Equal(0, len(ic.InterchainCounter))
	suite.Require().Equal(0, len(ic.ReceiptCounter))
	suite.Require().Equal(0, len(ic.SourceReceiptCounter))
	k1Nonce++
}

func (suite *Interchain) TestRegister() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := uint64(0)
	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))

	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.InterchainContractAddr.Address(), "Register", pb.String(from1.Address))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k1Nonce++
}
