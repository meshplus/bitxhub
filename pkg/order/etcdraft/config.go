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
	BatchSize uint64 `mapstructure:"batch_size"`
	PoolSize  uint64 `mapstructure:"pool_size"`
	TxSliceSize uint64 `mapstructure:"tx_slice_size"`

	BatchTick time.Duration `mapstructure:"batch_tick"`
	FetchTimeout time.Duration `mapstructure:"fetch_timeout"`
	TxSliceTimeout time.Duration `mapstructure:"tx_slice_timeout"`
}

type RAFT struct {
	ElectionTick              int           `mapstructure:"election_tick"`
	HeartbeatTick             int           `mapstructure:"heartbeat_tick"`
	MaxSizePerMsg             uint64        `mapstructure:"max_size_per_msg"`
	MaxInflightMsgs           int           `mapstructure:"max_inflight_msgs"`
	CheckQuorum               bool          `mapstructure:"check_quorum"`
	PreVote                   bool          `mapstructure:"pre_vote"`
	DisableProposalForwarding bool          `mapstructure:"disable_proposal_forwarding"`
	MempoolConfig             MempoolConfig `mapstructure:"mempool"`
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

func generateRaftConfig(id uint64, repoRoot string, logger logrus.FieldLogger, ram MemoryStorage) (*raft.Config, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return &raft.Config{}, nil
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
	return &defaultConfig, nil
}

func generateMempoolConfig(repoRoot string) (*MempoolConfig, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	mempoolConf := &MempoolConfig{}
	mempoolConf.BatchSize = readConfig.RAFT.MempoolConfig.BatchSize
	mempoolConf.PoolSize = readConfig.RAFT.MempoolConfig.PoolSize
	mempoolConf.TxSliceSize = readConfig.RAFT.MempoolConfig.TxSliceSize
	mempoolConf.BatchTick = readConfig.RAFT.MempoolConfig.BatchTick
	mempoolConf.FetchTimeout = readConfig.RAFT.MempoolConfig.FetchTimeout
	mempoolConf.TxSliceTimeout = readConfig.RAFT.MempoolConfig.TxSliceTimeout
	return mempoolConf, nil
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
