package mempool

import (
	"testing"
	"time"

	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/stretchr/testify/assert"
)

func TestAppendTx(t *testing.T) {
	ast := assert.New(t)
	logger := log.NewWithModule("consensus")
	sliceTimeout := 1 * time.Millisecond
	txCache := newTxCache(sliceTimeout, 2, logger)
	go txCache.listenEvent()

	tx := &pb.Transaction{}
	txCache.appendTx(nil)
	ast.Equal(0, len(txCache.txSet), "nil transaction")

	tx = &pb.Transaction{Nonce: 1}
	txCache.appendTx(tx)
	select {
	case txSet := <-txCache.txSetC:
		ast.Equal(1, len(txSet.TxList), "post tx set by timeout")
		ast.Equal(0, len(txCache.txSet))
	}
	txCache.stopTxSetTimer()

	txCache.txSetTick = 1 * time.Second
	tx1 := &pb.Transaction{Nonce: 2}
	tx2 := &pb.Transaction{Nonce: 3}
	go txCache.appendTx(tx1)
	go txCache.appendTx(tx2)
	select {
	case txSet := <-txCache.txSetC:
		ast.Equal(2, len(txSet.TxList), "post tx set by size")
		ast.Equal(0, len(txCache.txSet))
	}
	// test exit txCache
	close(txCache.close)
}