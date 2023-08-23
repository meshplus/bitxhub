package coreapi

import (
	"encoding/json"
	"fmt"

	"github.com/axiomesh/axiom/internal/coreapi/api"
)

type NetworkAPI CoreAPI

var _ api.NetworkAPI = (*NetworkAPI)(nil)

// PeerInfo collects the peers' info in p2p network.
func (network *NetworkAPI) PeerInfo() ([]byte, error) {
	peerInfo := network.axiom.PeerMgr.OrderPeers()

	data, err := json.Marshal(peerInfo)
	if err != nil {
		return nil, fmt.Errorf("marshal peer info error: %w", err)
	}

	return data, nil
}

func (network *NetworkAPI) DelVPNode(pid string) ([]byte, error) {
	if pid == "" {
		return nil, fmt.Errorf("pid is null")
	}
	return nil, nil
}
