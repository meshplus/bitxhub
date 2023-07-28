package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	rpctypes "github.com/meshplus/bitxhub/api/jsonrpc/types"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/sirupsen/logrus"
)

// BitxhubAPI provides an API to get related info
type BitxhubAPI struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *repo.Config
	api    api.CoreAPI
	logger logrus.FieldLogger
}

func NewBitxhubAPI(config *repo.Config, api api.CoreAPI, logger logrus.FieldLogger) *BitxhubAPI {
	ctx, cancel := context.WithCancel(context.Background())
	return &BitxhubAPI{ctx: ctx, cancel: cancel, config: config, api: api, logger: logger}
}

// GasPrice returns the current gas price based on dynamic adjustment strategy.
func (api *BitxhubAPI) GasPrice() *hexutil.Big {
	api.logger.Debug("eth_gasPrice")
	gasPrice, err := api.api.Gas().GetGasPrice()
	if err != nil {
		api.logger.Errorf("get gas price err: %v", err)
	}
	out := big.NewInt(int64(gasPrice))
	return (*hexutil.Big)(out)
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic transactions.
// todo Supplementary gas fee
func (api *BitxhubAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	api.logger.Debug("eth_maxPriorityFeePerGas")
	return (*hexutil.Big)(new(big.Int)), nil
}

type feeHistoryResult struct {
	OldestBlock  rpctypes.BlockNumber `json:"oldestBlock"`
	Reward       [][]*hexutil.Big     `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big       `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64            `json:"gasUsedRatio"`
}

// FeeHistory return feeHistory
// todo Supplementary feeHsitory
func (api *BitxhubAPI) FeeHistory(blockCount rpctypes.DecimalOrHex, lastBlock rpctypes.BlockNumber, rewardPercentiles []float64) (*feeHistoryResult, error) {
	api.logger.Debug("eth_feeHistory")
	return nil, NotSupportApiError
}

// Syncing returns whether or not the current node is syncing with other peers. Returns false if not, or a struct
// outlining the state of the sync if it is.
func (api *BitxhubAPI) Syncing() (interface{}, error) {
	api.logger.Debug("eth_syncing")

	// TODO
	// Supplementary data
	syncBlock := make(map[string]string)
	meta, err := api.api.Chain().Meta()

	if err != nil {
		syncBlock["result"] = "false"
		return syncBlock, err
	}

	syncBlock["startingBlock"] = fmt.Sprintf("%d", hexutil.Uint64(1))
	syncBlock["highestBlock"] = fmt.Sprintf("%d", hexutil.Uint64(meta.Height))
	syncBlock["currentBlock"] = syncBlock["highestBlock"]
	return syncBlock, nil
}
