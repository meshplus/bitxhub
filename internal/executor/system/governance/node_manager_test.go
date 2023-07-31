package governance

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub/internal/ledger"
	vm "github.com/meshplus/eth-kit/evm"
	"github.com/meshplus/eth-kit/ledger/mock_ledger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNodeManager_Run(t *testing.T) {
	nm := NewNodeManager(logrus.New())

	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}
	nm.Reset(mockLedger)

	gabi, err := GetABI()
	assert.Nil(t, err)

	data, err := gabi.Pack(ProposalMethod, "title", "desc", 1000, []byte(""))
	res, err := nm.Run(&vm.Message{
		Data: data,
	})
	assert.Nil(t, err)

	assert.Equal(t, uint64(0), res.UsedGas)
}