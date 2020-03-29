package network

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub/pkg/network/proto"
)

// ID represents peer id
type ID interface{}

type ConnectCallback func(ID) error

type IDStore interface {
	Add(ID, *peer.AddrInfo)
	Remove(ID)
	Addr(ID) *peer.AddrInfo
	Addrs() map[ID]*peer.AddrInfo
}

type Network interface {
	// Start start the network service.
	Start() error

	// Stop stop the network service.
	Stop() error

	// Connect connects peer by ID.
	Connect(ID) error

	// Disconnect peer with id
	Disconnect(ID) error

	// SetConnectionCallback Sets the callback after connecting
	SetConnectCallback(ConnectCallback)

	// Send message to peer with peer info.
	Send(ID, *proto.Message) error

	// Send message using existed stream
	SendWithStream(s network.Stream, msg *proto.Message) error

	// Sync Send message
	SyncSend(ID, *proto.Message) (*proto.Message, error)

	// Broadcast message to all node
	Broadcast([]ID, *proto.Message) error

	// Receive message from the channel
	Receive() <-chan *MessageStream

	// IDStore
	IDStore() IDStore
}
