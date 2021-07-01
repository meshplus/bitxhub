package appchain

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/sirupsen/logrus"
)

type EthLightChainOracle struct {
	lc     *light.LightChain
	logger logrus.FieldLogger
}

const (
	// block 10105112
	RopstenHeader = "{\"parentHash\":\"0x4672d904ca88bdb365f83bc6050344fdcb672ce8e639e727f8c69247634e73f0\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x1cdb00d07b721b98da52532db9a7d82d2a4bf2e0\",\"stateRoot\":\"0x823fbc8d2cf2d9c32832c6c1bd451ea84acdd79e39e88a7061215746a0794f40\",\"transactionsRoot\":\"0x65abe72faaaa113cf0a9374cefde93bc64cd264e2931ba95ee27c8d62499e63d\",\"receiptsRoot\":\"0xfac424a8e1a45789a5456fe1bc58228d93c28871df42dd789434e4de65f7e2e9\",\"logsBloom\":\"0x00200000000000000000000080000000000000000000000080010000000000000000000000000000020000000000000000000000000000000000000000200000000000000008000000000008000000200000000800000000000000020002000000000000080000000000000000000000000000000000000000000010000000000800000000000000014000000000000000000000000000080000004000000000020000000000000140000000000000000000001000000000001000000000000000000002000000200004000000002200000000000000003000000000020020000010000000000000000000000201000000000080000000000000000400000000\",\"difficulty\":\"0x729df1d\",\"number\":\"0x9a3118\",\"gasLimit\":\"0x98f36f\",\"gasUsed\":\"0x2fd96\",\"timestamp\":\"0x608506bf\",\"extraData\":\"0xd683010a01846765746886676f312e3136856c696e7578\",\"mixHash\":\"0x62c65e608f10001004171345c230a231afed11dad637b93545f600b71adb0f5d\",\"nonce\":\"0x7ea6f1aedb5ea8c1\",\"hash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\"}\n"
)

