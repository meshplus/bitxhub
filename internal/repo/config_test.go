package repo

import (
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadNetworkConfig(t *testing.T) {
	path := "../../config/network.toml"
	cfg := &NetworkConfig{}
	err := ReadConfig(viper.New(), path, "toml", cfg)
	assert.Nil(t, err)

	assert.True(t, 1 == cfg.ID)
	assert.True(t, 4 == cfg.N)
	assert.True(t, 4 == len(cfg.Nodes))

	for i, node := range cfg.Nodes {
		assert.True(t, uint64(i+1) == node.ID)
	}

}

func TestReadConfig(t *testing.T) {
	_, err := DefaultConfig()
	require.Nil(t, err)

	path := "../../config/bitxhub.toml"
	cfg := &Config{}
	v := viper.New()
	err = ReadConfig(v, path, "toml", cfg)
	assert.Nil(t, err)

	_, err = cfg.Bytes()
	require.Nil(t, err)

	_, err = UnmarshalConfig(v, "../../config", "")
	require.Nil(t, err)

	pathRoot, err := PathRoot()
	require.Nil(t, err)
	dir, err := homedir.Expand(defaultPathRoot)
	require.Nil(t, err)
	require.Equal(t, dir, pathRoot)

	rootWithDefault, err := PathRootWithDefault("../../config")
	require.Nil(t, err)
	require.Equal(t, "../../config", rootWithDefault)
}
