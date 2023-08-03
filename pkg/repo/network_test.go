package repo

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestNetworkConfig(t *testing.T) {
	path := t.TempDir()

	genesis := Genesis{
		Admins: lo.Map(defaultNodeAddrs, func(item string, i int) *Admin {
			return &Admin{
				Address: item,
			}
		}),
	}
	cfg, err := LoadNetworkConfig(path, genesis)
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

	_, err = LoadNetworkConfig(path, genesis)
	require.Nil(t, err)
}
