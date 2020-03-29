package repo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type NetworkConfig struct {
	ID         uint64
	N          uint64
	LocalAddr  string
	Nodes      []*NetworkNodes
	OtherNodes map[uint64]*peer.AddrInfo
}

type NetworkNodes struct {
	ID   uint64
	Addr string
}

// AddrToPeerInfo transfer addr to PeerInfo
// addr example: "/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64"
func AddrToPeerInfo(multiAddr string) (*peer.AddrInfo, error) {
	maddr, err := ma.NewMultiaddr(multiAddr)
	if err != nil {
		return nil, err
	}

	return peer.AddrInfoFromP2pAddr(maddr)
}

func loadNetworkConfig(repoRoot string) (*NetworkConfig, error) {
	networkConfig := &NetworkConfig{}
	if err := ReadConfig(filepath.Join(repoRoot, "network.toml"), networkConfig); err != nil {
		return nil, err
	}

	if uint64(len(networkConfig.Nodes)) != networkConfig.N {
		return nil, fmt.Errorf("wrong nodes number")
	}

	for _, node := range networkConfig.Nodes {
		if node.ID == networkConfig.ID {
			networkConfig.LocalAddr = node.Addr
		}
	}

	if networkConfig.LocalAddr == "" {
		return nil, fmt.Errorf("lack of local address")
	}

	idx := strings.LastIndex(networkConfig.LocalAddr, "/p2p/")
	if idx == -1 {
		return nil, fmt.Errorf("pid is not existed in bootstrap")
	}

	networkConfig.LocalAddr = networkConfig.LocalAddr[:idx]

	nodes := networkConfig.Nodes
	m := make(map[uint64]*peer.AddrInfo)
	for _, node := range nodes {
		if node.ID != networkConfig.ID {
			addr, err := AddrToPeerInfo(node.Addr)
			if err != nil {
				return nil, fmt.Errorf("wrong network addr: %w", err)
			}
			m[node.ID] = addr
		}
	}

	networkConfig.OtherNodes = m

	return networkConfig, nil
}
