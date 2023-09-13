package repo

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoLoad(t *testing.T) {
	cfg, err := LoadConfig(t.TempDir())
	assert.Nil(t, err)

	cfg2, err := LoadConfig(cfg.RepoRoot)
	assert.Nil(t, err)
	assert.EqualValues(t, cfg, cfg2)

	tests := []struct {
		name      string
		cfg       string
		wantError bool
	}{
		{
			name: "expect int but get decimal",
			cfg: `
	[port]
	jsonrpc = 8882.123
`,
			wantError: true,
		},
		{
			name: "expect int but get string",
			cfg: `
	[port]
	jsonrpc = "123"
`,
			wantError: true,
		},
		{
			name: "correct cfg",
			cfg: `
	[port]
	jsonrpc = 1
`,
			wantError: false,
		},
	}
	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorCfgPath := path.Join(cfg.RepoRoot, fmt.Sprintf("errorCfg_%d.toml", idx))
			err = os.WriteFile(errorCfgPath, []byte(tt.cfg), 0755)
			assert.Nil(t, err)
			err = readConfigFromFile(errorCfgPath, &Config{})
			if tt.wantError {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
