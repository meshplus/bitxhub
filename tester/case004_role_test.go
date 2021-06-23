package tester

import (
	"encoding/base64"
	"encoding/json"
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
	"github.com/meshplus/bitxid"
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

	did := genUniqueAppchainDID(fromaddr)
	_, err = invokeBVMContract(suite.api, k1, k1nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
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
	suite.Equal("appchain_admin", string(receipt.Ret))
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

	did := genUniqueAppchainDID(addr1.String())
	// Register
	ret, err := invokeBVMContract(suite.api, k1, k1Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
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

	did2 := genUniqueAppchainDID(addr2.String())
	ret, err = invokeBVMContract(suite.api, k2, k2Nonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(did2),
		pb.String(string(bitxid.DID(did2).GetChainDID())),
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

func (suite *Role) TestSetAdminRoles() {
	// admin chain
	path1 := "./test_data/config/node1/key.json"
	keyPath1 := filepath.Join(path1)
	priAdmin, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	suite.Require().Nil(err)
	fromAdmin, err := priAdmin.PublicKey().Address()
	suite.Require().Nil(err)
	pubAdmin, err := priAdmin.PublicKey().Bytes()
	suite.Require().Nil(err)
	adminNonce := suite.api.Broker().GetPendingNonceByAccount(fromAdmin.String())

	// register
	did := genUniqueAppchainDID(fromAdmin.String())
	retReg, err := invokeBVMContract(suite.api, priAdmin, adminNonce, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(did),
		pb.String(string(bitxid.DID(did).GetChainDID())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String(""),
		pb.String("rbft"),
		pb.String("hyperchain"),
		pb.String("管理链"),
		pb.String("趣链管理链"),
		pb.String("1.8"),
		pb.String(base64.StdEncoding.EncodeToString(pubAdmin)),
	)
	suite.Require().Nil(err)
	suite.Require().True(retReg.IsSuccess(), string(retReg.Ret))
	adminNonce++

	// is admin
	retIsAdmin, err := invokeBVMContract(suite.api, priAdmin, adminNonce, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(fromAdmin.String()))
	suite.Require().Nil(err)
	suite.Require().True(retIsAdmin.IsSuccess())
	adminNonce++

	// get admin roles
	r1, err := invokeBVMContract(suite.api, priAdmin, adminNonce, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret1 := gjson.ParseBytes(r1.Ret)
	suite.EqualValues(4, len(ret1.Array()))
	adminNonce++

	as := make([]string, 0)
	as = append(as, fromAdmin.String())
	data, err := json.Marshal(as)
	suite.Nil(err)

	// set admin roles
	r, err := invokeBVMContract(suite.api, priAdmin, adminNonce, constant.RoleContractAddr.Address(), "SetAdminRoles", pb.String(string(data)))
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess())
	adminNonce++

	// get admin roles
	r2, err := invokeBVMContract(suite.api, priAdmin, adminNonce, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret2 := gjson.ParseBytes(r2.Ret)
	suite.EqualValues(1, len(ret2.Array()))
	adminNonce++

	// set more admin roles
	path2 := "./test_data/config/node2/key.json"
	path3 := "./test_data/config/node3/key.json"
	path4 := "./test_data/config/node4/key.json"

	keyPath2 := filepath.Join(path2)
	keyPath3 := filepath.Join(path3)
	keyPath4 := filepath.Join(path4)

	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	suite.Require().Nil(err)
	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	suite.Require().Nil(err)
	priAdmin4, err := asym.RestorePrivateKey(keyPath4, "bitxhub")
	suite.Require().Nil(err)

	fromAdmin2, err := priAdmin2.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin4, err := priAdmin4.PublicKey().Address()
	suite.Require().Nil(err)

	// set admin roles
	as2 := make([]string, 0)
	as2 = append(as2, fromAdmin.String(), fromAdmin2.String(), fromAdmin3.String(), fromAdmin4.String())
	data2, err := json.Marshal(as2)
	suite.Nil(err)
	r, err = invokeBVMContract(suite.api, priAdmin, adminNonce, constant.RoleContractAddr.Address(), "SetAdminRoles", pb.String(string(data2)))
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess())
	adminNonce++

	// get Admin Roles
	r3, err := invokeBVMContract(suite.api, priAdmin, adminNonce, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret3 := gjson.ParseBytes(r3.Ret)
	suite.EqualValues(4, len(ret3.Array()))
	adminNonce++
}
