package tester

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

type Role struct {
	suite.Suite
	api     api.CoreAPI
	privKey crypto.PrivateKey
	pubKey  crypto.PublicKey
}

func (suite *Role) SetupSuite() {
	var err error
	suite.privKey, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Assert().Nil(err)

	suite.pubKey = suite.privKey.PublicKey()
}

func (suite *Role) TestGetRole() {
	pubKey, err := suite.pubKey.Bytes()
	suite.Assert().Nil(err)
	_, err = invokeBVMContract(suite.api, suite.privKey, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(string(pubKey)),
	)
	suite.Assert().Nil(err)

	receipt, err := invokeBVMContract(suite.api, suite.privKey, constant.RoleContractAddr.Address(), "GetRole")
	suite.Require().Nil(err)
	suite.Equal("appchain_admin", string(receipt.Ret))

	k, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	r, err := invokeBVMContract(suite.api, k, constant.RoleContractAddr.Address(), "GetRole")
	suite.Assert().Nil(err)
	suite.Equal("none", string(r.Ret))
}

func (suite *Role) TestGetAdminRoles() {
	k, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	r, err := invokeBVMContract(suite.api, k, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret := gjson.ParseBytes(r.Ret)
	suite.EqualValues(4, len(ret.Array()))
}

func (suite *Role) TestIsAdmin() {
	// Not Admin Chain
	k, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	from, err := k.PublicKey().Address()
	suite.Require().Nil(err)

	r, err := invokeBVMContract(suite.api, k, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(from.Hex()))
	suite.Assert().Nil(err)
	ret, err := strconv.ParseBool(string(r.Ret))
	suite.Assert().Nil(err)
	suite.EqualValues(false, ret)

	// Admin Chain
	path := "./test_data/config/node1/key.json"
	keyPath := filepath.Join(path)
	priAdmin, err := asym.RestorePrivateKey(keyPath, "bitxhub")
	suite.Require().Nil(err)
	fromAdmin, err := priAdmin.PublicKey().Address()
	suite.Require().Nil(err)

	r, err = invokeBVMContract(suite.api, priAdmin, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(fromAdmin.Hex()))
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess())
	ret, err = strconv.ParseBool(string(r.Ret))
	suite.Assert().Nil(err)
	suite.EqualValues(true, ret)
}

func (suite *Role) TestGetRuleAddress() {
	k1, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	k2, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	f1, err := k1.PublicKey().Address()
	suite.Require().Nil(err)
	f2, err := k2.PublicKey().Address()
	suite.Require().Nil(err)

	// Register
	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("婚姻链"),
		pb.String("趣链婚姻链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	appchain := Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain)
	suite.Require().Nil(err)
	id1 := appchain.ID

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	appchain = Appchain{}
	err = json.Unmarshal(ret.Ret, &appchain)
	suite.Require().Nil(err)
	id2 := appchain.ID

	// deploy rule
	bytes, err := ioutil.ReadFile("./test_data/hpc_rule.wasm")
	suite.Require().Nil(err)
	addr1, err := deployContract(suite.api, k1, bytes)
	suite.Require().Nil(err)

	bytes, err = ioutil.ReadFile("./test_data/fabric_policy.wasm")
	suite.Require().Nil(err)
	addr2, err := deployContract(suite.api, k2, bytes)
	suite.Require().Nil(err)

	suite.Require().NotEqual(addr1, addr2)

	// register rule
	ret, err = invokeBVMContract(suite.api, k1, constant.RuleManagerContractAddr.Address(), "RegisterRule", pb.String(f1.Hex()), pb.String(addr1.Hex()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	ret, err = invokeBVMContract(suite.api, k2, constant.RuleManagerContractAddr.Address(), "RegisterRule", pb.String(f2.Hex()), pb.String(addr2.Hex()))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	// get role address
	ret, err = invokeBVMContract(suite.api, k1, constant.RuleManagerContractAddr.Address(), "GetRuleAddress", pb.String(string(id1)), pb.String("hyperchain"))
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess())
	suite.Require().Equal(addr1.String(), string(ret.Ret))

	ret, err = invokeBVMContract(suite.api, k2, constant.RuleManagerContractAddr.Address(), "GetRuleAddress", pb.String(string(id2)), pb.String("fabric"))
	suite.Assert().Nil(err)
	suite.Require().True(ret.IsSuccess())
	suite.Require().Equal(addr2.String(), string(ret.Ret))
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

	// register
	retReg, err := invokeBVMContract(suite.api, priAdmin, constant.AppchainMgrContractAddr.Address(), "Register",
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("管理链"),
		pb.String("趣链管理链"),
		pb.String("1.8"),
		pb.String(string(pubAdmin)),
	)
	suite.Require().Nil(err)
	suite.Require().True(retReg.IsSuccess(), string(retReg.Ret))

	// is admin
	retIsAdmin, err := invokeBVMContract(suite.api, priAdmin, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(fromAdmin.Hex()))
	suite.Require().Nil(err)
	suite.Require().True(retIsAdmin.IsSuccess())

	// get admin roles
	r1, err := invokeBVMContract(suite.api, priAdmin, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret1 := gjson.ParseBytes(r1.Ret)
	suite.EqualValues(4, len(ret1.Array()))

	as := make([]string, 0)
	as = append(as, fromAdmin.Hex())
	data, err := json.Marshal(as)
	suite.Nil(err)

	// set admin roles
	r, err := invokeBVMContract(suite.api, priAdmin, constant.RoleContractAddr.Address(), "SetAdminRoles", pb.String(string(data)))
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess())

	// get admin roles
	r2, err := invokeBVMContract(suite.api, priAdmin, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret2 := gjson.ParseBytes(r2.Ret)
	suite.EqualValues(1, len(ret2.Array()))

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
	as2 = append(as2, fromAdmin.Hex(), fromAdmin2.Hex(), fromAdmin3.Hex(), fromAdmin4.Hex())
	data2, err := json.Marshal(as2)
	suite.Nil(err)
	r, err = invokeBVMContract(suite.api, priAdmin, constant.RoleContractAddr.Address(), "SetAdminRoles", pb.String(string(data2)))
	suite.Require().Nil(err)
	suite.Require().True(r.IsSuccess())

	// get Admin Roles
	r3, err := invokeBVMContract(suite.api, priAdmin, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret3 := gjson.ParseBytes(r3.Ret)
	suite.EqualValues(4, len(ret3.Array()))
}
