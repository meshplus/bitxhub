package coreapi

import (
	"encoding/json"

	"github.com/meshplus/bitxhub/internal/coreapi/api"
)

type NetworkAPI CoreAPI

var _ api.NetworkAPI = (*NetworkAPI)(nil)

// PeerInfo collects the peers' info in p2p network.
func (network *NetworkAPI) PeerInfo() ([]byte, error) {
	peerInfo := network.bxh.PeerMgr.Peers()

	data, err := json.Marshal(peerInfo)
	if err != nil {
		return nil, err
	}

	return data, nil
}
