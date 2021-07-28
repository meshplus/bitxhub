package router

import (
	"github.com/meshplus/bitxhub-model/pb"
)

type Router interface {
	// Start starts the router module
	Start() error

	// Stop
	Stop() error

	// PutBlock
	PutBlockAndMeta(*pb.Block, *pb.InterchainMeta)

	// AddPier
	AddPier(pierID string) (chan *pb.InterchainTxWrappers, error)

	// RemovePier
	RemovePier(pierID string)

	// GetBlockHeader
	GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error

	GetInterchainTxWrappers(appchainID string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error
}
