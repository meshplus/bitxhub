package types

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BlockNumberOrHash supports querying data through block height or block hash
type BlockNumberOrHash struct {
	BlockNumber *BlockNumber `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash `json:"blockHash,omitempty"`
}

func (bnh *BlockNumberOrHash) UnmarshalJSON(data []byte) error {
	type erased BlockNumberOrHash
	e := erased{}
	err := json.Unmarshal(data, &e)
	if err == nil {
		if e.BlockNumber == nil && e.BlockHash != nil {
			return fmt.Errorf("only support blockNumber")
		}
		if e.BlockNumber != nil && *e.BlockNumber != LatestBlockNumber && *e.BlockNumber != PendingBlockNumber {
			return fmt.Errorf("only support latest and pending")
		}
		bnh.BlockNumber = e.BlockNumber

		/*if e.BlockNumber != nil && e.BlockHash != nil {
			return fmt.Errorf("cannot specify both BlockHash and BlockNumber, choose one or the other")
		}
		bnh.BlockNumber = e.BlockNumber
		bnh.BlockHash = e.BlockHash*/
		return nil
	}
	var input string
	err = json.Unmarshal(data, &input)
	if err != nil {
		return err
	}
	switch input {
	// case "earliest":
	// 	bn := EarliestBlockNumber
	// 	bnh.BlockNumber = &bn
	// 	return nil
	case "latest":
		bn := LatestBlockNumber
		bnh.BlockNumber = &bn
		return nil
	case "pending":
		bn := PendingBlockNumber
		bnh.BlockNumber = &bn
		return nil
	default:
		//todo Default to use the LatestBlockNumber
		//	   Improved after modifying the accounting module
		bn := LatestBlockNumber
		bnh.BlockNumber = &bn
		return nil

		// if len(input) == 66 {
		// 	hash := common.Hash{}
		// 	err := hash.UnmarshalText([]byte(input))
		// 	if err != nil {
		// 		return err
		// 	}
		// 	bnh.BlockHash = &hash
		// 	return nil
		// } else {
		// 	blckNum, err := hexutil.DecodeUint64(input)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	if blckNum > math.MaxInt64 {
		// 		return fmt.Errorf("blocknumber too high")
		// 	}
		// 	bn := BlockNumber(blckNum)
		// 	bnh.BlockNumber = &bn
		// 	return nil
		// }
	}
}

func (bnh *BlockNumberOrHash) Number() (BlockNumber, bool) {
	if bnh.BlockNumber != nil {
		return *bnh.BlockNumber, true
	}
	return BlockNumber(0), false
}

func (bnh *BlockNumberOrHash) String() string {
	if bnh.BlockNumber != nil {
		return strconv.Itoa(int(*bnh.BlockNumber))
	}
	if bnh.BlockHash != nil {
		return bnh.BlockHash.String()
	}
	return "nil"
}

func (bnh *BlockNumberOrHash) Hash() (common.Hash, bool) {
	if bnh.BlockHash != nil {
		return *bnh.BlockHash, true
	}
	return common.Hash{}, false
}

func BlockNumberOrHashWithNumber(blockNr BlockNumber) BlockNumberOrHash {
	return BlockNumberOrHash{
		BlockNumber: &blockNr,
		BlockHash:   nil,
	}
}

func BlockNumberOrHashWithHash(hash common.Hash, canonical bool) BlockNumberOrHash {
	return BlockNumberOrHash{
		BlockNumber: nil,
		BlockHash:   &hash,
	}
}

// BlockNumber represents decoding hex string to block values
type BlockNumber int64

const (
	// LatestBlockNumber mapping from "latest" to 0 for tm query
	LatestBlockNumber = BlockNumber(-2)
	// PendingBlockNumber mapping from "pending" to -1 for tm query
	PendingBlockNumber = BlockNumber(-1)
	// EarliestBlockNumber mapping from "earliest" to 1 for tm query (earliest query not supported)
	EarliestBlockNumber = BlockNumber(1)
)

// NewBlockNumber creates a new BlockNumber instance.
func NewBlockNumber(n *big.Int) BlockNumber {
	return BlockNumber(n.Int64())
}

// UnmarshalJSON parses the given JSON fragment into a BlockNumber. It supports:
// - "latest", "earliest" or "pending" as string arguments
// - the block number
// Returned errors:
// - an invalid block number error when the given argument isn't a known strings
// - an out of range error when the given block number is either too little or too large
func (bn *BlockNumber) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	switch input {
	case "earliest":
		*bn = EarliestBlockNumber
		return nil
	case "latest":
		*bn = LatestBlockNumber
		return nil
	case "pending":
		*bn = PendingBlockNumber
		return nil
	}

	blckNum, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if blckNum > math.MaxInt64 {
		return fmt.Errorf("blocknumber too high")
	}

	*bn = BlockNumber(blckNum)
	return nil
}

// Int64 converts block number to primitive type
func (bn BlockNumber) Int64() int64 {
	return int64(bn)
}
