package tester

import (
	"context"
	"fmt"
	"testing"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/key"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	rpcx "github.com/meshplus/go-bitxhub-client"
	"github.com/stretchr/testify/suite"
)

type Pier struct {
	suite.Suite
	privKey crypto.PrivateKey
	address types.Address
	client  rpcx.Client // bitxhub admin
}

func (suite *Pier) SetupSuite() {
	key, err := key.LoadKey("./test_data/key")
	suite.Require().Nil(err)

	privKey, err := key.GetPrivateKey("bitxhub")
	suite.Require().Nil(err)

	address, err := privKey.PublicKey().Address()
	suite.Require().Nil(err)

	client, err := rpcx.New(
		rpcx.WithPrivateKey(privKey),
		rpcx.WithAddrs(grpcAddresses()),
	)
	suite.Require().Nil(err)

	suite.privKey = privKey
	suite.address = address
	suite.client = client
}

func (suite *Pier) TestSyncMerkleWrapper() {
	privKey, err := ecdsa.GenerateKey(ecdsa.Secp256r1)
	suite.Assert().Nil(err)

	address, err := privKey.PublicKey().Address()
	suite.Require().Nil(err)

	client, err := rpcx.New(
		rpcx.WithPrivateKey(privKey),
		rpcx.WithAddrs(grpcAddresses()),
	)
	suite.Require().Nil(err)

	args := []*pb.Arg{
		rpcx.String(""),
		rpcx.Int32(0),
		rpcx.String("hyperchain"),
		rpcx.String("税务链"),
		rpcx.String("趣链税务链"),
		rpcx.String("1.8"),
	}

	ret, err := client.InvokeBVMContract(rpcx.InterchainContractAddr, "Register", args...)
	suite.Require().Nil(err)
	suite.Require().EqualValues("SUCCESS", ret.Status.String())

	go func() {
		w, err := client.Subscribe(context.Background(), pb.SubscriptionRequest_INTERCHAIN_TX_WRAPPER, address.Bytes())
		suite.Require().Nil(err)
		fmt.Println(<-w)
	}()

	res, err := suite.client.InvokeBVMContract(rpcx.InterchainContractAddr, "Audit",
		rpcx.String(address.Hex()),
		rpcx.Int32(1),
		rpcx.String("pass"),
	)
	suite.Require().Nil(err)
	suite.Require().EqualValues("SUCCESS", res.Status.String())
}

func TestPier(t *testing.T) {
	suite.Run(t, &Pier{})
}
