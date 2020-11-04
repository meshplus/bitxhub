package txpool

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPoolContext(t *testing.T) {
	ast := assert.New(t)
	poolContext := newTxPoolContext()
	oldSeed := poolContext.ctx.Value(cancelKey)
	oldCancel := poolContext
	s1 := oldSeed.(string)
	var s2 string
	go func() {
		for {
			select {
			case <- poolContext.ctx.Done():
				s2 = s1
			}
		}
	}()

	poolContext = newTxPoolContext()
	oldCancel.cancel()
	time.Sleep(500 * time.Millisecond)
	ast.NotEqual(s1,s2)
}