package solo

import (
	"time"
)

type RAFTConfig struct {
	RAFT RAFT
}

type MempoolConfig struct {
	BatchSize   uint64 `mapstructure:"batch_size"`
	PoolSize    uint64 `mapstructure:"pool_size"`
	TxSliceSize uint64 `mapstructure:"tx_slice_size"`

	BatchTick      time.Duration `mapstructure:"batch_tick"`
	FetchTimeout   time.Duration `mapstructure:"fetch_timeout"`
	TxSliceTimeout time.Duration `mapstructure:"tx_slice_timeout"`
}

type RAFT struct {
	TickTimeout               time.Duration `mapstructure:"tick_timeout"`
	ElectionTick              int           `mapstructure:"election_tick"`
	HeartbeatTick             int           `mapstructure:"heartbeat_tick"`
	MaxSizePerMsg             uint64        `mapstructure:"max_size_per_msg"`
	MaxInflightMsgs           int           `mapstructure:"max_inflight_msgs"`
	CheckQuorum               bool          `mapstructure:"check_quorum"`
	PreVote                   bool          `mapstructure:"pre_vote"`
	DisableProposalForwarding bool          `mapstructure:"disable_proposal_forwarding"`
	MempoolConfig             MempoolConfig `mapstructure:"mempool"`
}
