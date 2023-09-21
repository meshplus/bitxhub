package adaptor

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb/errors"

	"github.com/axiomesh/axiom-kit/storage"
	"github.com/axiomesh/axiom-kit/storage/leveldb"
	"github.com/axiomesh/axiom-kit/storage/minifile"
	"github.com/axiomesh/axiom-kit/storage/pebble"
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
func (a *RBFTAdaptor) StoreState(key string, value []byte) error {
	a.store.DB.Put([]byte("consensus."+key), value)
	return nil
}

// DelState removes a key,value pair from the database with the given namespace
func (a *RBFTAdaptor) DelState(key string) error {
	a.store.DB.Delete([]byte("consensus." + key))
	return nil
}

// ReadState retrieves a value to a key from the database with the given namespace
func (a *RBFTAdaptor) ReadState(key string) ([]byte, error) {
	b := a.store.DB.Get([]byte("consensus." + key))
	if b == nil {
		return nil, errors.ErrNotFound
	}
	return b, nil
}

// ReadStateSet retrieves all key-value pairs where the key starts with prefix from the database with the given namespace
func (a *RBFTAdaptor) ReadStateSet(prefix string) (map[string][]byte, error) {
	prefixRaw := []byte("consensus." + prefix)

	ret := make(map[string][]byte)
	it := a.store.DB.Prefix(prefixRaw)
	if it == nil {
		return nil, errors.New("can't get Iterator")
	}

	if !it.Seek(prefixRaw) {
		err := fmt.Errorf("can not find key with %v in database", prefixRaw)
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
func (a *RBFTAdaptor) Destroy(key string) error {
	_ = a.store.DB.Close()
	_ = a.DeleteAllBatchState()
	return nil
}

func (a *RBFTAdaptor) StoreBatchState(key string, value []byte) error {
	return a.store.File.Put(key, value)
}

func (a *RBFTAdaptor) DelBatchState(key string) error {
	return a.store.File.Delete(key)
}

func (a *RBFTAdaptor) ReadBatchState(key string) ([]byte, error) {
	return a.store.File.Get(key)
}

func (a *RBFTAdaptor) ReadAllBatchState() (map[string][]byte, error) {
	return a.store.File.GetAll()
}

func (a *RBFTAdaptor) DeleteAllBatchState() error {
	return a.store.File.DeleteAll()
}
