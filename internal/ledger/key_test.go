package ledger

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestCompositeKey(t *testing.T) {
	assert.Equal(t, compositeKey(blockKey, 1), []byte("block-1"))
	assert.Equal(t, compositeKey(blockHashKey, "0x112233"), []byte("block-hash-0x112233"))
	assert.Equal(t, compositeKey(transactionKey, "0x112233"), []byte("tx-0x112233"))
}
