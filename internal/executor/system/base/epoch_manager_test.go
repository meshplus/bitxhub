package base

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
	"github.com/axiomesh/axiom/pkg/repo"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
)

func TestEpochManager(t *testing.T) {
	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)
	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	ld, err := leveldb.New(filepath.Join(repoRoot, "epoch_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.EpochManagerContractAddr), ledger.NewChanger())
	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()

	epochMgr := NewEpochManager(&common.SystemContractConfig{
		Logger: logrus.New(),
	})
	epochMgr.Reset(stateLedger)
	_, err = epochMgr.EstimateGas(nil)
	assert.Error(t, err)
	_, err = epochMgr.Run(nil)
	assert.Error(t, err)
	epochMgr.CheckAndUpdateState(0, stateLedger)

	g := repo.GenesisEpochInfo()
	g.EpochPeriod = 100
	g.StartBlock = 1
	err = InitEpochInfo(stateLedger, g)
	assert.Nil(t, err)

	currentEpoch, err := GetCurrentEpochInfo(stateLedger)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, currentEpoch.Epoch)
	assert.EqualValues(t, 1, currentEpoch.StartBlock)

	nextEpoch, err := GetNextEpochInfo(stateLedger)
	assert.Nil(t, err)
	assert.EqualValues(t, 2, nextEpoch.Epoch)
	assert.EqualValues(t, 101, nextEpoch.StartBlock)

	epoch1, err := GetEpochInfo(stateLedger, 1)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, epoch1.Epoch)

	_, err = GetEpochInfo(stateLedger, 2)
	assert.Error(t, err)

	newCurrentEpoch, err := TurnIntoNewEpoch(stateLedger)
	assert.Nil(t, err)
	assert.EqualValues(t, 2, newCurrentEpoch.Epoch)
	assert.EqualValues(t, 101, newCurrentEpoch.StartBlock)

	currentEpoch, err = GetCurrentEpochInfo(stateLedger)
	assert.Nil(t, err)
	assert.EqualValues(t, 2, currentEpoch.Epoch)
	assert.EqualValues(t, 101, currentEpoch.StartBlock)

	nextEpoch, err = GetNextEpochInfo(stateLedger)
	assert.Nil(t, err)
	assert.EqualValues(t, 3, nextEpoch.Epoch)
	assert.EqualValues(t, 201, nextEpoch.StartBlock)

	epoch2, err := GetEpochInfo(stateLedger, 2)
	assert.Nil(t, err)
	assert.EqualValues(t, 2, epoch2.Epoch)

	_, err = GetEpochInfo(stateLedger, 3)
	assert.Error(t, err)
}
