package peermgr

import (
	orderPeerMgr "github.com/meshplus/bitxhub-core/peer-mgr"
	"github.com/meshplus/bitxhub-model/pb"
	network "github.com/meshplus/go-lightp2p"
)

//go:generate mockgen -destination mock_peermgr/mock_peermgr.go -package mock_peermgr -source peermgr.go
type PeerManager interface {
	orderPeerMgr.OrderPeerManager

	// SendWithStream sends message using existed stream
	SendWithStream(network.Stream, *pb.Message) error

	// PierManager
	PierManager() PierManager

	// ReConfig
	ReConfig(config interface{}) error
}

type PierManager interface {
	Piers() *Piers

	AskPierMaster(string) (bool, error)
}
