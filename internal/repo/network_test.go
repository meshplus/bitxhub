package repo

import (
	"github.com/spf13/viper"
	"testing"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/require"
)

func TestNetworkConfig(t *testing.T) {
	path := "./testdata"
	cfg, err := loadNetworkConfig(viper.New(), path, Genesis{
		Admins: []*Admin{
			&Admin{
				Address: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
				Weight:  1,
			},
			&Admin{
				Address: "0x79a1215469FaB6f9c63c1816b45183AD3624bE34",
				Weight:  1,
			},
			&Admin{
				Address: "0x97c8B516D19edBf575D72a172Af7F418BE498C37",
				Weight:  1,
			},
			&Admin{
				Address: "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8",
				Weight:  1,
			},
		},
	})

	require.Nil(t, err)
	peers, err := cfg.GetNetworkPeers()
	require.Nil(t, err)
	require.Equal(t, 4, len(peers))

	accounts := cfg.GetVpGenesisAccount()
	require.Equal(t, 4, len(accounts))

	vpAccounts := cfg.GetVpAccount()
	require.Equal(t, 4, len(vpAccounts))

	vpInfos := cfg.GetVpInfos()
	require.Equal(t, 4, len(vpInfos))
}

func TestRewriteNetworkConfig(t *testing.T) {
	infos := make(map[uint64]*pb.VpInfo, 0)
	{
		infos[1] = &pb.VpInfo{
			Id:      1,
			Hosts:   []string{"/ip4/127.0.0.1/tcp/4001/p2p/"},
			Pid:     "QmQUcDYCtqbpn5Nhaw4FAGxQaSSNvdWfAFcpQT9SPiezbS",
			Account: "0xc7F999b83Af6DF9e67d0a37Ee7e900bF38b3D013",
		}
		infos[2] = &pb.VpInfo{
			Id:      2,
			Hosts:   []string{"/ip4/127.0.0.1/tcp/4002/p2p/"},
			Pid:     "QmQW3bFn8XX1t4W14Pmn37bPJUpUVBrBjnPuBZwPog3Qdy",
			Account: "0x79a1215469FaB6f9c63c1816b45183AD3624bE34",
		}
		infos[3] = &pb.VpInfo{
			Id:      3,
			Hosts:   []string{"/ip4/127.0.0.1/tcp/4003/p2p/"},
			Pid:     "QmXi58fp9ZczF3Z5iz1yXAez3Hy5NYo1R8STHWKEM9XnTL",
			Account: "0x97c8B516D19edBf575D72a172Af7F418BE498C37",
		}
		infos[4] = &pb.VpInfo{
			Id:      4,
			Hosts:   []string{"/ip4/127.0.0.1/tcp/4004/p2p/"},
			Pid:     "QmbmD1kzdsxRiawxu7bRrteDgW1ituXupR8GH6E2EUAHY4",
			Account: "0xc0Ff2e0b3189132D815b8eb325bE17285AC898f8",
		}
	}
	err := RewriteNetworkConfig("./testdata", infos, false)
	require.Nil(t, err)

	_, err = GetPidFromPrivFile("testdata/certs/node.priv")
	require.Nil(t, err)
}
