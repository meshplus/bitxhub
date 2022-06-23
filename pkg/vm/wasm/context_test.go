package wasm

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/ledger"
	"github.com/meshplus/bitxhub/internal/ledger/mock_ledger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	hash          = "0x9f41dd84524bf8a42f8ab58ecfca6e1752d6fd93fe8dc00af4c71963c97db59f"
	formalAccount = "0x929545f44692178EDb7FA468B44c5351596184Ba"
)

func TestNewContextAndOtherMethod(t *testing.T) {
	ctr := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(ctr)
	stateLedger := mock_ledger.NewMockStateLedger(ctr)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	tx := &pb.BxhTransaction{
		From:      types.NewAddressByStr(formalAccount),
		To:        types.NewAddressByStr(formalAccount),
		Timestamp: 1567345493,
		Nonce:     0,
	}
	//bytes, _ := ioutil.ReadFile("./testdata/fabric_policy.wasm")
	txData := &pb.TransactionData{}
	logger := logrus.Logger{}
	context := NewContext(tx, txData, mockLedger, &logger)
	assert.NotNil(t, context)

	callee := context.Callee()
	assert.NotNil(t, callee)
	caller := context.Caller()
	assert.NotNil(t, caller)

	log := context.Logger()
	assert.NotNil(t, log)

}
