package adaptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestInitialize(t *testing.T) {
	ast := assert.New(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			repoRoot := t.TempDir()
			wrapper, err := newStorageWrapper(repoRoot, tc.kvType)
			ast.Nil(err)
			ast.NotNil(wrapper)
			wrapper, err = newStorageWrapper(repoRoot, tc.kvType)
			ast.Nil(wrapper)
			ast.NotNil(err)
		},
		)
	}
}

func TestInitializeWrongType(t *testing.T) {
	ast := assert.New(t)
	repoRoot := t.TempDir()
	wrapper, err := newStorageWrapper(repoRoot, "unsupport")
	ast.NotNil(err)
	ast.Nil(wrapper)
	ast.Contains(err.Error(), "unknow kv type unsupport")
}

func TestDBStore(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			adaptor := mockAdaptor(ctrl, tc.kvType, t)

			err := adaptor.StoreState("test1", []byte("value1"))
			ast.Nil(err)
			err = adaptor.StoreState("test2", []byte("value2"))
			ast.Nil(err)
			err = adaptor.StoreState("test3", []byte("value3"))
			ast.Nil(err)

			_, err = adaptor.ReadStateSet("not found")
			ast.NotNil(err)

			ret, _ := adaptor.ReadStateSet("test")
			ast.Equal(3, len(ret))

			err = adaptor.DelState("test1")
			ast.Nil(err)
			_, err = adaptor.ReadState("test1")
			ast.NotNil(err, "not found")
			ret1, _ := adaptor.ReadState("test2")
			ast.Equal([]byte("value2"), ret1)

			err = adaptor.Destroy("t")
			ast.Nil(err)
		})
	}
}

func TestFileStore(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)

	testcase := map[string]struct {
		kvType string
	}{
		"leveldb": {kvType: "leveldb"},
		"pebble":  {kvType: "pebble"},
	}

	for name, tc := range testcase {
		t.Run(name, func(t *testing.T) {
			adaptor := mockAdaptor(ctrl, tc.kvType, t)

			err := adaptor.StoreBatchState("test1", []byte("value1"))
			ast.Nil(err)
			err = adaptor.StoreBatchState("test2", []byte("value2"))
			ast.Nil(err)
			err = adaptor.StoreBatchState("test3", []byte("value3"))
			ast.Nil(err)
			ret, _ := adaptor.ReadAllBatchState()
			ast.Equal(3, len(ret))

			err = adaptor.DelBatchState("test1")
			ast.Nil(err)
			ret1, _ := adaptor.ReadBatchState("test1")
			ast.Nil(ret1, "not found")
			ret1, _ = adaptor.ReadBatchState("test2")
			ast.Equal([]byte("value2"), ret1)

			err = adaptor.Destroy("t")
			ast.Nil(err)
			ret, _ = adaptor.ReadAllBatchState()
			ast.Equal(0, len(ret))
		})
	}
}
