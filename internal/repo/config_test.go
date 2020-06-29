package repo

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p"
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

func testLibP2p(t *testing.T) {
	// create a background context (i.e. one that never cancels)
	ctx := context.Background()

	// start a libp2p node with default settings
	node, err := libp2p.New(ctx)
	if err != nil {
		panic(err)
	}

	// print the node's listening addresses
	fmt.Println("Listen addresses:", node.Addrs())
}

func testReadNewConfig(t *testing.T) {
	repo := "./testdata"
	// cfg, err := loadNetworkConfigNew(repo)
	cfg, err := loadNetworkConfig(repo)
	assert.Nil(t, err)
	PrintNetworkConfig(cfg)
}
