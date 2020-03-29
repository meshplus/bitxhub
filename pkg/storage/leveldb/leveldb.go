package leveldb

import (
	"github.com/meshplus/bitxhub/pkg/storage"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type ldb struct {
	db *leveldb.DB
}

func New(path string) (storage.Storage, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &ldb{
		db: db,
	}, nil
}

func (l *ldb) Put(key, value []byte) error {
	return l.db.Put(key, value, nil)
}

func (l *ldb) Delete(key []byte) error {
	return l.db.Delete(key, nil)
}

func (l *ldb) Get(key []byte) (value []byte, err error) {
	return l.db.Get(key, nil)
}

func (l *ldb) Has(key []byte) (exists bool, err error) {
	return l.db.Has(key, nil)
}

func (l *ldb) Iterator(start, end []byte) storage.Iterator {
	rg := &util.Range{
		Start: start,
		Limit: end,
	}
	it := l.db.NewIterator(rg, nil)

	return &iter{iter: it}
}

func (l *ldb) Prefix(prefix []byte) storage.Iterator {
	rg := util.BytesPrefix(prefix)

	return &iter{iter: l.db.NewIterator(rg, nil)}
}

func (l *ldb) NewBatch() storage.Batch {
	return &ldbBatch{
		ldb:   l.db,
		batch: &leveldb.Batch{},
	}
}

func (l *ldb) Close() error {
	return l.db.Close()
}

type ldbBatch struct {
	ldb   *leveldb.DB
	batch *leveldb.Batch
}

func (l *ldbBatch) Put(key, value []byte) {
	l.batch.Put(key, value)
}

func (l *ldbBatch) Delete(key []byte) {
	l.batch.Delete(key)
}

func (l *ldbBatch) Commit() error {
	return l.ldb.Write(l.batch, nil)
}
