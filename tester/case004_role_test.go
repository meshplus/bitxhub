package tester

import (
	"strconv"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
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
	suite.privKey, err = ecdsa.GenerateKey(ecdsa.Secp256r1)
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

	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	r, err := invokeBVMContract(suite.api, k, constant.RoleContractAddr.Address(), "GetRole")
	suite.Assert().Nil(err)
	suite.Equal("none", string(r.Ret))
}

func (suite *Role) TestGetAdminRoles() {
	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	r, err := invokeBVMContract(suite.api, k, constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret := gjson.ParseBytes(r.Ret)
	suite.EqualValues(4, len(ret.Array()))
}

func (suite *Role) TestIsAdmin() {
	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)
	from, err := k.PublicKey().Address()
	suite.Require().Nil(err)

	r, err := invokeBVMContract(suite.api, k, constant.RoleContractAddr.Address(), "IsAdmin", pb.String(from.Hex()))
	suite.Assert().Nil(err)
	ret, err := strconv.ParseBool(string(r.Ret))
	suite.Assert().Nil(err)
	suite.EqualValues(false, ret)
}
