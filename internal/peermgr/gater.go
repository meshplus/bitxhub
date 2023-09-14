package peermgr

import (
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom/internal/executor/system/base"
	"github.com/axiomesh/axiom/internal/ledger"
)

// ConnectionGater
type connectionGater struct {
	logger logrus.FieldLogger
	ledger *ledger.Ledger
}

func newConnectionGater(logger logrus.FieldLogger, ledger *ledger.Ledger) *connectionGater {
	return &connectionGater{
		logger: logger,
		ledger: ledger,
	}
}

func (g *connectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	peerID := p.String()
	// Use the latest EpochInfo for node connection
	epoch, err := base.GetNextEpochInfo(g.ledger.StateLedger)
	if err != nil {
		g.logger.Errorf("InterceptSecured, auth node %s failed, get node members error: %v", peerID, err)
		return false
	}
	if !lo.ContainsBy(epoch.ValidatorSet, func(item *rbft.NodeInfo) bool {
		return item.P2PNodeID == peerID
	}) && !lo.ContainsBy(epoch.CandidateSet, func(item *rbft.NodeInfo) bool {
		return item.P2PNodeID == peerID
	}) {
		g.logger.Warnf("InterceptSecured, auth node %s failed, unavailable node", peerID)
		return false
	}
	return true
}

func (g *connectionGater) InterceptAddrDial(p peer.ID, addr ma.Multiaddr) (allow bool) {
	return true
}

func (g *connectionGater) InterceptAccept(addr network.ConnMultiaddrs) (allow bool) {
	return true
}

func (g *connectionGater) InterceptSecured(d network.Direction, p peer.ID, addr network.ConnMultiaddrs) (allow bool) {
	return true
}

func (g *connectionGater) InterceptUpgraded(conn network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
