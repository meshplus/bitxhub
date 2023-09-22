package repo

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadWriteConsensusConfig(t *testing.T) {
	repoPath := t.TempDir()
	cnf, err := LoadConsensusConfig(repoPath)
	require.Nil(t, err)
	cnf.TxPool.PoolSize = 100
	cnf.TxCache.SetTimeout = Duration(200 * time.Millisecond)
	err = writeConfigWithEnv(path.Join(repoPath, consensusCfgFileName), cnf)
	require.Nil(t, err)
	cnf2, err := LoadConsensusConfig(repoPath)
	require.Nil(t, err)
	require.Equal(t, uint64(100), cnf2.TxPool.PoolSize)
	require.Equal(t, Duration(200*time.Millisecond), cnf2.TxCache.SetTimeout)
}
