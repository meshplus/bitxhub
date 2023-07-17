package adaptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRBFTTXPoolAdaptor_IsRequestsExist(t *testing.T) {
	a := &RBFTTXPoolAdaptor{}

	txs := [][]byte{{}, {}}
	a.CheckSigns(txs)
	res := a.IsRequestsExist(txs)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, false, res[0])
	assert.Equal(t, false, res[1])
}
