package repo

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadWriteOrderConfig(t *testing.T) {
	repoPath := t.TempDir()
	cnf, err := LoadOrderConfig(repoPath)
	require.Nil(t, err)
	cnf.Mempool.PoolSize = 100
	cnf.Rbft.CheckpointPeriod = 20
	cnf.TxCache.SetTimeout = Duration(200 * time.Millisecond)
	err = writeConfigWithEnv(path.Join(repoPath, orderCfgFileName), cnf)
	require.Nil(t, err)
	cnf2, err := LoadOrderConfig(repoPath)
	require.Nil(t, err)
	require.Equal(t, uint64(100), cnf2.Mempool.PoolSize)
	require.Equal(t, uint64(20), cnf2.Rbft.CheckpointPeriod)
	require.Equal(t, Duration(200*time.Millisecond), cnf2.TxCache.SetTimeout)
}
