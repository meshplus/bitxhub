package ledger

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

var (
	chainKey = []byte("chain-meta")
)

func loadChainMeta(store storage.Storage) (*pb.ChainMeta, error) {
	ok := store.Has(chainKey)

	chain := &pb.ChainMeta{
		Height:            0,
		BlockHash:         &types.Hash{},
		InterchainTxCount: 0,
	}
	if ok {
		body := store.Get(chainKey)

		if err := chain.Unmarshal(body); err != nil {
			return nil, fmt.Errorf("unmarshal chain meta: %w", err)
		}
	}

	return chain, nil
}
