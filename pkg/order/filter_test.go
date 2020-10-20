package order

import (
	"testing"

	"github.com/meshplus/bitxhub-kit/log"

	"github.com/meshplus/bitxhub-kit/storage/leveldb"
	"github.com/stretchr/testify/require"
)

func TestReqLookUp_Add(t *testing.T) {
	storage, err := leveldb.New("./build")
	require.Nil(t, err)
	r, err := NewReqLookUp(storage, log.NewWithModule("bloom_filter"))
	require.Nil(t, err)
	r.Add([]byte("abcd"))
	require.Nil(t, err)
	err = r.Build()
	require.Nil(t, err)
	require.True(t, r.LookUp([]byte("abcd")))
}
