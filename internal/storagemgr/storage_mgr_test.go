package storagemgr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	dir := t.TempDir()

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			err := Initialize(dir+tc.kvType, tc.kvType)
			require.Nil(t, err)
		})
	}
}

func TestInitializeWrongType(t *testing.T) {
	dir := t.TempDir()
	err := Initialize(dir, "unsupport")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "unknow kv type unsupport")
}

func TestGet(t *testing.T) {
	dir := t.TempDir()

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			err := Initialize(dir+tc.kvType, tc.kvType)
			require.Nil(t, err)

			s, err := Open(BlockChain)
			require.Nil(t, err)
			require.NotNil(t, s)
		})
	}
}
