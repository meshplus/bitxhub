package appchain

import (
	"bytes"
	"fmt"
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
	MinConfirmNum = 15
	// block 10105112
	RopstenHeader = "{\"parentHash\":\"0x4672d904ca88bdb365f83bc6050344fdcb672ce8e639e727f8c69247634e73f0\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x1cdb00d07b721b98da52532db9a7d82d2a4bf2e0\",\"stateRoot\":\"0x823fbc8d2cf2d9c32832c6c1bd451ea84acdd79e39e88a7061215746a0794f40\",\"transactionsRoot\":\"0x65abe72faaaa113cf0a9374cefde93bc64cd264e2931ba95ee27c8d62499e63d\",\"receiptsRoot\":\"0xfac424a8e1a45789a5456fe1bc58228d93c28871df42dd789434e4de65f7e2e9\",\"logsBloom\":\"0x00200000000000000000000080000000000000000000000080010000000000000000000000000000020000000000000000000000000000000000000000200000000000000008000000000008000000200000000800000000000000020002000000000000080000000000000000000000000000000000000000000010000000000800000000000000014000000000000000000000000000080000004000000000020000000000000140000000000000000000001000000000001000000000000000000002000000200004000000002200000000000000003000000000020020000010000000000000000000000201000000000080000000000000000400000000\",\"difficulty\":\"0x729df1d\",\"number\":\"0x9a3118\",\"gasLimit\":\"0x98f36f\",\"gasUsed\":\"0x2fd96\",\"timestamp\":\"0x608506bf\",\"extraData\":\"0xd683010a01846765746886676f312e3136856c696e7578\",\"mixHash\":\"0x62c65e608f10001004171345c230a231afed11dad637b93545f600b71adb0f5d\",\"nonce\":\"0x7ea6f1aedb5ea8c1\",\"hash\":\"0x0d37ff8f4a8f1adcfd16add9cf8726e17a8097baa9a50d3b5fd51849476f7ec3\"}\n"
)

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
func NewRopstenOracle(storagePath string, logger logrus.FieldLogger) (*EthLightChainOracle, error) {
	appchainBlockHeaderPath := filepath.Join(storagePath, "eth_ropsten")
	db, err := leveldb.New(appchainBlockHeaderPath, 256, 0, "", false)
	if err != nil {
		return nil, err
	}
	database := rawdb.NewDatabase(db)

	// block 10105112
	header := types.Header{}
	err = header.UnmarshalJSON([]byte(RopstenHeader))

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
		return nil, err
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
