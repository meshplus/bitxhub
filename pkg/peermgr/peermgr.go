package peermgr

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
)

//go:generate mockgen -destination mock_peermgr/mock_peermgr.go -package mock_peermgr -source peermgr.go
type PeerManager interface {
	// Start
	Start() error

	// Stop
	Stop() error

	// Send message to peer with peer info.
	Send(uint64, *pb.Message) error

	// Send message using existed stream
	SendWithStream(network.Stream, *pb.Message) error

	// Sync Send message
	SyncSend(uint64, *pb.Message) (*pb.Message, error)

	// Broadcast message to all node
	Broadcast(*pb.Message) error

	// Peers
	Peers() map[uint64]*peer.AddrInfo

	// OtherPeers
	OtherPeers() map[uint64]*peer.AddrInfo

	// SubscribeOrderMessage
	SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription
}
