package etcdraft

import (
	"path/filepath"
	"time"

	"github.com/coreos/etcd/raft"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type RAFTConfig struct {
	RAFT RAFT
}

type MempoolConfig struct {
	BatchSize      uint64        `mapstructure:"batch_size"`
	PoolSize       uint64        `mapstructure:"pool_size"`
	TxSliceSize    uint64        `mapstructure:"tx_slice_size"`
	TxSliceTimeout time.Duration `mapstructure:"tx_slice_timeout"`
}

type SyncerConfig struct {
	SyncBlocks    uint64 `mapstructure:"sync_blocks"`
	SnapshotCount uint64 `mapstructure:"snapshot_count"`
}

type RAFT struct {
	BatchTimeout              time.Duration `mapstructure:"batch_timeout"`
	CheckInterval             time.Duration `mapstructure:"check_interval"`
	TickTimeout               time.Duration `mapstructure:"tick_timeout"`
	ElectionTick              int           `mapstructure:"election_tick"`
	HeartbeatTick             int           `mapstructure:"heartbeat_tick"`
	MaxSizePerMsg             uint64        `mapstructure:"max_size_per_msg"`
	MaxInflightMsgs           int           `mapstructure:"max_inflight_msgs"`
	CheckQuorum               bool          `mapstructure:"check_quorum"`
	PreVote                   bool          `mapstructure:"pre_vote"`
	DisableProposalForwarding bool          `mapstructure:"disable_proposal_forwarding"`
	MempoolConfig             MempoolConfig `mapstructure:"mempool"`
	SyncerConfig              SyncerConfig  `mapstructure:"syncer"`
}

func defaultRaftConfig() raft.Config {
	return raft.Config{
		ElectionTick:              10,          // ElectionTick is the number of Node.Tick invocations that must pass between elections.(s)
		HeartbeatTick:             1,           // HeartbeatTick is the number of Node.Tick invocations that must pass between heartbeats.(s)
		MaxSizePerMsg:             1024 * 1024, // 1024*1024, MaxSizePerMsg limits the max size of each append message.
		MaxInflightMsgs:           500,         // MaxInflightMsgs limits the max number of in-flight append messages during optimistic replication phase.
		PreVote:                   true,        // PreVote prevents reconnected node from disturbing network.
		CheckQuorum:               true,        // Leader steps down when quorum is not active for an electionTimeout.
		DisableProposalForwarding: true,        // This prevents blocks from being accidentally proposed by followers
	}
}

func generateEtcdRaftConfig(id uint64, repoRoot string, logger logrus.FieldLogger, ram MemoryStorage) (*raft.Config, time.Duration, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return &raft.Config{}, 100 * time.Millisecond, nil
	}
	defaultConfig := defaultRaftConfig()
	defaultConfig.ID = id
	defaultConfig.Storage = ram
	defaultConfig.Logger = logger
	if readConfig.RAFT.ElectionTick > 0 {
		defaultConfig.ElectionTick = readConfig.RAFT.ElectionTick
	}
	if readConfig.RAFT.HeartbeatTick > 0 {
		defaultConfig.HeartbeatTick = readConfig.RAFT.HeartbeatTick
	}
	if readConfig.RAFT.MaxSizePerMsg > 0 {
		defaultConfig.MaxSizePerMsg = readConfig.RAFT.MaxSizePerMsg
	}
	if readConfig.RAFT.MaxInflightMsgs > 0 {
		defaultConfig.MaxInflightMsgs = readConfig.RAFT.MaxInflightMsgs
	}
	defaultConfig.PreVote = readConfig.RAFT.PreVote
	defaultConfig.CheckQuorum = readConfig.RAFT.CheckQuorum
	defaultConfig.DisableProposalForwarding = readConfig.RAFT.DisableProposalForwarding
	return &defaultConfig, readConfig.RAFT.TickTimeout, nil
}

func generateRaftConfig(repoRoot string) (*RAFTConfig, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	return readConfig, nil
}

func readConfig(repoRoot string) (*RAFTConfig, error) {
	v := viper.New()
	v.SetConfigFile(filepath.Join(repoRoot, "order.toml"))
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &RAFTConfig{}

	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
