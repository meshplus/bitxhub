package repo

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestGetStoragePath(t *testing.T) {
	p := GetStoragePath("/data", "order")
	assert.Equal(t, p, "/data/storage/order")
	p = GetStoragePath("/data")
	assert.Equal(t, p, "/data/storage")
}
