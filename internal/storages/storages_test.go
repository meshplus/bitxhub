package storages

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestInitialize")
	require.Nil(t, err)

	err = Initialize(dir)
	require.Nil(t, err)

	// Initialize twice
	err = Initialize(dir)
	require.Contains(t, err.Error(), "create blockchain storage: resource temporarily unavailable")
}

func TestGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestGet")
	require.Nil(t, err)

	err = Initialize(dir)
	require.Nil(t, err)

	s, err := Get(BlockChain)
	require.Nil(t, err)
	require.NotNil(t, s)

	s, err = Get("WrongName")
	require.NotNil(t, err)
	require.Nil(t, s)
}
