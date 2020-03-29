package tester

import (
	"strconv"

	"github.com/meshplus/bitxhub/internal/constant"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	rpcx "github.com/meshplus/go-bitxhub-client"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

type Role struct {
	suite.Suite
	privKey crypto.PrivateKey
	client  rpcx.Client
}

func (suite *Role) SetupSuite() {
	var err error
	suite.privKey, err = ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)

	suite.client, err = rpcx.New(
		rpcx.WithPrivateKey(suite.privKey),
		rpcx.WithAddrs([]string{
			"localhost:60011",
			"localhost:60012",
			"localhost:60013",
			"localhost:60014",
		}),
	)
	suite.Require().Nil(err)
}

func (suite *Role) TestGetRole() {
	_, err := suite.client.InvokeBVMContract(constant.InterchainContractAddr.Address(), "Register",
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("婚姻链"),
		rpcx.String("趣链婚姻链"),
		rpcx.String("1.8"),
	)
	suite.Assert().Nil(err)

	receipt, err := suite.client.InvokeBVMContract(constant.RoleContractAddr.Address(), "GetRole")
	suite.Require().Nil(err)
	suite.Equal("appchain_admin", string(receipt.Ret))

	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	suite.client.SetPrivateKey(k)
	r, err := suite.client.InvokeBVMContract(constant.RoleContractAddr.Address(), "GetRole")
	suite.Assert().Nil(err)
	suite.Equal("none", string(r.Ret))
}

func (suite *Role) TestGetAdminRoles() {
	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	suite.client.SetPrivateKey(k)
	r, err := suite.client.InvokeBVMContract(constant.RoleContractAddr.Address(), "GetAdminRoles")
	suite.Assert().Nil(err)
	ret := gjson.ParseBytes(r.Ret)
	suite.EqualValues(4, len(ret.Array()))
}

func (suite *Role) TestIsAdmin() {
	k, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)
	from, err := k.PublicKey().Address()
	suite.Require().Nil(err)

	suite.client.SetPrivateKey(k)
	r, err := suite.client.InvokeBVMContract(constant.RoleContractAddr.Address(), "IsAdmin", rpcx.String(from.Hex()))
	suite.Assert().Nil(err)
	ret, err := strconv.ParseBool(string(r.Ret))
	suite.Assert().Nil(err)
	suite.EqualValues(false, ret)
}
