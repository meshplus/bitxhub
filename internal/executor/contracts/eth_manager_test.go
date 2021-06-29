package contracts

import (
	"encoding/json"
	"github.com/meshplus/bitxhub-core/boltvm"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub/internal/executor/oracle/appchain"
	"github.com/stretchr/testify/require"
)

const address = "0x5B38DA6A701C568545DCFCB03FCB875F56BEDDC4"

func TestEthHeaderManager_PreMint(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestRopstenLightClient")
	require.Nil(t, err)
	defer os.RemoveAll(repoRoot)

	oracle, err := appchain.NewRopstenOracle("../../../config/appchain/eth_header1.json", repoRoot, false, log.NewWithModule("test"))
	require.Nil(t, err)
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	mockStub.EXPECT().Caller().Return(address).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(),gomock.Any(), gomock.Any()).Return(boltvm.Success([]byte("true"))).AnyTimes()

	ehm := NewEthHeaderManager(oracle)
	ehm.Stub = mockStub

	res := ehm.CurrentBlockHeader()
	require.True(t, res.Ok)


	header1 := types.Header{}
	err = header1.UnmarshalJSON([]byte(appchain.RopstenHeader1))
	require.Nil(t, err)

	header2 := types.Header{}
	err = header2.UnmarshalJSON([]byte(appchain.RopstenHeader2))
	require.Nil(t, err)

	headers := []types.Header{header1, header2}
	headersData, err := json.Marshal(headers)
	require.Nil(t, err)

	res = ehm.InsertBlockHeaders(headersData)
	require.True(t, res.Ok)
	num, err := strconv.Atoi(string(res.Result))
	require.Nil(t, err)
	require.Equal(t, 0, num)

	res = ehm.SetEscrowAddr(address)
	require.True(t, res.Ok)

	res = ehm.GetEscrowAddr()
	require.True(t, res.Ok)

	res = ehm.SetInterchainSwapAddr(address)
	require.True(t, res.Ok)

	res = ehm.GetInterchainSwapAddr()
	require.True(t, res.Ok)
}

