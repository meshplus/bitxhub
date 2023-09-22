package storagemgr

import (
	"fmt"
	"sync"

	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/storage/pebble"
	"github.com/axiomesh/axiom-ledger/pkg/repo"
)

const (
	BlockChain = "blockchain"
	Ledger     = "ledger"
	Consensus  = "consensus"
)

var globalStorageMgr = &storageMgr{
	storages: make(map[string]storage.Storage),
	lock:     new(sync.Mutex),
}

type storageMgr struct {
	storageBuilder func(name string) (storage.Storage, error)
	storages       map[string]storage.Storage
	lock           *sync.Mutex
}

func Initialize(repoRoot string, typ string) error {
	switch typ {
	case repo.KVStorageTypeLeveldb:
		globalStorageMgr.storageBuilder = func(name string) (storage.Storage, error) {
			return leveldb.New(repo.GetStoragePath(repoRoot, name))
		}
	case repo.KVStorageTypePebble:
		globalStorageMgr.storageBuilder = func(name string) (storage.Storage, error) {
			return pebble.New(repo.GetStoragePath(repoRoot, name))
		}
	default:
		return fmt.Errorf("unknow kv type %s, expect leveldb or pebble", typ)
	}
	return nil
}

func Open(name string) (storage.Storage, error) {
	globalStorageMgr.lock.Lock()
	defer globalStorageMgr.lock.Unlock()
	s, ok := globalStorageMgr.storages[name]
	if !ok {
		var err error
		s, err = globalStorageMgr.storageBuilder(name)
		if err != nil {
			return nil, err
		}
		globalStorageMgr.storages[name] = s
	}
	return s, nil
}
