package peermgr

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/model/events"
	network "github.com/meshplus/go-lightp2p"
)

//go:generate mockgen -destination mock_peermgr/mock_peermgr.go -package mock_peermgr -source peermgr.go
type PeerManager interface {
	// Start
	Start() error

	// Stop
	Stop() error

	// AsyncSend sends message to peer with peer info.
	AsyncSend(uint64, *pb.Message) error

	// SendWithStream sends message using existed stream
	SendWithStream(network.Stream, *pb.Message) error

	// Send sends message waiting response
	Send(uint64, *pb.Message) (*pb.Message, error)

	// Broadcast message to all node
	Broadcast(*pb.Message) error

	// Peers
	Peers() map[uint64]*peer.AddrInfo

	// OtherPeers
	OtherPeers() map[uint64]*peer.AddrInfo

	// SubscribeOrderMessage
	SubscribeOrderMessage(ch chan<- events.OrderMessageEvent) event.Subscription
}
