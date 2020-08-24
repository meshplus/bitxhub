package router

import "github.com/meshplus/bitxhub-model/pb"

type Router interface {
	// Start starts the router module
	Start() error

	// Stop
	Stop() error

	// PutBlock
	PutBlockAndMeta(*pb.Block, *pb.InterchainMeta)

	// AddPier
	AddPier(id string, isUnion bool) (chan *pb.InterchainTxWrappers, error)

	// RemovePier
	RemovePier(id string, isUnion bool)

	// GetBlockHeader
	GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error

	GetInterchainTxWrappers(pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error
}
