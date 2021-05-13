package peermgr

import (
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/meshplus/bitxhub-model/pb"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
	"sync"
)

type notifiee struct {
	peers   map[uint64]*pb.VpInfo
	// TODO (Peer): keep access goroutine safety
	newPeer string
	mu      sync.RWMutex
	logger  logrus.FieldLogger
}

func newNotifiee(peers map[uint64]*pb.VpInfo, logger logrus.FieldLogger) *notifiee {
	return &notifiee{
		peers: peers,
		logger: logger,
	}
}

func (n *notifiee) Listen(network network.Network, multiaddr ma.Multiaddr) {
}

func (n *notifiee) ListenClose(network network.Network, multiaddr ma.Multiaddr) {
}

func (n *notifiee) Connected(network network.Network, conn network.Conn) {
	peers := n.getPeers()
	newAddr := conn.RemotePeer().String()
	// check if the newAddr has already in peers.
	for _, p := range peers {
		if p.Pid == newAddr {
			return
		}
	}
	if n.newPeer == "" {
		n.newPeer = newAddr
		n.logger.Infof("Updating notifiee newPeer %s", newAddr)
		return
	}
	n.logger.Infof("The newPeer %s is not nil, skip the new addr %s", n.newPeer, newAddr)
}

func (n *notifiee) Disconnected(network network.Network, conn network.Conn) {
}

func (n *notifiee) OpenedStream(network network.Network, stream network.Stream) {
}

func (n *notifiee) ClosedStream(network network.Network, stream network.Stream) {
}

func (n *notifiee) getPeers() map[uint64]*pb.VpInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.peers
}

func (n *notifiee) setPeers(peers map[uint64]*pb.VpInfo) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers = peers
}
