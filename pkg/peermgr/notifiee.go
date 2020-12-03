package peermgr

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"
)

type notifiee struct {
}

func (n notifiee) Listen(network network.Network, multiaddr ma.Multiaddr) {
}

func (n notifiee) ListenClose(network network.Network, multiaddr ma.Multiaddr) {
}

func (n notifiee) Connected(network network.Network, conn network.Conn) {
	fmt.Println("new connect:" + conn.RemotePeer().String())
}

func (n notifiee) Disconnected(network network.Network, conn network.Conn) {
}

func (n notifiee) OpenedStream(network network.Network, stream network.Stream) {
}

func (n notifiee) ClosedStream(network network.Network, stream network.Stream) {
}
