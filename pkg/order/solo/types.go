package solo

import (
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/sirupsen/logrus"
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
	Enable       bool          `toml:"enable" json:"enable"`
	BlockTimeout time.Duration `mapstructure:"block_timeout" json:"block_timeout"`
}

type BatchTimer struct {
	logger        logrus.FieldLogger
	timeout       time.Duration      // default timeout of this timer
	isActive      cmap.ConcurrentMap // track all the timers with this timerName if it is active now
	timeoutEventC chan bool
}

// consensusEvent is a type meant to clearly convey that the return type or parameter to a function will be supplied to/from an events.Manager
type consensusEvent interface{}
