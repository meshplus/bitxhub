package tester

import (
	"strconv"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

type RegisterAppchain struct {
	suite.Suite
	api     api.CoreAPI
	privKey crypto.PrivateKey
	from    types.Address
}

func (suite *RegisterAppchain) SetupSuite() {
	var err error
	suite.privKey, err = ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	suite.from, err = suite.privKey.PublicKey().Address()
	suite.Require().Nil(err)
}

// Appchain registers in bitxhub
func (suite *RegisterAppchain) TestRegisterAppchain() {
	pub, err := suite.privKey.PublicKey().Bytes()
	suite.Require().Nil(err)

	args := []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub)),
	}

	ret, err := invokeBVMContract(suite.api, suite.privKey, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Equal("hyperchain", gjson.Get(string(ret.Ret), "chain_type").String())
}

func (suite *RegisterAppchain) TestFetchAppchains() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)
	k2, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Require().Nil(err)

	pub1, err := k1.PublicKey().Bytes()
	suite.Require().Nil(err)
	pub2, err := k2.PublicKey().Bytes()
	suite.Require().Nil(err)

	args := []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(pub1)),
	}
	ret, err := invokeBVMContract(suite.api, k1, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))

	args = []*pb.Arg{
		pb.String(""),
		pb.Int32(0),
		pb.String("fabric"),
		pb.String("政务链"),
		pb.String("fabric政务"),
		pb.String("1.4"),
		pb.String(string(pub2)),
	}

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Register", args...)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	suite.Require().Nil(err)

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "Appchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())

	rec, err := invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "CountAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err := strconv.Atoi(string(rec.Ret))
	suite.Require().Nil(err)
	result := gjson.Parse(string(ret.Ret))
	suite.Require().GreaterOrEqual(num, len(result.Array()))

	ret, err = invokeBVMContract(suite.api, k2, constant.AppchainMgrContractAddr.Address(), "CountApprovedAppchains")
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	num, err = strconv.Atoi(string(ret.Ret))
	suite.Require().Nil(err)
	suite.Require().EqualValues(0, num)
}

func TestRegisterAppchain(t *testing.T) {
	suite.Run(t, &RegisterAppchain{})
}
