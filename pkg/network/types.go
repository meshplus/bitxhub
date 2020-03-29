package network

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub/pkg/network/proto"
)

type OnConnectCallback func(*peer.AddrInfo, ID)

type MessageStream struct {
	Message *proto.Message
	Stream  network.Stream
}

func Message(data []byte) *proto.Message {
	return &proto.Message{
		Data:    data,
		Version: "1.0",
	}
}
