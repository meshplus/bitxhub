package repo

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"

	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/axiomesh/axiom-kit/types"
)

type NetworkConfig struct {
	ID        uint64          `mapstructure:"id" toml:"id"`
	N         uint64          `mapstructure:"n" toml:"n"`
	New       bool            `mapstructure:"new" toml:"new"`
	LocalAddr string          `mapstructure:"-" toml:"-"`
	Nodes     []*NetworkNodes `mapstructure:"nodes" toml:"nodes"`
	Genesis   Genesis         `mapstructure:"-" toml:"-"`
}

type NetworkNodes struct {
	ID      uint64   `mapstructure:"id" toml:"id"`
	Pid     string   `mapstructure:"pid" toml:"pid"`
	Hosts   []string `mapstructure:"hosts" toml:"hosts"`
	Account string   `mapstructure:"account" toml:"account"`
}

func DefaultNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		ID:  1,
		N:   4,
		New: false,
		Nodes: []*NetworkNodes{
			{
				ID:      1,
				Pid:     defaultNodeIDs[0],
				Hosts:   []string{"/ip4/127.0.0.1/tcp/4001/p2p/"},
				Account: DefaultNodeAddrs[0],
			},
			{
				ID:      2,
				Pid:     defaultNodeIDs[1],
				Hosts:   []string{"/ip4/127.0.0.1/tcp/4002/p2p/"},
				Account: DefaultNodeAddrs[1],
			},
			{
				ID:      3,
				Pid:     defaultNodeIDs[2],
				Hosts:   []string{"/ip4/127.0.0.1/tcp/4003/p2p/"},
				Account: DefaultNodeAddrs[2],
			},
			{
				ID:      4,
				Pid:     defaultNodeIDs[3],
				Hosts:   []string{"/ip4/127.0.0.1/tcp/4004/p2p/"},
				Account: DefaultNodeAddrs[3],
			},
		},
	}
}

func LoadNetworkConfig(repoRoot string, genesis Genesis) (*NetworkConfig, error) {
	cfg, err := func() (*NetworkConfig, error) {
		cfg := DefaultNetworkConfig()
		cfgPath := path.Join(repoRoot, networkCfgFileName)
		existConfig := fileutil.Exist(cfgPath)
		if !existConfig {
			if err := writeConfigWithEnv(cfgPath, cfg); err != nil {
				return nil, errors.Wrap(err, "failed to build default network config")
			}
		} else {
			if err := readConfigFromFile(cfgPath, cfg); err != nil {
				return nil, err
			}
		}

		return cfg, nil
	}()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load network config")
	}
	cfg.Genesis = genesis
	return cfg, nil
}

func (config *NetworkConfig) updateLocalAddr() error {
	checkReaptAddr := make(map[string]uint64)
	for _, node := range config.Nodes {
		if node.ID == config.ID {
			if len(node.Hosts) == 0 {
				return fmt.Errorf("no hosts found by node:%d", node.ID)
			}
			config.LocalAddr = node.Hosts[0]
			addr, err := ma.NewMultiaddr(fmt.Sprintf("%s%s", node.Hosts[0], node.Pid))
			if err != nil {
				return fmt.Errorf("new multiaddr: %w", err)
			}
			config.LocalAddr = strings.ReplaceAll(config.LocalAddr, ma.Split(addr)[0].String(), "/ip4/0.0.0.0")
		}
		if _, ok := checkReaptAddr[node.Hosts[0]]; !ok {
			checkReaptAddr[node.Hosts[0]] = node.ID
		} else {
			return errors.New("reapt address")
		}
	}

	if config.LocalAddr == "" {
		return errors.New("lack of local address")
	}

	idx := strings.LastIndex(config.LocalAddr, "/p2p/")
	if idx == -1 {
		return errors.New("pid is not existed in bootstrap")
	}

	config.LocalAddr = config.LocalAddr[:idx]
	return nil
}

// GetVpInfos gets vp info from network config
func (config *NetworkConfig) GetVpInfos() map[uint64]*types.VpInfo {
	vpNodes := make(map[uint64]*types.VpInfo)
	for _, node := range config.Nodes {
		vpInfo := &types.VpInfo{
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

func RewriteNetworkConfig(repoRoot string, networkConfig *NetworkConfig, infos map[uint64]*types.VpInfo) error {
	nodes := make([]*NetworkNodes, 0, len(infos))
	routers := make([]*types.VpInfo, 0, len(nodes))
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
	return writeConfigWithEnv(path.Join(repoRoot, networkCfgFileName), networkConfig)
}
