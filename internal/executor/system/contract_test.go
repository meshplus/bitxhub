package system

import (
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/ledger"
	"github.com/axiomesh/axiom/internal/ledger/mock_ledger"
	"github.com/axiomesh/axiom/pkg/repo"
)

var systemContractAddrs = []string{
	common.NodeManagerContractAddr,
}

var notSystemContractAddrs = []string{
	"0x1000000000000000000000000000000000000000",
	"0x0340000000000000000000000000000000000000",
	"0x0200000000000000000000000000000000000000",
	"0xffddd00000000000000000000000000000000000",
}

func TestContract_GetSystemContract(t *testing.T) {
	Initialize(logrus.New())

	for _, addr := range systemContractAddrs {
		contract, ok := GetSystemContract(types.NewAddressByStr(addr))
		assert.True(t, ok)
		assert.NotNil(t, contract)
	}

	for _, addr := range notSystemContractAddrs {
		contract, ok := GetSystemContract(types.NewAddressByStr(addr))
		assert.False(t, ok)
		assert.Nil(t, contract)
	}

	// test nil address
	contract, ok := GetSystemContract(nil)
	assert.False(t, ok)
	assert.Nil(t, contract)

	// test empty address
	contract, ok = GetSystemContract(&types.Address{})
	assert.False(t, ok)
	assert.Nil(t, contract)
}

func TestContractInitGenesisData(t *testing.T) {
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	repoRoot := t.TempDir()
	genesis := repo.DefaultConfig(repoRoot, false)
	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	ld, err := leveldb.New(filepath.Join(repoRoot, "executor"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())
	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()

	err = InitGenesisData(&genesis.Genesis, mockLedger.StateLedger)
	assert.Nil(t, err)
}

func TestContractCheckAndUpdateAllState(t *testing.T) {
	mockCtl := gomock.NewController(t)
	chainLedger := mock_ledger.NewMockChainLedger(mockCtl)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	mockLedger := &ledger.Ledger{
		ChainLedger: chainLedger,
		StateLedger: stateLedger,
	}

	repoRoot := t.TempDir()
	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	ld, err := leveldb.New(filepath.Join(repoRoot, "executor"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())
	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()
	stateLedger.EXPECT().SetBalance(gomock.Any(), gomock.Any()).AnyTimes()

	CheckAndUpdateAllState(1, mockLedger.StateLedger)
	assert.Nil(t, err)
	exist, _ := account.Query("")
	assert.False(t, exist)
}
