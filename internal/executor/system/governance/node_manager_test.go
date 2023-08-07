package governance

import (
	"path/filepath"
	"testing"

	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/executor/system/common"
	"github.com/axiomesh/axiom/internal/ledger"
	vm "github.com/axiomesh/eth-kit/evm"
	"github.com/axiomesh/eth-kit/ledger/mock_ledger"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNodeManager_Run(t *testing.T) {
	nm := NewNodeManager(logrus.New())

	mockCtl := gomock.NewController(t)
	stateLedger := mock_ledger.NewMockStateLedger(mockCtl)

	accountCache, err := ledger.NewAccountCache()
	assert.Nil(t, err)
	repoRoot := t.TempDir()
	ld, err := leveldb.New(filepath.Join(repoRoot, "node_manager"))
	assert.Nil(t, err)
	account := ledger.NewAccount(ld, accountCache, types.NewAddressByStr(common.NodeManagerContractAddr), ledger.NewChanger())

	stateLedger.EXPECT().GetOrCreateAccount(gomock.Any()).Return(account).AnyTimes()

	nm.Reset(stateLedger)

	gabi, err := GetABI()
	assert.Nil(t, err)

	data, err := gabi.Pack(ProposeMethod, "title", "desc", 1000, []byte(""))
	res, err := nm.Run(&vm.Message{
		Data: data,
	})
	assert.Nil(t, err)

	assert.Equal(t, uint64(0), res.UsedGas)
}
