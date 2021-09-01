package tester

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
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
	from    *types.Address
}

func (suite *API) SetupSuite() {
	privKey, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Assert().Nil(err)

	from, err := privKey.PublicKey().Address()
	suite.Assert().Nil(err)

	suite.privKey = privKey
	suite.from = from

	err = transfer(suite.Suite, suite.api, from, 10000000000000)
	suite.Require().Nil(err)
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

func testSendTransaction(suite *API) *types.Hash {
	tx, err := genContractTransaction(pb.TransactionData_BVM, suite.privKey, 0,
		constant.StoreContractAddr.Address(), "Set", pb.String("key"), pb.String(value))
	suite.Nil(err)

	fmt.Printf("api is %v\n", suite.api)
	suite.Nil(suite.api.Broker().HandleTransaction(tx))
	return tx.GetHash()
}

func testSendView(suite *API) {
	tx, err := genContractTransaction(pb.TransactionData_BVM, suite.privKey, 0,
		constant.StoreContractAddr.Address(), "Get", pb.String("key"))

	receipt, err := suite.api.Broker().HandleView(tx)
	suite.Nil(err)
	suite.Equal(receipt.Status, pb.Receipt_SUCCESS)
	suite.Equal(value, string(receipt.Ret))
}

func transfer(suite suite.Suite, api api.CoreAPI, address *types.Address, amount uint64) error {
	keyPath1 := filepath.Join("./test_data/config/node4/key.json")
	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	suite.Require().Nil(err)

	fromAdmin1, err := priAdmin1.PublicKey().Address()
	suite.Require().Nil(err)
	adminNonce1 := api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	tx, err := genTransferTransaction(priAdmin1, adminNonce1, address, amount)
	suite.Require().Nil(err)
	adminNonce1++

	err = api.Broker().HandleTransaction(tx)
	suite.Require().Nil(err)

	hash := tx.Hash()

	if err := retry.Retry(func(attempt uint) error {
		receipt, err := api.Broker().GetReceipt(hash)
		if err != nil {
			return err
		}
		if receipt.IsSuccess() {
			return nil
		}
		return nil
	}, strategy.Wait(1*time.Second)); err != nil {
	}

	return nil
}

//
//func (suite *API) TestDelVPNode() {
//	err := suite.api.Broker().DelVPNode(1)
//	suite.NotNil(err)
//
//	err = suite.api.Broker().DelVPNode(2)
//	suite.Nil(err)
//}

func TestAPI(t *testing.T) {
	suite.Run(t, &API{})
}
