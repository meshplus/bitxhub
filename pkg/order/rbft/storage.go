package rbft

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/meshplus/bitxhub-kit/storage/minifile"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

type Storage struct {
	DB   storage.Storage
	File *minifile.MiniFile
}

func NewStorage(path string) (*Storage, error) {
	db, err := leveldb.New(path)
	if err != nil {
		return nil, err
	}

	file, err := minifile.New(filepath.Join(path, "file"))
	if err != nil {
		return nil, err
	}

	return &Storage{
		DB:   db,
		File: file,
	}, nil
}

// StoreState stores a key,value pair to the database with the given namespace
func (s *Stack) StoreState(key string, value []byte) error {
	s.store.DB.Put([]byte("consensus."+key), value)
	return nil
}

// DelState removes a key,value pair from the database with the given namespace
func (s *Stack) DelState(key string) error {
	s.store.DB.Delete([]byte("consensus." + key))
	return nil
}

// ReadState retrieves a value to a key from the database with the given namespace
func (s *Stack) ReadState(key string) ([]byte, error) {
	b := s.store.DB.Get([]byte("consensus." + key))
	if b == nil {
		return nil, errors.ErrNotFound
	}
	return b, nil
}

// ReadStateSet retrieves all key-value pairs where the key starts with prefix from the database with the given namespace
func (s *Stack) ReadStateSet(prefix string) (map[string][]byte, error) {
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

func (s *Stack) Destroy() error {
	// TODO (xcc): Destroy db
	_ = s.store.DB.Close()
	_ = s.DeleteAllBatchState()
	return nil
}

func (s *Stack) StoreBatchState(key string, value []byte) error {
	return s.store.File.Put(key, value)
}

func (s *Stack) DelBatchState(key string) error {
	return s.store.File.Delete(key)
}

func (s *Stack) ReadBatchState(key string) ([]byte, error) {
	return s.store.File.Get(key)
}

func (s *Stack) ReadAllBatchState() (map[string][]byte, error) {
	return s.store.File.GetAll()
}

func (s *Stack) DeleteAllBatchState() error {
	return s.store.File.DeleteAll()
}