// block headers from infura api server
const (
	RinkebyHeader1 = "{\"parentHash\":\"0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x0000000000000000000000000000000000000000\",\"stateRoot\":\"0x53580584816f617295ea26c0e17641e0120cab2f0a8ffb53a866fd53aa8e8c2d\",\"transactionsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"receiptsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"difficulty\":\"0x2\",\"number\":\"0x1\",\"gasLimit\":\"0x47c94c\",\"gasUsed\":\"0x0\",\"timestamp\":\"0x58ee45da\",\"extraData\":\"0xd783010600846765746887676f312e372e33856c696e757800000000000000009f1efa1efa72af138c915966c639544a0255e6288e188c22ce9168c10dbe46da3d88b4aa065930119fb886210bf01a084fde5d3bc48d8aa38bca92e4fcc5215100\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":\"0x0000000000000000\",\"hash\":\"0xa7684ac44d48494670b2e0d9085b7750e7341620f0a271db146ed5e70c1db854\"}\n"
	RinkebyHeader2 = "{\"parentHash\":\"0xa7684ac44d48494670b2e0d9085b7750e7341620f0a271db146ed5e70c1db854\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x0000000000000000000000000000000000000000\",\"stateRoot\":\"0x53580584816f617295ea26c0e17641e0120cab2f0a8ffb53a866fd53aa8e8c2d\",\"transactionsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"receiptsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"difficulty\":\"0x2\",\"number\":\"0x2\",\"gasLimit\":\"0x47db3d\",\"gasUsed\":\"0x0\",\"timestamp\":\"0x58ee45ea\",\"extraData\":\"0xd783010600846765746887676f312e372e33856c696e75780000000000000000b5a4a624d2e19fdab62ff7f4d2f2b80dfab4c518761beb56c2319c4224dd156f698bb1a2750c7edf12d61c4022079622062039637f40fb817e2cce0f0a4dae9c01\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":\"0x0000000000000000\",\"hash\":\"0x9b095b36c15eaf13044373aef8ee0bd3a382a5abb92e402afa44b8249c3a90e9\"}\n"
	RinkebyHeader3 = "{\"parentHash\":\"0x9b095b36c15eaf13044373aef8ee0bd3a382a5abb92e402afa44b8249c3a90e9\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x0000000000000000000000000000000000000000\",\"stateRoot\":\"0x53580584816f617295ea26c0e17641e0120cab2f0a8ffb53a866fd53aa8e8c2d\",\"transactionsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"receiptsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"difficulty\":\"0x2\",\"number\":\"0x3\",\"gasLimit\":\"0x47e7c4\",\"gasUsed\":\"0x0\",\"timestamp\":\"0x58ee45f9\",\"extraData\":\"0xd783010600846765746887676f312e372e33856c696e757800000000000000004e10f96536e45ceca7e34cc1bdda71db3f3bb029eb69afd28b57eb0202c0ec0859d383a99f63503c4df9ab6c1dc63bf6b9db77be952f47d86d2d7b208e77397301\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":\"0x0000000000000000\",\"hash\":\"0x9eb9db9c3ec72918c7db73ae44e520139e95319c421ed6f9fc11fa8dd0cddc56\"}\n"

	RopstenHeader1 = "{\"parentHash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x033ef6db9fbd0ee60e2931906b987fe0280471a0\",\"stateRoot\":\"0xfc22ae06eb79345da19322573c740ccaf4e224e32c916226c0adf7223c881bc5\",\"transactionsRoot\":\"0x792fa9b9b95e586cd5757922f5f06adff5333bd5fb77059d95debcde4a4d5e0d\",\"receiptsRoot\":\"0x23f3951c0cede08fa61c20c77f36c8d7853d0f205f5053ee9a39f4cf68833546\",\"logsBloom\":\"0x00004000000000200000400000008000000010000000020000000000001004000000001000010000000000000000001040100000001000000010091010200000000000000000000000000008000004000000000000108400000100040000000000100000020000080000000020001880000200800000002000000010000018000000000240000000000000008000000000000020000000102020000000800000420004000000000000000000000020000000000000000014000200200000000000000102000000040000080000100000000060000000008000000200000020000010000000000000000000040000080004000000000000000000000000000000\",\"difficulty\":\"0x72ac658\",\"number\":\"0x9a3119\",\"gasLimit\":\"0x98cd34\",\"gasUsed\":\"0x1d58f8\",\"timestamp\":\"0x608506c2\",\"extraData\":\"0xd683010a01846765746886676f312e3136856c696e7578\",\"mixHash\":\"0xdbca2ed1c216550ef4d7f63d1c223da0530fc196f14960e70ac8f384cd0f8d63\",\"nonce\":\"0x2293d516a92c2912\",\"hash\":\"0x6a04391a825d6a5a7a247c6e5134f020abec8f8073b1eb1fc778bd9e1502b852\"}\n"
	RopstenHeader2 = "{\"parentHash\":\"0x6a04391a825d6a5a7a247c6e5134f020abec8f8073b1eb1fc778bd9e1502b852\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x773b556e7f3222f3b93d6519484cef0c443b9d7e\",\"stateRoot\":\"0xc45bd719ef83edd7f6afbaa7afb43a3422ef9ca0faf6853565dbd04671a0736f\",\"transactionsRoot\":\"0xf2a97635c98b8c96c6977e5e16418f9b480c0bd6f4922d15a34dffac2a2b23de\",\"receiptsRoot\":\"0x400c370ac70c3031022c8fd923fbff191f5dcfa6b4454e90ef8c5de8d9475af3\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000020000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000\",\"difficulty\":\"0x72ac858\",\"number\":\"0x9a311a\",\"gasLimit\":\"0x98a702\",\"gasUsed\":\"0xd09e6\",\"timestamp\":\"0x608506cb\",\"extraData\":\"0xd683010a01846765746886676f312e3136856c696e7578\",\"mixHash\":\"0xc7d603705499308d968e7e4687325bc227fbe104d62f0bd53eeff154e2749223\",\"nonce\":\"0x3733ef9174a63f69\",\"hash\":\"0x91c4c53daae0fc7fec3310c2c60597349f84183322b1bdaec9faf9a602627151\"}\n"
)

var MinConfirmNum uint64 = 15

// TODO: need to start with special header height
func NewRinkebyOracle(storagePath string, logger logrus.FieldLogger) (*EthLightChainOracle, error) {
	appchainBlockHeaderPath := filepath.Join(storagePath, "eth_rinkeby")
	db, err := leveldb.New(appchainBlockHeaderPath, 256, 0, "", false)
	if err != nil {
		return nil, err
	}
	database := rawdb.NewDatabase(db)
	core.DefaultRinkebyGenesisBlock().MustCommit(database)
	lc, err := light.NewLightChain(les.NewLesOdr(database, light.DefaultServerIndexerConfig, nil, nil),
		core.DefaultRinkebyGenesisBlock().Config, clique.New(params.RinkebyChainConfig.Clique, database), params.RinkebyTrustedCheckpoint)
	if err != nil {
		return nil, err
	}
	return &EthLightChainOracle{lc: lc, logger: logger}, nil
}

