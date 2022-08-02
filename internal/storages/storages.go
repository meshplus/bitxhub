package storages

import (
	"fmt"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/syndtr/goleveldb/leveldb/opt"
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
	bcStorage, err := leveldb.NewMultiLdb(repo.GetStoragePath(repoRoot, BlockChain), &opt.Options{
		WriteBuffer:         40 * opt.MiB,
		CompactionTableSize: 20 * opt.MiB,
		CompactionTotalSize: 100 * opt.MiB,
	}, 30*opt.GiB)
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
