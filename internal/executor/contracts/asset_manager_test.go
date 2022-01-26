package contracts

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/meshplus/bitxhub-kit/log"
	types2 "github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub/internal/executor/oracle/appchain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const pier = "0x56CcF2466a27E4D231824b823e04cD672E52002B"
const address = "0x3cd213723e81326c4783459f0cdf356833a4cf93"
const receiptJson = "{\"root\":\"0x\",\"status\":\"0x1\",\"cumulativeGasUsed\":\"0x154444\",\"logsBloom\":\"0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000824000000800000000000000000000000000000000000000020000000000000000000000000000000000080000000000000000001000000000000000000000000000000000000000000004040000000000004000000000000002000000000000000000000000000000000000000040000000000000000000000000000a000000000020010000000000000000000000000000000010000000000010000000000000000000000000004000000000000000000000002000000000\",\"logs\":[{\"address\":\"0x2862f68e270e7024776a6e10a4056d1f3eda67c6\",\"topics\":[\"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef\",\"0x0000000000000000000000002962b85e2bee2e1ea9c4cd69f2758cf7bbc3297e\",\"0x0000000000000000000000003cd213723e81326c4783459f0cdf356833a4cf93\"],\"data\":\"0x00000000000000000000000000000000000000000000021e19e0c9bab2400000\",\"blockNumber\":\"0xa2f324\",\"transactionHash\":\"0x5a5ba88ff023f5058921946f3708a3b9e6b5cd70b4f3e6cb348a48e8f02b3a7c\",\"transactionIndex\":\"0xe\",\"blockHash\":\"0x8e9b606d8a6d36d0a0f129b5e9df53407e13687b278e1ecf7429f1f284fcbed5\",\"logIndex\":\"0x18\",\"removed\":false},{\"address\":\"0x2862f68e270e7024776a6e10a4056d1f3eda67c6\",\"topics\":[\"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925\",\"0x0000000000000000000000002962b85e2bee2e1ea9c4cd69f2758cf7bbc3297e\",\"0x0000000000000000000000003cd213723e81326c4783459f0cdf356833a4cf93\"],\"data\":\"0xfffffffffffffffffffffffffffffffffffffffffffffde1e61f36454dbfffff\",\"blockNumber\":\"0xa2f324\",\"transactionHash\":\"0x5a5ba88ff023f5058921946f3708a3b9e6b5cd70b4f3e6cb348a48e8f02b3a7c\",\"transactionIndex\":\"0xe\",\"blockHash\":\"0x8e9b606d8a6d36d0a0f129b5e9df53407e13687b278e1ecf7429f1f284fcbed5\",\"logIndex\":\"0x19\",\"removed\":false},{\"address\":\"0x3cd213723e81326c4783459f0cdf356833a4cf93\",\"topics\":[\"0x67741de31257ee580484c64e9f8b91449aa7df22ae38fcb86e50bdadfca0ad23\"],\"data\":\"0x0000000000000000000000002862f68e270e7024776a6e10a4056d1f3eda67c60000000000000000000000008a413d6366c4a88caee1c4efe45a029dd87aebd20000000000000000000000002962b85e2bee2e1ea9c4cd69f2758cf7bbc3297e00000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000021e19e0c9bab2400000000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000002a30783239363262383565326245653265316541394334434436396632373538634637626263333239374500000000000000000000000000000000000000000000\",\"blockNumber\":\"0xa2f324\",\"transactionHash\":\"0x5a5ba88ff023f5058921946f3708a3b9e6b5cd70b4f3e6cb348a48e8f02b3a7c\",\"transactionIndex\":\"0xe\",\"blockHash\":\"0x8e9b606d8a6d36d0a0f129b5e9df53407e13687b278e1ecf7429f1f284fcbed5\",\"logIndex\":\"0x1a\",\"removed\":false}],\"transactionHash\":\"0x5a5ba88ff023f5058921946f3708a3b9e6b5cd70b4f3e6cb348a48e8f02b3a7c\",\"contractAddress\":\"0x0000000000000000000000000000000000000000\",\"gasUsed\":\"0x165cc\",\"blockHash\":\"0x8e9b606d8a6d36d0a0f129b5e9df53407e13687b278e1ecf7429f1f284fcbed5\",\"blockNumber\":\"0xa2f324\",\"transactionIndex\":\"0xe\"}\n"
const receiptJson1 = "{\"root\":\"0x\",\"status\":\"0x1\",\"cumulativeGasUsed\":\"0x44b934\",\"logsBloom\":\"0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000824000000800000000000000000000000000000000000001020000000000000000000000000000000000080000000000000000009000000000000000000000000000000000000000000004040000000000004000000000000002000000000000000000000000000000000000000040000000000000000000000000000a000002000020010000000000000000000000000000000010000000000010000000000000000000000000004000000000000000000000002000000000\",\"logs\":[{\"address\":\"0x2862f68e270e7024776a6e10a4056d1f3eda67c6\",\"topics\":[\"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef\",\"0x0000000000000000000000002962b85e2bee2e1ea9c4cd69f2758cf7bbc3297e\",\"0x0000000000000000000000003cd213723e81326c4783459f0cdf356833a4cf93\"],\"data\":\"0x000000000000000000000000000000000000000000000000002386f26fc10000\",\"blockNumber\":\"0xa2f382\",\"transactionHash\":\"0xa4eabcc1ebe1a3b082256db6909c561f2f229372c79b8305dcd7c078b48a1d54\",\"transactionIndex\":\"0x1b\",\"blockHash\":\"0x7c2c608de6948e86bd1172143adb8311c55dd23dc47b1772af3f5a36c308aacb\",\"logIndex\":\"0x20\",\"removed\":false},{\"address\":\"0x2862f68e270e7024776a6e10a4056d1f3eda67c6\",\"topics\":[\"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925\",\"0x0000000000000000000000002962b85e2bee2e1ea9c4cd69f2758cf7bbc3297e\",\"0x0000000000000000000000003cd213723e81326c4783459f0cdf356833a4cf93\"],\"data\":\"0xfffffffffffffffffffffffffffffffffffffffffffffde1e5fbaf52ddfeffff\",\"blockNumber\":\"0xa2f382\",\"transactionHash\":\"0xa4eabcc1ebe1a3b082256db6909c561f2f229372c79b8305dcd7c078b48a1d54\",\"transactionIndex\":\"0x1b\",\"blockHash\":\"0x7c2c608de6948e86bd1172143adb8311c55dd23dc47b1772af3f5a36c308aacb\",\"logIndex\":\"0x21\",\"removed\":false},{\"address\":\"0x3cd213723e81326c4783459f0cdf356833a4cf93\",\"topics\":[\"0x67741de31257ee580484c64e9f8b91449aa7df22ae38fcb86e50bdadfca0ad23\"],\"data\":\"0x0000000000000000000000002862f68e270e7024776a6e10a4056d1f3eda67c60000000000000000000000008a413d6366c4a88caee1c4efe45a029dd87aebd20000000000000000000000002962b85e2bee2e1ea9c4cd69f2758cf7bbc3297e00000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000002386f26fc10000000000000000000000000000000000000000000000000000000000000000000d000000000000000000000000000000000000000000000000000000000000002a30783239363262383565326245653265316541394334434436396632373538634637626263333239374500000000000000000000000000000000000000000000\",\"blockNumber\":\"0xa2f382\",\"transactionHash\":\"0xa4eabcc1ebe1a3b082256db6909c561f2f229372c79b8305dcd7c078b48a1d54\",\"transactionIndex\":\"0x1b\",\"blockHash\":\"0x7c2c608de6948e86bd1172143adb8311c55dd23dc47b1772af3f5a36c308aacb\",\"logIndex\":\"0x22\",\"removed\":false},{\"address\":\"0x3cd213723e81326c4783459f0cdf356833a4cf93\",\"topics\":[\"0xbf42a9b8a78d1a7612fe5abd3e8bb6a6d68d4a31cc3a0ead24cdad19450fab83\"],\"data\":\"0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000002a30784361396332443930333542353034324245323546653331354631343441453832373864446466376300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a30783933433241344536666337303132346563353943353632613232396639313938386436413238363500000000000000000000000000000000000000000000\",\"blockNumber\":\"0xa2f382\",\"transactionHash\":\"0xa4eabcc1ebe1a3b082256db6909c561f2f229372c79b8305dcd7c078b48a1d54\",\"transactionIndex\":\"0x1b\",\"blockHash\":\"0x7c2c608de6948e86bd1172143adb8311c55dd23dc47b1772af3f5a36c308aacb\",\"logIndex\":\"0x23\",\"removed\":false}],\"transactionHash\":\"0xa4eabcc1ebe1a3b082256db6909c561f2f229372c79b8305dcd7c078b48a1d54\",\"contractAddress\":\"0x0000000000000000000000000000000000000000\",\"gasUsed\":\"0x18241\",\"blockHash\":\"0x7c2c608de6948e86bd1172143adb8311c55dd23dc47b1772af3f5a36c308aacb\",\"blockNumber\":\"0xa2f382\",\"transactionIndex\":\"0x1b\"}\n"

