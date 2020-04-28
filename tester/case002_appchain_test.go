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
	pubKey  crypto.PublicKey
	client  rpcx.Client
}

func (suite *RegisterAppchain) SetupSuite() {
	var err error
	suite.privKey, err = ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)

	suite.pubKey = suite.privKey.PublicKey()

	suite.client, err = rpcx.New(
		rpcx.WithPrivateKey(suite.privKey),
		rpcx.WithAddrs(grpcAddresses()),
	)
	suite.Assert().Nil(err)
}

// Appchain registers in bitxhub
func (suite *RegisterAppchain) TestRegisterAppchain() {
	suite.client.SetPrivateKey(suite.privKey)
	pubKey, err := suite.pubKey.Bytes()
	suite.Assert().Nil(err)
	args := []*pb.Arg{
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("税务链"),
		rpcx.String("趣链税务链"),
		rpcx.String("1.8"),
		rpcx.String(string(pubKey)),
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
	pk1, err := k1.PublicKey().Bytes()
	suite.Assert().Nil(err)
	args := []*pb.Arg{
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("税务链"),
		rpcx.String("趣链税务链"),
		rpcx.String("1.8"),
		rpcx.String(string(pk1)),
	}
	_, err = suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "Register", args...)
	suite.Assert().Nil(err)

	suite.client.SetPrivateKey(k2)
	pk2, err := k2.PublicKey().Bytes()
	suite.Assert().Nil(err)
	args = []*pb.Arg{
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("fabric"),
		rpcx.String("政务链"),
		rpcx.String("fabric政务"),
		rpcx.String("1.4"),
		rpcx.String(string(pk2)),
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
