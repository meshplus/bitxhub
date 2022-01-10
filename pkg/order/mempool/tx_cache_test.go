package mempool

import (
	"context"
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
	txCache := NewTxCache(sliceTimeout, 2, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go txCache.ListenEvent(ctx)

	tx := &pb.BxhTransaction{}
	txCache.appendTx(nil)
	ast.Equal(0, len(txCache.txSet), "nil transaction")

	tx = &pb.BxhTransaction{Nonce: 1}
	txCache.appendTx(tx)
	select {
	case txSet := <-txCache.TxSetC:
		ast.Equal(1, len(txSet.Transactions), "post tx set by timeout")
		ast.Equal(0, len(txCache.txSet))
	}
	txCache.stopTxSetTimer()

	txCache.txSetTick = 1 * time.Second
	tx1 := &pb.BxhTransaction{Nonce: 2}
	tx2 := &pb.BxhTransaction{Nonce: 3}
	go txCache.appendTx(tx1)
	go txCache.appendTx(tx2)
	select {
	case txSet := <-txCache.TxSetC:
		ast.Equal(2, len(txSet.Transactions), "post tx set by size")
		ast.Equal(0, len(txCache.txSet))
	}
}
