package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig(t *testing.T) {
	path := "../../config/network.toml"
	cfg := &NetworkConfig{}
	err := ReadConfig(path, "toml", cfg)
	assert.Nil(t, err)

	assert.True(t, 1 == cfg.ID)
	assert.True(t, 4 == cfg.N)
	assert.True(t, 4 == len(cfg.Nodes))

	for i, node := range cfg.Nodes {
		assert.True(t, uint64(i+1) == node.ID)
	}
}
