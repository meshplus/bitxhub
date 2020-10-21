package boltvm

import (
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -destination mock_stub/mock_stub.go -package mock_stub -source stub.go
type Stub interface {
	// Caller
	Caller() string
	// Callee
	Callee() string
	// Logger
	Logger() logrus.FieldLogger
	// GetTxHash returns the transaction hash
	GetTxHash() *types.Hash
	// GetTxIndex returns the transaction index in the block
	GetTxIndex() uint64
	// Has judges key
	Has(key string) bool
	// Get gets value from datastore by key
	Get(key string) (bool, []byte)
	// GetObject
	GetObject(key string, ret interface{}) bool
	// Set sets k-v
	Set(key string, value []byte)
	// SetObject sets k with object v, v will be marshaled using json
	SetObject(key string, value interface{})
	// AddObject adds k with object v, v will be marshaled using json
	AddObject(key string, value interface{})
	// Delete deletes k-v
	Delete(key string)
	// QueryByPrefix queries object by prefix
	Query(prefix string) (bool, [][]byte)
	// PostEvent posts event to external
	PostEvent(interface{})
	// PostInterchainEvent posts interchain event to external
	PostInterchainEvent(interface{})
	// Validator returns the instance of validator
	ValidationEngine() validator.Engine
	// CrossInvoke cross contract invoke
	CrossInvoke(address, method string, args ...*pb.Arg) *Response
}
