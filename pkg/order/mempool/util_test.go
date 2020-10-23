package mempool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolIsFull(t *testing.T) {
	ast := assert.New(t)
	mpi, _ := mockMempoolImpl()
	defer cleanTestData()

	isFull := mpi.poolIsFull()
	ast.Equal(false, isFull)
	mpi.txStore.poolSize = DefaultPoolSize
	isFull = mpi.poolIsFull()
	ast.Equal(true, isFull)
}
