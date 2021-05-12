package router

import (
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxid"
)

type Router interface {
	// Start starts the router module
	Start() error

	// Stop
	Stop() error

	// PutBlock
	PutBlockAndMeta(*pb.Block, *pb.InterchainMeta)

	// AddPier
	AddPier(subscribeDID bitxid.DID, pierID string, isUnion bool) (chan *pb.InterchainTxWrappers, error)

	// RemovePier
	RemovePier(subscribeDID bitxid.DID, pierID string, isUnion bool)

	// GetBlockHeader
	GetBlockHeader(begin, end uint64, ch chan<- *pb.BlockHeader) error

	GetInterchainTxWrappers(did string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error
}
