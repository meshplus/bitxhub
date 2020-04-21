package network

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub/pkg/network/proto"
)

type ConnectCallback func(*peer.AddrInfo) error

type Network interface {
	// Start start the network service.
	Start() error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by ID.
	Connect(*peer.AddrInfo) error

	// Disconnect peer with id
	Disconnect(*peer.AddrInfo) error

	// SetConnectionCallback Sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// AsyncSend sends message to peer with peer info.
	AsyncSend(*peer.AddrInfo, *proto.Message) error

	// Send message using existed stream
	SendWithStream(s network.Stream, msg *proto.Message) error

	// Send sends message waiting response
	Send(*peer.AddrInfo, *proto.Message) (*proto.Message, error)

	// Broadcast message to all node
	Broadcast([]*peer.AddrInfo, *proto.Message) error

	// Receive message from the channel
	Receive() <-chan *MessageStream
}
