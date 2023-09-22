package repo

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	repoPath := t.TempDir()
	cnf, err := LoadConfig(repoPath)
	require.Nil(t, err)
	require.Equal(t, "rbft", cnf.Consensus.Type)
	require.Equal(t, int64(10000), cnf.JsonRPC.Limiter.Capacity)
	require.Equal(t, false, cnf.P2P.Ping.Enable)
	cnf.Consensus.Type = "solo"
	cnf.JsonRPC.Limiter.Capacity = 100
	cnf.P2P.Ping.Enable = true
	err = writeConfigWithEnv(path.Join(repoPath, CfgFileName), cnf)
	require.Nil(t, err)
	cnf2, err := LoadConfig(repoPath)
	require.Nil(t, err)
	require.Equal(t, "solo", cnf2.Consensus.Type)
	require.Equal(t, int64(100), cnf2.JsonRPC.Limiter.Capacity)
	require.Equal(t, true, cnf2.P2P.Ping.Enable)
}