// NewRopstenOracle inits with ropsten block 10105112, receives above the 10105112 headers
func NewRopstenOracle(ropstenPath string, storagePath string, readOnly bool, logger logrus.FieldLogger) (*EthLightChainOracle, error) {
	db, err := leveldb.New(storagePath, 16, 0, "", readOnly)
	if err != nil {
		return nil, err
	}
	database := rawdb.NewDatabase(db)

	headerData, err := ioutil.ReadFile(ropstenPath)
	if err != nil {
		return nil, err
	}
	header := types.Header{}
	err = header.UnmarshalJSON(headerData)
	if err != nil {
		return nil, err
	}
	if head := rawdb.ReadHeadHeaderHash(database); head == (common.Hash{}) {
		core.DefaultRopstenGenesisBlock().MustCommit(database)
		rawdb.WriteHeader(database, &header)
		rawdb.WriteTd(database, header.Hash(), header.Number.Uint64(), header.Difficulty)
		rawdb.WriteCanonicalHash(database, header.Hash(), header.Number.Uint64())
		rawdb.WriteHeadBlockHash(database, header.Hash())
		rawdb.WriteHeadFastBlockHash(database, header.Hash())
		rawdb.WriteHeadHeaderHash(database, header.Hash())
	}

	lc, err := light.NewLightChain(les.NewLesOdr(database, light.DefaultServerIndexerConfig, nil, nil),
		core.DefaultRopstenGenesisBlock().Config, ethash.New(ethash.Config{}, nil, false), params.RopstenTrustedCheckpoint)
	if err != nil {
		return nil, fmt.Errorf("new light client error:%v", err)
	}

	return &EthLightChainOracle{lc: lc, logger: logger}, nil
}

// InsertBlockHeaders attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
// Ropsten receives the block header after the height of 10105112
func (oracle *EthLightChainOracle) InsertBlockHeaders(headers []*types.Header) (int, error) {
	if len(headers) == 0 {
		return 0, fmt.Errorf("insert empty headers")
	}
	sort.Slice(headers, func(i, j int) bool {
		return headers[i].Number.Cmp(headers[j].Number) < 0
	})

	oracle.logger.WithFields(logrus.Fields{
		"start": headers[0].Number.Uint64(),
		"end":   headers[len(headers)-1].Number.Uint64(),
	}).Debugf("insert ethereum block headers")
	return oracle.lc.InsertHeaderChain(headers, 0)
}

// CurrentHeader retrieves the current head header of the canonical chain.
func (oracle *EthLightChainOracle) CurrentHeader() *types.Header {
	return oracle.lc.CurrentHeader()
}

// GetHeader retrieves a block header by hash
func (oracle *EthLightChainOracle) GetHeader(hash common.Hash) *types.Header {
	return oracle.lc.GetHeaderByHash(hash)
}

func (oracle *EthLightChainOracle) VerifyProof(receipt *types.Receipt, proof []byte) error {
	if receipt.Status == 0 {
		return fmt.Errorf("receipt status is fail, hash is:%v", receipt.TxHash.String())
	}
	header := oracle.GetHeader(receipt.BlockHash)
	if header == nil {
		return fmt.Errorf("not found header:%v", receipt.BlockHash.String())
	}
	currentHeader := oracle.CurrentHeader()
	if currentHeader.Number.Uint64()-header.Number.Uint64() < MinConfirmNum {
		return fmt.Errorf("not enough confirmed")
	}

	keyBuf := bytes.Buffer{}
	keyBuf.Reset()
	if err := rlp.Encode(&keyBuf, receipt.TransactionIndex); err != nil {
		return err
	}
	nodeList := &light.NodeList{}
	if err := rlp.DecodeBytes(proof, nodeList); err != nil {
		return err
	}
	value, err := trie.VerifyProof(header.ReceiptHash, keyBuf.Bytes(), nodeList.NodeSet())
	if err != nil {
		return err
	}
	receiptData, err := rlp.EncodeToBytes(receipt)
	if err != nil {
		return err
	}
	if !bytes.Equal(receiptData, value) {
		return fmt.Errorf("invaild receipt:%v", receipt.TxHash.String())
	}
	return nil
}
