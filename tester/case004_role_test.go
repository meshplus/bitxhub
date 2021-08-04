package tester

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/meshplus/bitxhub-core/governance"
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

type Role struct {
	suite.Suite
	api api.CoreAPI
}

func (suite *Role) SetupSuite() {
}

func (suite *Role) TestGetRole() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	rawpub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub1 := base64.StdEncoding.EncodeToString(rawpub1)

	from1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)

	suite.Require().Nil(transfer(suite.Suite, suite.api, from1, 10000000000000))
	fromaddr := from1.String()

	k1nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())

	_, err = invokeBVMContract(suite.api, k1, k1nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(fmt.Sprintf("appchain%s", fromaddr)),
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
	suite.Assert().Nil(err)
	k1nonce++

	receipt, err := invokeBVMContract(suite.api, k1, k1nonce, constant.RoleContractAddr.Address(), "GetRole")
	suite.Require().Nil(err)
	suite.Equal(string(contracts.AppchainAdmin), string(receipt.Ret))
	k1nonce++
}

func (suite *Role) TestGetAdminRoles() {
	k, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from, err := k.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, from, 10000000000000))
	kNonce := suite.api.Broker().GetPendingNonceByAccount(from.String())

	r, err := invokeBVMContract(suite.api, k, kNonce, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret := gjson.ParseBytes(r.Ret)
	suite.EqualValues(4, len(ret.Array()))
	kNonce++
}

func (suite *Role) TestIsAdmin() {
	// Not Admin Chain
	k, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from, err := k.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, from, 10000000000000))
	kNonce := suite.api.Broker().GetPendingNonceByAccount(from.String())

	r, err := invokeBVMContract(suite.api, k, kNonce, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(from.String()))
	suite.Assert().Nil(err)
	ret, err := strconv.ParseBool(string(r.Ret))
	suite.Assert().Nil(err)
	suite.EqualValues(false, ret)
	kNonce++

	// Admin Chain
	path := "./test_data/config/node1/key.json"
	keyPath := filepath.Join(path)
	priAdmin, err := asym.RestorePrivateKey(keyPath, "bitxhub")
	suite.Require().Nil(err)
	fromAdmin, err := priAdmin.PublicKey().Address()
	suite.Require().Nil(err)
	adminNonce := suite.api.Broker().GetPendingNonceByAccount(fromAdmin.String())

	r, err = invokeBVMContract(suite.api, priAdmin, adminNonce, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(fromAdmin.String()))
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess())
	ret, err = strconv.ParseBool(string(r.Ret))
	suite.Assert().Nil(err)
	suite.EqualValues(true, ret)
	adminNonce++
}

func (suite *Role) TestGetRuleAddress() {
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
	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
	from1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	k1Nonce := suite.api.Broker().GetPendingNonceByAccount(from1.String())
	from2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)
	k2Nonce := suite.api.Broker().GetPendingNonceByAccount(from2.String())

	rawpub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub1 := base64.StdEncoding.EncodeToString(rawpub1)
	rawpub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2 := base64.StdEncoding.EncodeToString(rawpub2)

	// Register
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
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(pub2),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	k2Nonce++
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	id2 := string(gRet.Extra)
	proposalId2 := gRet.ProposalID
	// Fabric automatically deploys validation rules

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
	ruleAddr1, err := deployContract(suite.api, k1, k1Nonce, bytes)
	suite.Require().Nil(err)
	k1Nonce++

	bytes, err = ioutil.ReadFile("./test_data/fabric_policy.wasm")
	suite.Require().Nil(err)
	ruleAddr2, err := deployContract(suite.api, k2, k2Nonce, bytes)
	suite.Require().Nil(err)
	k2Nonce++

	suite.Require().NotEqual(ruleAddr1, ruleAddr2)

	// register rule
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "RegisterRule", pb.String(id1), pb.String(ruleAddr1.String()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
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

	// get role address
	ret, err = invokeBVMContract(suite.api, k1, k1Nonce, constant.RuleManagerContractAddr.Address(), "GetAvailableRuleAddr", pb.String(id1))
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Equal(ruleAddr1.String(), string(ret.Ret))
	k1Nonce++

	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.RuleManagerContractAddr.Address(), "GetAvailableRuleAddr", pb.String(id2))
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess())
	suite.Require().Equal(validator.FabricRuleAddr, string(ret.Ret))
	k2Nonce++
}

func (suite *Role) TestRegisterRoles() {
	// admin chain
	path1 := "./test_data/config/node1/key.json"
	path2 := "./test_data/config/node2/key.json"
	path3 := "./test_data/config/node3/key.json"
	path4 := "./test_data/config/node4/key.json"
	path5 := "./test_data/key.json"

	keyPath1 := filepath.Join(path1)
	keyPath2 := filepath.Join(path2)
	keyPath3 := filepath.Join(path3)
	keyPath4 := filepath.Join(path4)
	keyPath5 := filepath.Join(path5)

	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	suite.Require().Nil(err)
	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	suite.Require().Nil(err)
	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	suite.Require().Nil(err)
	priAdmin4, err := asym.RestorePrivateKey(keyPath4, "bitxhub")
	suite.Require().Nil(err)
	priAdmin5, err := asym.RestorePrivateKey(keyPath5, "bitxhub")
	suite.Require().Nil(err)

	fromAdmin1, err := priAdmin1.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin4, err := priAdmin4.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin5, err := priAdmin5.PublicKey().Address()
	suite.Require().Nil(err)

	pubAppAdmin, err := priAdmin5.PublicKey().Bytes()
	suite.Require().Nil(err)

	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())
	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())
	adminNonce4 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin4.String())

	// register
	retReg, err := invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(fmt.Sprintf("appchain%s", fromAdmin1.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String("rbft"),
		pb.String("hyperchain"),
		pb.String("管理链"),
		pb.String("趣链管理链"),
		pb.String("1.8"),
		pb.String(base64.StdEncoding.EncodeToString(pubAppAdmin)),
	)
	suite.Require().Nil(err)
	suite.Require().True(retReg.IsSuccess(), string(retReg.Ret))
	adminNonce1++

	// is admin
	retIsAdmin, err := invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(fromAdmin1.String()))
	suite.Require().Nil(err)
	suite.Require().True(retIsAdmin.IsSuccess())
	adminNonce1++

	// get admin roles
	r1, err := invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret1 := gjson.ParseBytes(r1.Ret)
	suite.EqualValues(4, len(ret1.Array()))
	adminNonce1++

	// ！！！Adding an administrator may affect other integration tests, so this section is commented out
	// register role
	r, err := invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.RoleContractAddr.Address(), "RegisterRole",
		pb.String(fromAdmin5.String()),
		pb.String(string(contracts.GovernanceAdmin)),
		pb.String(""),
	)
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess(), string(r.Ret))
	adminNonce1++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(r.Ret, gRet)
	suite.Require().Nil(err)
	proposalId1 := gRet.ProposalID

	// vote
	r, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess(), string(r.Ret))
	adminNonce2++

	r, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess(), string(r.Ret))
	adminNonce3++

	r, err = invokeBVMContract(suite.api, priAdmin4, adminNonce4, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(proposalId1),
		pb.String(string(contracts.APPOVED)),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess(), string(r.Ret))
	adminNonce4++

	// get admin roles
	r2, err := invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret2 := gjson.ParseBytes(r2.Ret)
	suite.EqualValues(5, len(ret2.Array()))
	adminNonce1++
}
