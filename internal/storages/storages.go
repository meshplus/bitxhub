package storages

import (
	"fmt"

	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/storage/pebble"
	"github.com/axiomesh/axiom/internal/repo"
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

func Initialize(repoRoot string, typ string) error {
	var bcStorage storage.Storage
	var err error
	if typ == "leveldb" {
		bcStorage, err = leveldb.New(repo.GetStoragePath(repoRoot, BlockChain))
		if err != nil {
			return fmt.Errorf("create blockchain storage: %w", err)
		}
	} else if typ == "pebble" {
		bcStorage, err = pebble.New(repo.GetStoragePath(repoRoot, BlockChain))
		if err != nil {
			return fmt.Errorf("create blockchain storage: %w", err)
		}
	} else {
		return fmt.Errorf("unknow kv type %s, expect leveldb or pebble", typ)
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
