package repo

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "TestInitialize")
	require.Nil(t, err)
	defer os.RemoveAll(tempDir)

	initialized := Initialized(tempDir)
	require.Equal(t, false, initialized)

	err = Initialize(tempDir)
	require.Nil(t, err)

	require.Equal(t, true, Initialized(tempDir))
}
