package repo

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStoragePath(t *testing.T) {
	p := GetStoragePath("/data", "order")
	assert.Equal(t, p, "/data/storage/order")
	p = GetStoragePath("/data")
	assert.Equal(t, p, "/data/storage")

	_, err := Load("testdata", "")
	require.Nil(t, err)

	_, err = GetAPI("testdata")
	require.Nil(t, err)

	path := GetKeyPath("testdata")
	require.Contains(t, path, KeyName)
}
