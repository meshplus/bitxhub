package order

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/sirupsen/logrus"
	"github.com/willf/bloom"
)

const (
	filterDbKey = "bloom_filter"
	m           = 10000000 //bits
	k           = 4        //calc hash times
)

type ReqLookUp struct {
	sync.Mutex
	filter  *bloom.BloomFilter
	storage storage.Storage
	buffer  bytes.Buffer
	logger  logrus.FieldLogger //logger
}

func NewReqLookUp(storage storage.Storage, logger logrus.FieldLogger) (*ReqLookUp, error) {
	filter := bloom.New(m, k)
	filterDB := storage.Get([]byte(filterDbKey))
	if filterDB != nil {
		var b bytes.Buffer
		if _, err := b.Write(filterDB); err != nil {
			return nil, fmt.Errorf("write to buffer failed: %w", err)
		}
		if _, err := filter.ReadFrom(&b); err != nil {
			return nil, fmt.Errorf("read from filter error: %w", err)
		}
	}
	return &ReqLookUp{
		filter:  filter,
		storage: storage,
		logger:  logger,
	}, nil

}

func (r *ReqLookUp) Add(key []byte) {
	r.filter.Add(key)
}

func (r *ReqLookUp) LookUp(key []byte) bool {
	return r.filter.TestAndAdd(key)
}

func (r *ReqLookUp) Build() error {
	r.Lock()
	defer r.Unlock()
	if _, err := r.filter.WriteTo(&r.buffer); err != nil {
		return fmt.Errorf("write to buffer failed: %w", err)
	}
	r.storage.Put([]byte(filterDbKey), r.buffer.Bytes())
	r.buffer.Reset()
	return nil
}
