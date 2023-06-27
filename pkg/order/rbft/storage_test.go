package rbft

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDBStore(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)

	err := node.stack.StoreState("test1", []byte("value1"))
	ast.Nil(err)
	err = node.stack.StoreState("test2", []byte("value2"))
	ast.Nil(err)
	err = node.stack.StoreState("test3", []byte("value3"))
	ast.Nil(err)

	_, err = node.stack.ReadStateSet("not found")
	ast.NotNil(err)

	ret, _ := node.stack.ReadStateSet("test")
	ast.Equal(3, len(ret))

	err = node.stack.DelState("test1")
	ast.Nil(err)
	_, err = node.stack.ReadState("test1")
	ast.NotNil(err, "not found")
	ret1, _ := node.stack.ReadState("test2")
	ast.Equal([]byte("value2"), ret1)

	err = node.stack.Destroy()
	ast.Nil(err)
	ret, _ = node.stack.ReadStateSet("test")
	ast.Equal(0, len(ret))
}

func TestFileStore(t *testing.T) {
	ast := assert.New(t)
	ctrl := gomock.NewController(t)
	node := mockNode(ctrl, t)

	err := node.stack.StoreBatchState("test1", []byte("value1"))
	ast.Nil(err)
	err = node.stack.StoreBatchState("test2", []byte("value2"))
	ast.Nil(err)
	err = node.stack.StoreBatchState("test3", []byte("value3"))
	ast.Nil(err)
	ret, _ := node.stack.ReadAllBatchState()
	ast.Equal(3, len(ret))

	err = node.stack.DelBatchState("test1")
	ast.Nil(err)
	ret1, _ := node.stack.ReadBatchState("test1")
	ast.Nil(ret1, "not found")
	ret1, _ = node.stack.ReadBatchState("test2")
	ast.Equal([]byte("value2"), ret1)

	err = node.stack.Destroy()
	ast.Nil(err)
	ret, _ = node.stack.ReadAllBatchState()
	ast.Equal(0, len(ret))
}
