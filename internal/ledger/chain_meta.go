package ledger

import (
	"fmt"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/storage"
)

var (
	chainKey = []byte("chain-meta")
)

func loadChainMeta(store storage.Storage) (*pb.ChainMeta, error) {
	ok, err := store.Has(chainKey)
	if err != nil {
		return nil, fmt.Errorf("judge chain meta: %w", err)
	}

	chain := &pb.ChainMeta{}
	if ok {
		body, err := store.Get(chainKey)
		if err != nil {
			return nil, fmt.Errorf("get chain meta: %w", err)
		}

		if err := chain.Unmarshal(body); err != nil {
			return nil, fmt.Errorf("unmarshal chain meta: %w", err)
		}
	}

	return chain, nil
}
