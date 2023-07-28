package solo

import (
	"time"

	cmap "github.com/orcaman/concurrent-map"
)

type SOLOConfig struct {
	SOLO          SOLO
	TimedGenBlock TimedGenBlock `mapstructure:"timed_gen_block"`
}

type SOLO struct {
	BatchTimeout  time.Duration `mapstructure:"batch_timeout"`
	MempoolConfig MempoolConfig `mapstructure:"mempool"`
}

type MempoolConfig struct {
	BatchSize      uint64        `mapstructure:"batch_size"`
	PoolSize       uint64        `mapstructure:"pool_size"`
	TxSliceSize    uint64        `mapstructure:"tx_slice_size"`
	TxSliceTimeout time.Duration `mapstructure:"tx_slice_timeout"`
}

type TimedGenBlock struct {
	Enable           bool          `toml:"enable" json:"enable"`
	NoTxBatchTimeout time.Duration `mapstructure:"no_tx_batch_timeout" json:"no_tx_batch_timeout"`
}

// singleTimer manages timer with the same timer name, which, we allow different timer with the same timer name, such as:
// we allow several request timers at the same time, each timer started after received a new request batch
type singleTimer struct {
	timerName string             // the unique timer name
	timeout   time.Duration      // default timeout of this timer
	isActive  cmap.ConcurrentMap // track all the timers with this timerName if it is active now
}

// consensusEvent is a type meant to clearly convey that the return type or parameter to a function will be supplied to/from an events.Manager
type consensusEvent interface{}
