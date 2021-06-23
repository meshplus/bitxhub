package tester

import (
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
)

type Store struct {
	suite.Suite
	api     api.CoreAPI
	privKey crypto.PrivateKey
	pubKey  crypto.PublicKey
}

func (suite *Store) SetupSuite() {
	var err error
	suite.privKey, err = asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Assert().Nil(err)

	suite.pubKey = suite.privKey.PublicKey()
}

func (suite *Store) TestStore() {
	k, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	kNonce := uint64(0)

	args := []*pb.Arg{
		pb.String("123"),
		pb.String("abc"),
	}

	addr, err := k.PublicKey().Address()
	suite.Require().Nil(err)
	suite.Require().Nil(transfer(suite.Suite, suite.api, addr, 10000000000000))

	ret, err := invokeBVMContract(suite.api, k, kNonce, constant.StoreContractAddr.Address(), "Set", args...)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess())
	kNonce++

	ret2, err := invokeBVMContract(suite.api, k, kNonce, constant.StoreContractAddr.Address(), "Get", pb.String("123"))
	suite.Require().Nil(err)
	suite.Require().True(ret2.IsSuccess())
	suite.Require().Equal("abc", string(ret2.Ret))
	kNonce++
}
