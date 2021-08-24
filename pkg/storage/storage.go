package storage

import "fmt"

var (
	ErrorNotFound = fmt.Errorf("not found in DB")
)

type Storage interface {
	Write

	// Get retrieves the object `value` named by `key`.
	// Get will return nil if the key is not mapped to a value.
	Get(key []byte) []byte

	// Has returns whether the `key` is mapped to a `value`.
	Has(key []byte) bool

	// Iterator iterates over a DB's key/value pairs in key order.
	Iterator(start, end []byte) Iterator

	// Prefix iterates over a DB's key/value pairs in key order including prefix.
	Prefix(prefix []byte) Iterator

	NewBatch() Batch

	Close() error
}

// Write is the write-side of the storage interface.
type Write interface {
	// Put stores the object `value` named by `key`.
	Put(key, value []byte)

	// Delete removes the value for given `key`.
	Delete(key []byte)
}

type Iterator interface {
	// Next moves the iterator to the next key/value pair.
	// It returns false if the iterator is exhausted.
	Next() bool

	// Prev moves the iterator to the previous key/value pair.
	// It returns false if the iterator is exhausted.
	Prev() bool

	// Seek moves the iterator to the first key/value pair whose key is greater
	// than or equal to the given key.
	// It returns whether such pair exist.
	//
	// It is safe to modify the contents of the argument after Seek returns.
	Seek(key []byte) bool

	// Key returns the key of the current key/value pair, or nil if done.
	Key() []byte

	// Value returns the value of the current key/value pair, or nil if done.
	Value() []byte
}

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
	Commit()
}
