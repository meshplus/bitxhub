package repo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub/pkg/cert"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	networkConfigFile = "network.toml"
	nodePrivFile      = "certs/node.priv"
)

// NetworkConfig .
// @param OtherNodes to fit original code
// @param OthersNodes for new network config
type NetworkConfig struct {
	ID         uint64
	N          uint64
	LocalAddr  string
	Nodes      []*NetworkNode
	OtherNodes map[uint64]*peer.AddrInfo
}

// NetworkNode is the struct to describe network conf of a node
// @param ID is the id of the node, it is origined by sorting.
// @param Addrs is the address array of the node.
// @param Addr is the default used address the node.
type NetworkNode struct {
	ID    uint64
	Addr  string // the optimal address of a node
	Addrs []string
}

// ReadinNetworkConfig is used for read in toml file
type ReadinNetworkConfig struct {
	Addrs [][]string
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

// AddrsToPeerInfo transfer addrs to PeerInfo
func AddrsToPeerInfo(multiAddrs []string) ([]peer.AddrInfo, error) {
	maddrs := []ma.Multiaddr{}
	for _, multiAddr := range multiAddrs {
		maddr, err := ma.NewMultiaddr(multiAddr)
		if err != nil {
			return nil, err
		}
		maddrs = append(maddrs, maddr)
	}

	return peer.AddrInfosFromP2pAddrs(maddrs...)
}

// loadNetworkConfig is compatible with old network.toml and support new network.toml config file
func loadNetworkConfig(repoRoot string) (*NetworkConfig, error) {

	rdiNetworkConfig := &ReadinNetworkConfig{}
	if err := ReadConfig(filepath.Join(repoRoot, networkConfigFile), "toml", rdiNetworkConfig); err != nil {
		return nil, err
	}

	networkConfig := &NetworkConfig{}
	for _, node := range rdiNetworkConfig.Addrs {
		networkConfig.Nodes = append(networkConfig.Nodes, &NetworkNode{Addrs: node})
	}

	// whether new network format is new
	if networkConfig.N == 0 { // judge whether new network format is new
		networkConfig.N = uint64(len(networkConfig.Nodes))
	}

	if uint64(len(networkConfig.Nodes)) != networkConfig.N {
		return nil, fmt.Errorf("wrong nodes number")
	}

	// use the first address of node as its default addr
	for _, node := range networkConfig.Nodes {
		if node.Addr == "" {
			node.Addr = node.Addrs[0]
		}
	}
	// read private key to get PeerID
	PeerID, err := GetPidFromPrivFile(filepath.Join(repoRoot, nodePrivFile))
	if err != nil {
		return nil, err
	}
	// sort PeerId of nodes to produce IDs:
	sort.Sort(networkConfig)

	findSelf := false

	for i, node := range networkConfig.Nodes {
		// write ID into node struct:
		node.ID = uint64(i + 1)

		pid, err := MultiaddrToPeerID(networkConfig.Nodes[i].Addrs[0])
		if err != nil {
			return nil, err
		}
		if pid == PeerID {
			// match PeerID to know node's self ID:
			networkConfig.ID = node.ID
			findSelf = true
		}
	}

	if findSelf == false {
		return nil, fmt.Errorf("PeerID of this node was not matched to any of these nodes")
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
			addrs, err := AddrsToPeerInfo(node.Addrs)
			if err != nil {
				return nil, fmt.Errorf("wrong network addr: %w", err)
			}
			if len(addrs) != 1 {
				return nil, fmt.Errorf("different PeerIDs in the same node")
			}
			// overwrite addr if formatIsNew is true.
			addr := &addrs[0]
			m[node.ID] = addr
		}
	}

	networkConfig.OtherNodes = m

	return networkConfig, nil
}

// Len returns length of the struct to be sorted
func (p NetworkConfig) Len() int { return len(p.Nodes) }

// Less compares two iterms ascending(ASC)
func (p NetworkConfig) Less(i, j int) bool {
	multiAddri := p.Nodes[i].Addrs[0]
	multiAddrj := p.Nodes[j].Addrs[0]
	maddri, err := ma.NewMultiaddr(multiAddri)
	if err != nil {
		panic(err)
	}
	maddrj, err := ma.NewMultiaddr(multiAddrj)
	if err != nil {
		panic(err)
	}
	_, idi := peer.SplitAddr(maddri)
	if idi == "" {
		panic(err)
	}
	_, idj := peer.SplitAddr(maddrj)
	if idj == "" {
		panic(err)
	}
	return idi < idj
}

// Swap swaps iterms
func (p NetworkConfig) Swap(i, j int) { p.Nodes[i], p.Nodes[j] = p.Nodes[j], p.Nodes[i] }

// GetPidFromPrivFile gets pid from libp2p node priv file
func GetPidFromPrivFile(privPath string) (string, error) {
	data, err := ioutil.ReadFile(privPath)
	if err != nil {
		return "", fmt.Errorf("read private key: %w", err)
	}
	privKey, err := cert.ParsePrivateKey(data, crypto2.ECDSA_P256)
	if err != nil {
		return "", err
	}

	_, pk, err := crypto.KeyPairFromStdKey(privKey.K)
	if err != nil {
		return "", err
	}

	pid, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", err
	}

	return pid.String(), nil
}

// MultiaddrToPeerID .
func MultiaddrToPeerID(multiAddr string) (string, error) {
	maddri, err := ma.NewMultiaddr(multiAddr)
	if err != nil {
		return "", err
	}
	_, PeerID := peer.SplitAddr(maddri)
	if PeerID == "" {
		return "", err
	}
	return PeerID.String(), nil
}
