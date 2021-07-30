package repo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadKey(t *testing.T) {
	path := "testdata/key.json"
	_, err := LoadKey(path)
	require.Nil(t, err)

	_, err = loadPrivKey("testdata", "")
	require.Nil(t, err)
}
