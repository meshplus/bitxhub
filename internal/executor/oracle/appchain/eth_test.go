package appchain

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/stretchr/testify/require"
)

func TestRinkebyLightClient(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestRinkebyLightClient")
	require.Nil(t, err)
	defer os.RemoveAll(repoRoot)

	oracle, err := NewRinkebyOracle(repoRoot, log.NewWithModule("test"))
	require.Nil(t, err)

	header1 := types.Header{}
	err = header1.UnmarshalJSON([]byte(RinkebyHeader1))
	require.Nil(t, err)

	header2 := types.Header{}
	err = header2.UnmarshalJSON([]byte(RinkebyHeader2))
	require.Nil(t, err)

	header3 := types.Header{}
	err = header3.UnmarshalJSON([]byte(RinkebyHeader3))
	require.Nil(t, err)

	num, err := oracle.InsertBlockHeaders([]*types.Header{&header1, &header2})
	require.Nil(t, err)
	require.Equal(t, num, 0)

}

func TestRopstenLightClient(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestRopstenLightClient")
	require.Nil(t, err)
	defer os.RemoveAll(repoRoot)

	oracle, err := NewRopstenOracle("../../../../config/appchain/eth_header1.json", repoRoot, false, log.NewWithModule("test"))
	require.Nil(t, err)

	header1 := &types.Header{}
	err = header1.UnmarshalJSON([]byte(RopstenHeader1))
	require.Nil(t, err)

	header2 := &types.Header{}
	err = header2.UnmarshalJSON([]byte(RopstenHeader2))
	require.Nil(t, err)

	num, err := oracle.InsertBlockHeaders([]*types.Header{header1, header2})
	require.Nil(t, err)
	require.Equal(t, 0, num)

}

