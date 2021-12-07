package peermgr

import (
	"encoding/json"
	"strings"

	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	node_mgr "github.com/meshplus/bitxhub-core/node-mgr"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub/internal/ledger"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

//ConnectionGater
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
	return true
}

func (g *connectionGater) InterceptAddrDial(p peer.ID, addr ma.Multiaddr) (allow bool) {
	return true
}

func (g *connectionGater) InterceptAccept(addr network.ConnMultiaddrs) (allow bool) {
	return true
}

func (g *connectionGater) InterceptSecured(d network.Direction, p peer.ID, addr network.ConnMultiaddrs) (allow bool) {
	lg := g.ledger.Copy()
	ok, nodeAccount := lg.GetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.VpNodePidKey(p.String())))
	if !ok {
		g.logger.Infof("Intercept a connection with an unavailable node(get node err: %s), peer.Pid: %s", string(nodeAccount), p.String())
		return false
	}
	nodeAccountStr := strings.Trim(string(nodeAccount), "\"")
	ok, nodeData := lg.GetState(constant.NodeManagerContractAddr.Address(), []byte(node_mgr.NodeKey(nodeAccountStr)))
	if !ok {
		g.logger.Errorf("InterceptSecured, node pid %s exist but node %s not exist: %v", p.String(), nodeAccountStr, string(nodeData))
		return false
	}

	node := &node_mgr.Node{}
	if err := json.Unmarshal(nodeData, node); err != nil {
		g.logger.Errorf("InterceptSecured, unmarshal node error: %v, nodeData: %s, pid: %s, account: %s", err, string(nodeData), p.String(), nodeAccountStr)
		return false
	}

	if node.IsAvailable() {
		g.logger.Infof("Connect with an available node, peer.Pid: %s, peer.Id: %d, peer.status: %s", p.String(), node.VPNodeId, node.Status)
		return true
	}

	g.logger.Infof("Intercept a connection with an unavailable node, peer.Pid: %s, peer.Id: %d, peer.status: %s", p.String(), node.VPNodeId, node.Status)
	return false
}

func (g *connectionGater) InterceptUpgraded(conn network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
