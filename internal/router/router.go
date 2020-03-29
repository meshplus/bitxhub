package router

import "github.com/meshplus/bitxhub-model/pb"

type Router interface {
	// Start starts the router module
	Start() error

	// Stop
	Stop() error

	// PutBlock
	PutBlock(*pb.Block)

	// AddPier
	AddPier(id string) (chan *pb.MerkleWrapper, error)

	// RemovePier
	RemovePier(id string)

	// GetMerkleWrapper
	GetMerkleWrapper(pid string, begin, end uint64, ch chan<- *pb.MerkleWrapper) error
}
