package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testReadConfig(t *testing.T) {
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

func TestReadNewConfig(t *testing.T) {
	repo := "./testdata"
	cfg, err := loadNetworkConfig(repo)
	assert.Nil(t, err)
	assert.True(t, 4 == cfg.ID)
	assert.True(t, 4 == cfg.N)
	assert.True(t, 4 == len(cfg.Nodes))
	assert.True(t, "/ip4/127.0.0.1/tcp/4001" == cfg.LocalAddr)
	assert.True(t, "/ip4/127.0.0.1/tcp/4002/p2p/QmNRgD6djYJERNpDpHqRn3mxjJ9SYiiGWzExNSy4sEmSNL" == cfg.Nodes[0].Addr)
	assert.True(t, 3 == len(cfg.OtherNodes))
}
