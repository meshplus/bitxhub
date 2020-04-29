package tester

import (
	"math/rand"
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/constant"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/stretchr/testify/suite"
)

type API struct {
	suite.Suite
	api     api.CoreAPI
	privKey crypto.PrivateKey
	from    types.Address
}

func (suite *API) SetupSuite() {
	privKey, err := asym.GenerateKey(asym.ECDSASecp256r1)
	suite.Assert().Nil(err)

	from, err := privKey.PublicKey().Address()
	suite.Assert().Nil(err)

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
		To:        constant.StoreContractAddr.Address(),
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

	tx.TransactionHash = tx.Hash()

	err = suite.api.Broker().HandleTransaction(tx)
	suite.Nil(err)
}

func TestAPI(t *testing.T) {
	suite.Run(t, &API{})
}
