package tester

import (
	"strconv"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-model/pb"
	rpcx "github.com/meshplus/go-bitxhub-client"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

type RegisterAppchain struct {
	suite.Suite
	privKey crypto.PrivateKey
	client  rpcx.Client
}

func (suite *RegisterAppchain) SetupSuite() {
	var err error
	suite.privKey, err = ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)

	suite.client, err = rpcx.New(
		rpcx.WithPrivateKey(suite.privKey),
		rpcx.WithAddrs(grpcAddresses()),
	)
	suite.Assert().Nil(err)
}

// Appchain registers in bitxhub
func (suite *RegisterAppchain) TestRegisterAppchain() {
	suite.client.SetPrivateKey(suite.privKey)
	args := []*pb.Arg{
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("税务链"),
		rpcx.String("趣链税务链"),
		rpcx.String("1.8"),
	}
	ret, err := suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "Register", args...)
	suite.Assert().Nil(err)
	suite.Assert().Equal("hyperchain", gjson.Get(string(ret.Ret), "chain_type").String())
}

func (suite *RegisterAppchain) TestFetchAppchains() {
	k1, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)
	k2, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)

	suite.client.SetPrivateKey(k1)
	args := []*pb.Arg{
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("税务链"),
		rpcx.String("趣链税务链"),
		rpcx.String("1.8"),
	}
	_, err = suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "Register", args...)
	suite.Assert().Nil(err)

	suite.client.SetPrivateKey(k2)
	args = []*pb.Arg{
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("fabric"),
		rpcx.String("政务链"),
		rpcx.String("fabric政务"),
		rpcx.String("1.4"),
	}
	_, err = suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "Register", args...)

	suite.Assert().Nil(err)
	receipt, err := suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "Appchains")
	suite.Assert().Nil(err)

	rec, err := suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "CountAppchains")
	suite.Assert().Nil(err)
	num, err := strconv.Atoi(string(rec.Ret))
	suite.Assert().Nil(err)
	result := gjson.Parse(string(receipt.Ret))
	suite.Assert().EqualValues(num, len(result.Array()))

	r, err := suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "CountApprovedAppchains")
	suite.Assert().Nil(err)
	num, err = strconv.Atoi(string(r.Ret))
	suite.Assert().Nil(err)
	suite.Assert().EqualValues(0, num)
}

func TestRegisterAppchain(t *testing.T) {
	suite.Run(t, &RegisterAppchain{})
}