func TestVerifyProof(t *testing.T) {
	repoRoot, err := ioutil.TempDir("", "TestRopstenLightClient")
	require.Nil(t, err)
	defer os.RemoveAll(repoRoot)

	var (
		receipt0 = "{\"root\":\"0x\",\"status\":\"0x1\",\"cumulativeGasUsed\":\"0x11336\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000008000000000008000000000000000000000000000000020002000000000000000000000000000000000000000000000000000000000010000000000800000000000000010000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000002000000000004000000000200000000000000002000000000000000000010000000000000000000000001000000000000000000000000000000000000\",\"logs\":[{\"address\":\"0x7a9a60a43edd1d885f9e672dd498c829b101dd07\",\"topics\":[\"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef\",\"0x0000000000000000000000002266463a8b23ddfc606670a06b0ce2e66011f1bf\",\"0x0000000000000000000000000de52d48482bf7f8a4c94bfdbdedd2319e99e92f\"],\"data\":\"0x00000000000000000000000000000000000000000000043c33c1937564800000\",\"blockNumber\":\"0x9a3118\",\"transactionHash\":\"0xba4d94730e0e76992538f19597cc41669d154c103b2ad77fb53ce18c4c0fc3cd\",\"transactionIndex\":\"0x0\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"logIndex\":\"0x0\",\"removed\":false},{\"address\":\"0x7a9a60a43edd1d885f9e672dd498c829b101dd07\",\"topics\":[\"0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925\",\"0x0000000000000000000000002266463a8b23ddfc606670a06b0ce2e66011f1bf\",\"0x0000000000000000000000000de52d48482bf7f8a4c94bfdbdedd2319e99e92f\"],\"data\":\"0xfffffffffffffffffffffffffffffffffffffffffffffbc3cc3e6c8a9b7fffff\",\"blockNumber\":\"0x9a3118\",\"transactionHash\":\"0xba4d94730e0e76992538f19597cc41669d154c103b2ad77fb53ce18c4c0fc3cd\",\"transactionIndex\":\"0x0\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"logIndex\":\"0x1\",\"removed\":false}],\"transactionHash\":\"0xba4d94730e0e76992538f19597cc41669d154c103b2ad77fb53ce18c4c0fc3cd\",\"contractAddress\":\"0x0000000000000000000000000000000000000000\",\"gasUsed\":\"0x11336\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"blockNumber\":\"0x9a3118\",\"transactionIndex\":\"0x0\"}\n"
		receipt1 = "{\"root\":\"0x\",\"status\":\"0x1\",\"cumulativeGasUsed\":\"0x1653e\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"logs\":[],\"transactionHash\":\"0xcb158ce1be4b6648dcdc69dedbe7ddda8e3aa284db7dacdb5b22401e9a454845\",\"contractAddress\":\"0x0000000000000000000000000000000000000000\",\"gasUsed\":\"0x5208\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"blockNumber\":\"0x9a3118\",\"transactionIndex\":\"0x1\"}\n"
		receipt2 = "{\"root\":\"0x\",\"status\":\"0x1\",\"cumulativeGasUsed\":\"0x2fd96\",\"logsBloom\":\"0x00200000000000000000000080000000000000000000000080010000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000008000000200000000800000000000000000000000000000000080000000000000000000000000000000000000000000010000000000000000000000000004000000000000000000000000000080000004000000000000000000000000140000000000000000000001000000000001000000000000000000002000000200000000000002000000000000000001000000000020020000000000000000000000000000200000000000080000000000000000400000000\",\"logs\":[{\"address\":\"0x2d80502854fc7304c3e3457084de549f5016b73f\",\"topics\":[\"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef\",\"0x0000000000000000000000004183d62963434056e75e9854bc4ba92aa43a2d08\",\"0x000000000000000000000000dc914190feeb16d6f7c5d9a22826d515be5c5857\"],\"data\":\"0x00000000000000000000000000000000000000000000000000000000000b4cc6\",\"blockNumber\":\"0x9a3118\",\"transactionHash\":\"0x37fc9bf0e945443c862efe5405d9a179dced048c3baed58751f00d84ba64701c\",\"transactionIndex\":\"0x2\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"logIndex\":\"0x2\",\"removed\":false},{\"address\":\"0x0d9c8723b343a8368bebe0b5e89273ff8d712e3c\",\"topics\":[\"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef\",\"0x000000000000000000000000dc914190feeb16d6f7c5d9a22826d515be5c5857\",\"0x0000000000000000000000004183d62963434056e75e9854bc4ba92aa43a2d08\"],\"data\":\"0x0000000000000000000000000000000000000000000000000000000015ee0375\",\"blockNumber\":\"0x9a3118\",\"transactionHash\":\"0x37fc9bf0e945443c862efe5405d9a179dced048c3baed58751f00d84ba64701c\",\"transactionIndex\":\"0x2\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"logIndex\":\"0x3\",\"removed\":false},{\"address\":\"0xdc914190feeb16d6f7c5d9a22826d515be5c5857\",\"topics\":[\"0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1\"],\"data\":\"0x00000000000000000000000000000000000000000000000000001a4f9e78bc5e0000000000000000000000000000000000000000000000000000000d8457df78\",\"blockNumber\":\"0x9a3118\",\"transactionHash\":\"0x37fc9bf0e945443c862efe5405d9a179dced048c3baed58751f00d84ba64701c\",\"transactionIndex\":\"0x2\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"logIndex\":\"0x4\",\"removed\":false},{\"address\":\"0xdc914190feeb16d6f7c5d9a22826d515be5c5857\",\"topics\":[\"0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822\",\"0x0000000000000000000000007a250d5630b4cf539739df2c5dacb4c659f2488d\",\"0x0000000000000000000000004183d62963434056e75e9854bc4ba92aa43a2d08\"],\"data\":\"0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000b4cc60000000000000000000000000000000000000000000000000000000015ee03750000000000000000000000000000000000000000000000000000000000000000\",\"blockNumber\":\"0x9a3118\",\"transactionHash\":\"0x37fc9bf0e945443c862efe5405d9a179dced048c3baed58751f00d84ba64701c\",\"transactionIndex\":\"0x2\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"logIndex\":\"0x5\",\"removed\":false}],\"transactionHash\":\"0x37fc9bf0e945443c862efe5405d9a179dced048c3baed58751f00d84ba64701c\",\"contractAddress\":\"0x0000000000000000000000000000000000000000\",\"gasUsed\":\"0x19858\",\"blockHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"blockNumber\":\"0x9a3118\",\"transactionIndex\":\"0x2\"}\n"
	)
	receiptList := []string{receipt0, receipt1, receipt2}
	oracle, err := NewRopstenOracle("../../../../config/appchain/eth_header1.json", repoRoot, false, log.NewWithModule("test"))
	require.Nil(t, err)

	receipts := make([]*types.Receipt, 0, len(receiptList))
	for _, r := range receiptList {
		var receipt types.Receipt
		err := receipt.UnmarshalJSON([]byte(r))
		require.Nil(t, err)
		receipts = append(receipts, &receipt)
	}
	tReceipts := types.Receipts(receipts)

	keybuf := new(bytes.Buffer)
	receiptsTrie := new(trie.Trie)
	index := uint64(2)
	header := oracle.lc.GetHeaderByHash(receipts[index].BlockHash)

	deriveSha := types.DeriveSha(tReceipts, receiptsTrie)
	require.Equal(t, header.ReceiptHash, deriveSha)

	err = rlp.Encode(keybuf, index)
	require.Nil(t, err)
	nodeSet := light.NewNodeSet()
	err = receiptsTrie.Prove(keybuf.Bytes(), 0, nodeSet)
	require.Nil(t, err)

	proof, err := rlp.EncodeToBytes(nodeSet.NodeList())
	require.Nil(t, err)

	MinConfirmNum = 0
	err = oracle.VerifyProof(receipts[index], proof)
	require.Nil(t, err)
}
