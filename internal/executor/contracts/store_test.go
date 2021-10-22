package contracts

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-core/boltvm/mock_stub"
	"github.com/stretchr/testify/assert"
)

func TestStore_Get(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	key0 := "1"
	val0 := "10"
	key1 := "1"

	mockStub.EXPECT().GetObject(key0, gomock.Any()).SetArg(1, val0).Return(true)
	mockStub.EXPECT().GetObject(key1, gomock.Any()).Return(false)

	im := &Store{mockStub}

	res := im.Get(key0)
	assert.True(t, res.Ok)
	assert.Equal(t, val0, string(res.Result))

	res = im.Get(key1)
	assert.False(t, res.Ok)
	assert.Contains(t, string(res.Result), "there is not exist key")
}

func TestStore_Set(t *testing.T) {
	mockCtl := gomock.NewController(t)
	mockStub := mock_stub.NewMockStub(mockCtl)

	mockStub.EXPECT().SetObject(gomock.Any(), gomock.Any())

	im := &Store{mockStub}

	res := im.Set("1", "2")
	assert.True(t, res.Ok)
}
