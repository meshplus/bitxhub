package orderplg

import (
	"fmt"
	"path/filepath"
	"plugin"

	"github.com/meshplus/bitxhub/pkg/order"
)

//Load order plugin
func New(opts ...order.Option) (order.Order, error) {
	config, err := order.GenerateConfig(opts...)
	if err != nil {
		return nil, err
	}
	pluginPath := config.PluginPath

	if !filepath.IsAbs(pluginPath) {
		pluginPath = filepath.Join(config.RepoRoot, pluginPath)
	}

	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("plugin open: %s", err)
	}

	m, err := p.Lookup("NewNode")
	if err != nil {
		return nil, fmt.Errorf("plugin lookup: %s", err)
	}

	NewNode, ok := m.(func(...order.Option) (order.Order, error))
	if !ok {
		return nil, fmt.Errorf("assert NewOrder error")
	}
	return NewNode(opts...)
}
