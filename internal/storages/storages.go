package storages

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub/internal/repo"
)

const (
	BlockChain = "blockchain"
)

var s = &wrapper{
	storages: make(map[string]storage.Storage),
}

type wrapper struct {
	storages map[string]storage.Storage
}

func Initialize(repoRoot string) error {
	bcStorage, err := leveldb.New(repo.GetStoragePath(repoRoot, BlockChain))
	if err != nil {
		return fmt.Errorf("create blockchain storage: %w", err)
	}

	s.storages[BlockChain] = bcStorage

	return nil
}

func Get(name string) (storage.Storage, error) {
	storage, ok := s.storages[name]
	if !ok {
		return nil, fmt.Errorf("wrong storage name")
	}

	return storage, nil
}
