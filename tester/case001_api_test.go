package tester

import (
	"testing"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
)

const (
	value = "value"
)

type API struct {
	suite.Suite
	api     api.CoreAPI
	privKey crypto.PrivateKey
	from    types.Address
}

func (suite *API) SetupSuite() {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Assert().Nil(err)

	from, err := privKey.PublicKey().Address()
	suite.Assert().Nil(err)

	suite.privKey = privKey
	suite.from = from
}

func (suite *API) TestSend() {
	hash := testSendTransaction(suite)
	if err := retry.Retry(func(attempt uint) error {
		receipt, err := suite.api.Broker().GetReceipt(hash)
		if err != nil {
			return err
		}
		if receipt.IsSuccess() {
			return nil
		}
		return nil
	}, strategy.Wait(2*time.Second)); err != nil {
	}

	testSendView(suite)
}

func testSendTransaction(suite *API) types.Hash {
	tx, err := genContractTransaction(pb.TransactionData_BVM, suite.privKey,
		constant.StoreContractAddr.Address(), "Set", pb.String("key"), pb.String(value))
	suite.Nil(err)

	suite.Nil(suite.api.Broker().HandleTransaction(tx))
	return tx.TransactionHash
}

func testSendView(suite *API) {
	tx, err := genContractTransaction(pb.TransactionData_BVM, suite.privKey,
		constant.StoreContractAddr.Address(), "Get", pb.String("key"))

	receipt, err := suite.api.Broker().HandleView(tx)
	suite.Nil(err)
	suite.Equal(receipt.Status, pb.Receipt_SUCCESS)
	suite.Equal(value, string(receipt.Ret))
}

func TestAPI(t *testing.T) {
	suite.Run(t, &API{})
}
