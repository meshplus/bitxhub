package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	crypto2 "github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	libp2pcert "github.com/meshplus/go-libp2p-cert"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml"
	"github.com/spf13/viper"
)

type NetworkConfig struct {
	ID        uint64          `toml:"id" json:"id"`
	N         uint64          `toml:"n" json:"n"`
	New       bool            `toml:"new" json:"new"`
	LocalAddr string          `toml:"local_addr, omitempty" json:"local_addr"`
	Nodes     []*NetworkNodes `toml:"nodes" json:"nodes"`
	Genesis   Genesis         `toml:"genesis, omitempty" json:"genesis"`
}

type NetworkNodes struct {
	ID      uint64   `toml:"id" json:"id"`
	Pid     string   `toml:"pid" json:"pid"`
	Hosts   []string `toml:"hosts" json:"hosts"`
	Account string   `toml:"account" json:"account"`
}

func loadNetworkConfig(viper *viper.Viper, repoRoot string, genesis Genesis) (*NetworkConfig, error) {
	networkConfig := &NetworkConfig{Genesis: genesis}
	checkReaptAddr := make(map[string]uint64)
	if err := ReadConfig(viper, filepath.Join(repoRoot, "network.toml"), "toml", networkConfig); err != nil {
		return nil, fmt.Errorf("read network config error: %w", err)
	}

	for _, node := range networkConfig.Nodes {
		if node.ID == networkConfig.ID {
			if len(node.Hosts) == 0 {
				return nil, fmt.Errorf("no hosts found by node:%d", node.ID)
			}
			networkConfig.LocalAddr = node.Hosts[0]
			addr, err := ma.NewMultiaddr(fmt.Sprintf("%s%s", node.Hosts[0], node.Pid))
			if err != nil {
				return nil, fmt.Errorf("new multiaddr: %w", err)
			}
			networkConfig.LocalAddr = strings.Replace(networkConfig.LocalAddr, ma.Split(addr)[0].String(), "/ip4/0.0.0.0", -1)
		}
		if _, ok := checkReaptAddr[node.Hosts[0]]; !ok {
			checkReaptAddr[node.Hosts[0]] = node.ID
		} else {
			return nil, fmt.Errorf("reapt address")
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
	return networkConfig, nil
}

// GetVpInfos gets vp info from network config
func (config *NetworkConfig) GetVpInfos() map[uint64]*pb.VpInfo {
	vpNodes := make(map[uint64]*pb.VpInfo)
	for _, node := range config.Nodes {
		vpInfo := &pb.VpInfo{
			Id:      node.ID,
			Pid:     node.Pid,
			Account: node.Account,
			Hosts:   node.Hosts,
		}
		vpNodes[node.ID] = vpInfo
	}
	return vpNodes
}

// GetVpGenesisAccount gets genesis address from network config
func (config *NetworkConfig) GetVpGenesisAccount() map[uint64]types.Address {
	m := make(map[uint64]types.Address)
	for i, admin := range config.Genesis.Admins {
		m[uint64(i)+1] = *types.NewAddressByStr(admin.Address)
	}
	return m
}

// GetVpAccount gets genesis address from network config
func (config *NetworkConfig) GetVpAccount() map[uint64]types.Address {
	m := make(map[uint64]types.Address)
	for _, node := range config.Nodes {
		m[node.ID] = *types.NewAddressByStr(node.Account)
	}
	return m
}

// GetNetworkPeers gets all peers from network config
func (config *NetworkConfig) GetNetworkPeers() (map[uint64]*peer.AddrInfo, error) {
	peers := make(map[uint64]*peer.AddrInfo)
	for _, node := range config.Nodes {
		if len(node.Hosts) == 0 {
			return nil, fmt.Errorf("no hosts found by node:%d", node.ID)
		}
		multiaddr, err := ma.NewMultiaddr(fmt.Sprintf("%s%s", node.Hosts[0], node.Pid))
		if err != nil {
			return nil, fmt.Errorf("new Multiaddr error:%w", err)
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(multiaddr)
		if err != nil {
			return nil, fmt.Errorf("convert multiaddr to addrInfo failed: %w", err)
		}
		for i := 1; i < len(node.Hosts); i++ {
			multiaddr, err := ma.NewMultiaddr(fmt.Sprintf("%s%s", node.Hosts[i], node.Pid))
			if err != nil {
				return nil, fmt.Errorf("new Multiaddr error:%w", err)
			}
			addrInfo.Addrs = append(addrInfo.Addrs, multiaddr)
		}

		peers[node.ID] = addrInfo
	}
	return peers, nil
}

func RewriteNetworkConfig(repoRoot string, infos map[uint64]*pb.VpInfo, isNew bool) error {
	networkConfig := &NetworkConfig{}
	v := viper.New()
	v.SetConfigFile(filepath.Join(repoRoot, "network.toml"))
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("readInConfig error: %w", err)
	}
	if err := v.Unmarshal(networkConfig); err != nil {
		return fmt.Errorf("unmarshal network config error: %w", err)
	}

	nodes := make([]*NetworkNodes, 0, len(infos))
	routers := make([]*pb.VpInfo, 0, len(nodes))
	for _, info := range infos {
		routers = append(routers, info)
	}
	sort.Slice(routers, func(i, j int) bool {
		return routers[i].Id < routers[j].Id
	})
	for _, info := range routers {
		node := &NetworkNodes{
			ID:      info.Id,
			Pid:     info.Pid,
			Account: info.Account,
			Hosts:   info.Hosts,
		}
		nodes = append(nodes, node)
	}
	networkConfig.Nodes = nodes
	networkConfig.N = uint64(v.GetUint("N"))
	networkConfig.New = isNew
	data, err := toml.Marshal(*networkConfig)
	if err != nil {
		return fmt.Errorf("marshal network config error: %w", err)
	}
	err = v.ReadConfig(bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("readConfig error: %w", err)
	}
	return v.WriteConfig()
}

// GetPidFromPrivFile gets pid from libp2p node priv file
func GetPidFromPrivFile(privPath string) (string, error) {
	data, err := ioutil.ReadFile(privPath)
	if err != nil {
		return "", fmt.Errorf("read private key error: %w", err)
	}
	privKey, err := libp2pcert.ParsePrivateKey(data, crypto2.ECDSA_P256)
	if err != nil {
		return "", fmt.Errorf("parse private key failed: %w", err)
	}

	_, pk, err := crypto.KeyPairFromStdKey(privKey.K)
	if err != nil {
		return "", fmt.Errorf("wrap standard library private key to libp2p keys failed: %w", err)
	}

	pid, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", fmt.Errorf("get peer ID failed: %w", err)
	}

	return pid.String(), nil
}
