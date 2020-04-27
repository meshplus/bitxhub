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
	ok := store.Has(chainKey)

	chain := &pb.ChainMeta{}
	if ok {
		body := store.Get(chainKey)

		if err := chain.Unmarshal(body); err != nil {
			return nil, fmt.Errorf("unmarshal chain meta: %w", err)
		}
	}

	return chain, nil
}
