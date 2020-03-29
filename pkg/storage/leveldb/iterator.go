package leveldb

import "github.com/syndtr/goleveldb/leveldb/iterator"

type iter struct {
	iter iterator.Iterator
}

func (it *iter) Prev() bool {
	return it.iter.Prev()
}

func (it *iter) Seek(key []byte) bool {
	return it.iter.Seek(key)
}

func (it *iter) Next() bool {
	return it.iter.Next()
}

func (it *iter) Key() []byte {
	return it.iter.Key()
}

func (it *iter) Value() []byte {
	return it.iter.Value()
}
