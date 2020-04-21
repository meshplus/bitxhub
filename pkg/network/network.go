package network

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub/pkg/network/pb"
)

type ConnectCallback func(*peer.AddrInfo) error

type MessageHandler func(network.Stream, []byte)

type Network interface {
	// Start start the network service.
	Start() error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by ID.
	Connect(*peer.AddrInfo) error

	// Disconnect peer with id
	Disconnect(*peer.AddrInfo) error

	// SetConnectionCallback sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// SetMessageHandler sets message handler
	SetMessageHandler(MessageHandler)

	// AsyncSend sends message to peer with peer info.
	AsyncSend(*peer.AddrInfo, *pb.Message) error

	// Send message using existed stream
	SendWithStream(network.Stream, *pb.Message) error

	// Send sends message waiting response
	Send(*peer.AddrInfo, *pb.Message) (*pb.Message, error)

	// Broadcast message to all node
	Broadcast([]*peer.AddrInfo, *pb.Message) error
}