func TestEthHeaderManager(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestRopstenLightClient")
	require.Nil(t, err)
	defer os.RemoveAll(repoRoot)

	oracle, err := appchain.NewRopstenOracle("../../../config/appchain/eth_header1.json", repoRoot, false, log.NewWithModule("test"))
	require.Nil(t, err)

	contractAddr := &ContractAddr{address}
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)
	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mockStub.EXPECT().GetObject(EscrowsAddrKey+pier, gomock.Any()).SetArg(1, *contractAddr).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(InterchainSwapAddrKey, gomock.Any()).SetArg(1, *contractAddr).Return(true)
	mockStub.EXPECT().GetObject(InterchainSwapAddrKey, gomock.Any()).SetArg(1, contractAddr).Return(true).AnyTimes()
	mockStub.EXPECT().GetObject(ProxyAddrKey, gomock.Any()).SetArg(1, *contractAddr).Return(true)
	mockStub.EXPECT().GetObject(ProxyAddrKey, gomock.Any()).SetArg(1, contractAddr).Return(true).AnyTimes()
	mockStub.EXPECT().Get(gomock.Any()).Return(false, []byte("")).AnyTimes()
	mockStub.EXPECT().Caller().Return(pier).AnyTimes()
	mockStub.EXPECT().Logger().Return(logrus.New()).AnyTimes()
	mockStub.EXPECT().GetTxHash().Return(&types2.Hash{}).AnyTimes()
	mockStub.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockStub.EXPECT().CrossInvoke(gomock.Any(), gomock.Any(), gomock.Any()).Return(boltvm.Success([]byte("true"))).AnyTimes()
	mockStub.EXPECT().CrossInvokeEVM(gomock.Any(), gomock.Any()).Return(&boltvm.Response{Ok: true}).Times(2)

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

	res = ehm.SetEscrowAddr(pier, address)
	require.True(t, res.Ok)

	res = ehm.GetEscrowAddr(pier)
	require.True(t, res.Ok)

	header := ehm.GetBlockHeader(header1.Hash().String())
	require.NotNil(t, header)

	res = ehm.SetInterchainSwapAddr(address)
	require.True(t, res.Ok)

	res = ehm.GetInterchainSwapAddr()
	require.True(t, res.Ok)

	res = ehm.SetProxyAddr(address)
	require.True(t, res.Ok)

	res = ehm.GetProxyAddr()
	require.True(t, res.Ok)

	res = ehm.Mint([]byte(receiptJson), nil)
	require.True(t, res.Ok)

	res = ehm.Mint([]byte((receiptJson1)), nil)
	require.True(t, res.Ok)

	mockStub.EXPECT().CrossInvokeEVM(gomock.Any(), gomock.Any()).Return(&boltvm.Response{Ok: false}).AnyTimes()
	res = ehm.Mint([]byte((receiptJson1)), nil)
	require.True(t, !res.Ok)

}
