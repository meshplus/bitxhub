package adaptor

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/storage/minifile"
	"github.com/meshplus/bitxhub-kit/storage/pebble"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

type storageWrapper struct {
	DB   storage.Storage
	File *minifile.MiniFile
}

func newStorageWrapper(path, typ string) (*storageWrapper, error) {
	var db storage.Storage
	var err error
	if typ == "leveldb" {
		db, err = leveldb.New(path)
		if err != nil {
			return nil, err
		}
	} else if typ == "pebble" {
		db, err = pebble.New(path)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknow kv type %s, expect leveldb or pebble", typ)
	}

	file, err := minifile.New(filepath.Join(path, "file"))
	if err != nil {
		return nil, err
	}

	return &storageWrapper{
		DB:   db,
		File: file,
	}, nil
}

// StoreState stores a key,value pair to the database with the given namespace
func (s *RBFTAdaptor) StoreState(key string, value []byte) error {
	s.store.DB.Put([]byte("consensus."+key), value)
	return nil
}

// DelState removes a key,value pair from the database with the given namespace
func (s *RBFTAdaptor) DelState(key string) error {
	s.store.DB.Delete([]byte("consensus." + key))
	return nil
}

// ReadState retrieves a value to a key from the database with the given namespace
func (s *RBFTAdaptor) ReadState(key string) ([]byte, error) {
	b := s.store.DB.Get([]byte("consensus." + key))
	if b == nil {
		return nil, errors.ErrNotFound
	}
	return b, nil
}

// ReadStateSet retrieves all key-value pairs where the key starts with prefix from the database with the given namespace
func (s *RBFTAdaptor) ReadStateSet(prefix string) (map[string][]byte, error) {
	prefixRaw := []byte("consensus." + prefix)

	ret := make(map[string][]byte)
	it := s.store.DB.Prefix(prefixRaw)
	if it == nil {
		err := errors.New(fmt.Sprint("Can't get Iterator"))
		return nil, err
	}

	if !it.Seek(prefixRaw) {
		err := fmt.Errorf("can not find key with %s in database", prefixRaw)
		return nil, err
	}

	for bytes.HasPrefix(it.Key(), prefixRaw) {
		key := string(it.Key())
		key = key[len("consensus."):]
		ret[key] = append([]byte(nil), it.Value()...)
		if !it.Next() {
			break
		}
	}
	return ret, nil
}

// Notice: not used
func (s *RBFTAdaptor) Destroy(key string) error {
	_ = s.store.DB.Close()
	_ = s.DeleteAllBatchState()
	return nil
}

func (s *RBFTAdaptor) StoreBatchState(key string, value []byte) error {
	return s.store.File.Put(key, value)
}

func (s *RBFTAdaptor) DelBatchState(key string) error {
	return s.store.File.Delete(key)
}

func (s *RBFTAdaptor) ReadBatchState(key string) ([]byte, error) {
	return s.store.File.Get(key)
}

func (s *RBFTAdaptor) ReadAllBatchState() (map[string][]byte, error) {
	return s.store.File.GetAll()
}

func (s *RBFTAdaptor) DeleteAllBatchState() error {
	return s.store.File.DeleteAll()
}
