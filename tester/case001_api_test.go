package tester

import (
	"math/rand"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	rpcx "github.com/meshplus/go-bitxhub-client"

	"github.com/stretchr/testify/suite"
)

type API struct {
	suite.Suite
	client  rpcx.Client
	privKey crypto.PrivateKey
	from    types.Address
}

func (suite *API) SetupSuite() {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	suite.Assert().Nil(err)

	client, err := rpcx.New(
		rpcx.WithPrivateKey(privKey),
		rpcx.WithAddrs(grpcAddresses()),
	)
	suite.Assert().Nil(err)

	from, err := privKey.PublicKey().Address()
	suite.Assert().Nil(err)

	suite.client = client
	suite.privKey = privKey
	suite.from = from
}

func (suite *API) TestSendTransaction() {
	pl := &pb.InvokePayload{
		Method: "Set",
		Args: []*pb.Arg{
			pb.String("key"),
			pb.String("value"),
		},
	}

	data, err := pl.Marshal()
	suite.Assert().Nil(err)

	tx := &pb.Transaction{
		From:      suite.from,
		To:        rpcx.StoreContractAddr,
		Timestamp: time.Now().UnixNano(),
		Data: &pb.TransactionData{
			Type:    pb.TransactionData_INVOKE,
			VmType:  pb.TransactionData_BVM,
			Payload: data,
		},
		Nonce: rand.Int63(),
	}
	err = tx.Sign(suite.privKey)
	suite.Nil(err)
	hash, err := suite.client.SendTransaction(tx)
	suite.Nil(err)
	suite.NotEmpty(hash)
}

func (suite *API) TestSendWrongTransaction() {
	tx := &pb.Transaction{
		To:              suite.from,
		TransactionHash: types.Hash{},
		Nonce:           rand.Int63(),
	}

	_, err := suite.client.SendTransaction(tx)
	suite.Require().NotNil(err)
	suite.Contains(err.Error(), "tx data can't be empty")

	tx.Data = &pb.TransactionData{}
	_, err = suite.client.SendTransaction(tx)
	suite.Require().NotNil(err)
	suite.Contains(err.Error(), "amount can't be 0 in transfer tx")

	tx.Data = &pb.TransactionData{Amount: 10}
	_, err = suite.client.SendTransaction(tx)
	suite.Require().NotNil(err)
	suite.Contains(err.Error(), "from can't be empty")

	tx.From = suite.from
	_, err = suite.client.SendTransaction(tx)
	suite.Require().NotNil(err)
	suite.Contains(err.Error(), "from can`t be the same as to")

	tx.To = rpcx.StoreContractAddr
	_, err = suite.client.SendTransaction(tx)
	suite.Require().NotNil(err)
	suite.Contains(err.Error(), "timestamp is illegal")

	tx.Nonce = -100
	tx.Timestamp = time.Now().UnixNano()
	_, err = suite.client.SendTransaction(tx)
	suite.Require().NotNil(err)
	suite.Contains(err.Error(), "nonce is illegal")

	tx.Nonce = rand.Int63()

	_, err = suite.client.SendTransaction(tx)
	suite.Require().NotNil(err)
	suite.Contains(err.Error(), "signature can't be empty")

	err = tx.Sign(suite.privKey)
	suite.Require().Nil(err)
	hash, err := suite.client.SendTransaction(tx)
	suite.Require().Nil(err)
	suite.Assert().NotEmpty(hash)
}

func (suite *API) TestGetBlockByHash() {
}

func TestAPI(t *testing.T) {
	suite.Run(t, &API{})
}
