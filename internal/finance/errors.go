package finance

import "errors"

var (
	ErrTxsOutOfRange = errors.New("current txs is out of range")

	ErrGasOutOfRange = errors.New("parent gas price is out of range")
)
