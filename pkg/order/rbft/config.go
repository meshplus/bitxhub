package rbft

import (
	"fmt"
	"sort"
	"time"

	rbft "github.com/hyperchain/go-hpc-rbft/v2"
	"github.com/hyperchain/go-hpc-rbft/v2/common/metrics/disabled"
	"github.com/hyperchain/go-hpc-rbft/v2/txpool"
	rbfttypes "github.com/hyperchain/go-hpc-rbft/v2/types"
	"github.com/meshplus/bitxhub-core/order"
	"github.com/meshplus/bitxhub-model/pb"
	ethtypes "github.com/meshplus/eth-kit/types"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type RBFTConfig struct {
	Rbft          RBFT
	TimedGenBlock TimedGenBlock `mapstructure:"timed_gen_block"`
}

type TimedGenBlock struct {
	Enable       bool          `toml:"enable" json:"enable"`
	BlockTimeout time.Duration `mapstructure:"block_timeout" json:"block_timeout"`
}

type RBFT struct {
	SetSize             int           `mapstructure:"set_size"`
	BatchSize           uint64        `mapstructure:"batch_size"`
	PoolSize            uint64        `mapstructure:"pool_size"`
	CheckInterval       time.Duration `mapstructure:"check_interval"`
	ToleranceTime       time.Duration `mapstructure:"tolerance_time"`
	ToleranceRemoveTime time.Duration `mapstructure:"tolerance_remove_time"`
	BatchMemLimit       bool          `mapstructure:"batch_mem_limit"`
	BatchMaxMem         uint64        `mapstructure:"batch_max_mem"`
	VCPeriod            uint64        `mapstructure:"vc_period"`
	GetBlockByHeight    func(height uint64) (*pb.Block, error)
	Timeout
}

type Timeout struct {
	SyncState        time.Duration `mapstructure:"sync_state"`
	SyncInterval     time.Duration `mapstructure:"sync_interval"`
	Recovery         time.Duration `mapstructure:"recovery"`
	FirstRequest     time.Duration `mapstructure:"first_request"`
	Batch            time.Duration `mapstructure:"batch"`
	Request          time.Duration `mapstructure:"request"`
	NullRequest      time.Duration `mapstructure:"null_request"`
	ViewChange       time.Duration `mapstructure:"viewchange"`
	ResendViewChange time.Duration `mapstructure:"resend_viewchange"`
	CleanViewChange  time.Duration `mapstructure:"clean_viewchange"`
	Update           time.Duration `mapstructure:"update"`
	Set              time.Duration `mapstructure:"set"`
}

func defaultRbftConfig() rbft.Config[ethtypes.EthTransaction, *ethtypes.EthTransaction] {
	return rbft.Config[ethtypes.EthTransaction, *ethtypes.EthTransaction]{
		ID:        0,
		Hash:      "",
		Hostname:  "",
		EpochInit: 1,
		LatestConfig: &rbfttypes.MetaState{
			Height: 0,
			Digest: "",
		},
		Peers:                   []*rbfttypes.Peer{},
		IsNew:                   false,
		Applied:                 0,
		K:                       10,
		LogMultiplier:           4,
		SetSize:                 1000,
		SetTimeout:              100 * time.Millisecond,
		BatchTimeout:            200 * time.Millisecond,
		RequestTimeout:          6 * time.Second,
		NullRequestTimeout:      9 * time.Second,
		VCPeriod:                0,
		VcResendTimeout:         8 * time.Second,
		CleanVCTimeout:          60 * time.Second,
		NewViewTimeout:          1 * time.Second,
		SyncStateTimeout:        3 * time.Second,
		SyncStateRestartTimeout: 40 * time.Second,
		FetchCheckpointTimeout:  5 * time.Second,
		FetchViewTimeout:        1 * time.Second,
		CheckPoolTimeout:        100 * time.Second,
		External:                nil,
		RequestPool:             nil,
		FlowControl:             false,
		FlowControlMaxMem:       0,
		MetricsProv:             &disabled.Provider{},
		Tracer:                  trace.NewNoopTracerProvider().Tracer("bitxhub"),
		DelFlag:                 make(chan bool, 10),
		Logger:                  nil,
	}
}

func defaultTimedConfig() TimedGenBlock {
	return TimedGenBlock{
		Enable:       true,
		BlockTimeout: 2 * time.Second,
	}
}

func generateRbftConfig(repoRoot string, config *order.Config) (rbft.Config[ethtypes.EthTransaction, *ethtypes.EthTransaction], txpool.Config, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return rbft.Config[ethtypes.EthTransaction, *ethtypes.EthTransaction]{}, txpool.Config{}, nil
	}

	defaultConfig := defaultRbftConfig()
	defaultConfig.ID = config.ID
	defaultConfig.Hash = config.Nodes[config.ID].Pid
	defaultConfig.Hostname = config.Nodes[config.ID].Pid
	defaultConfig.EpochInit = 1
	defaultConfig.LatestConfig = &rbfttypes.MetaState{
		Height: 0,
		Digest: "",
	}
	defaultConfig.Peers, err = generateRbftPeers(config)
	if err != nil {
		return rbft.Config[ethtypes.EthTransaction, *ethtypes.EthTransaction]{}, txpool.Config{}, err
	}
	defaultConfig.IsNew = config.IsNew
	defaultConfig.Applied = config.Applied

	if readConfig.Rbft.CheckInterval > 0 {
		defaultConfig.CheckPoolTimeout = readConfig.Rbft.CheckInterval
	}
	if readConfig.Rbft.SetSize > 0 {
		defaultConfig.SetSize = readConfig.Rbft.SetSize
	}
	if readConfig.Rbft.VCPeriod > 0 {
		defaultConfig.VCPeriod = readConfig.Rbft.VCPeriod
	}
	if readConfig.Rbft.SyncState > 0 {
		defaultConfig.SyncStateTimeout = readConfig.Rbft.Timeout.SyncState
	}
	if readConfig.Rbft.Timeout.SyncInterval > 0 {
		defaultConfig.SyncStateRestartTimeout = readConfig.Rbft.Timeout.SyncInterval
	}
	if readConfig.Rbft.Timeout.Batch > 0 {
		defaultConfig.BatchTimeout = readConfig.Rbft.Timeout.Batch
	}
	if readConfig.Rbft.Timeout.Request > 0 {
		defaultConfig.RequestTimeout = readConfig.Rbft.Timeout.Request
	}
	if readConfig.Rbft.Timeout.NullRequest > 0 {
		defaultConfig.NullRequestTimeout = readConfig.Rbft.Timeout.NullRequest
	}
	if readConfig.Rbft.Timeout.ViewChange > 0 {
		defaultConfig.NewViewTimeout = readConfig.Rbft.Timeout.ViewChange
	}
	if readConfig.Rbft.Timeout.ResendViewChange > 0 {
		defaultConfig.VcResendTimeout = readConfig.Rbft.Timeout.ResendViewChange
	}
	if readConfig.Rbft.Timeout.CleanViewChange > 0 {
		defaultConfig.CleanVCTimeout = readConfig.Rbft.Timeout.CleanViewChange
	}
	if readConfig.Rbft.Timeout.Set > 0 {
		defaultConfig.SetTimeout = readConfig.Rbft.Timeout.Set
	}

	defaultConfig.Logger = &Logger{config.Logger}
	return defaultConfig, txpool.Config{
		K:             int(defaultConfig.K),
		PoolSize:      int(readConfig.Rbft.PoolSize),
		BatchSize:     int(readConfig.Rbft.BatchSize),
		BatchMemLimit: readConfig.Rbft.BatchMemLimit,
		BatchMaxMem:   uint(readConfig.Rbft.BatchMaxMem),
		ToleranceTime: readConfig.Rbft.ToleranceTime,
		MetricsProv:   defaultConfig.MetricsProv,
		Logger:        defaultConfig.Logger,
	}, nil
}

func generateRbftPeers(config *order.Config) ([]*rbfttypes.Peer, error) {
	return sortPeers(config.Nodes)
}

func sortPeers(nodes map[uint64]*pb.VpInfo) ([]*rbfttypes.Peer, error) {
	peers := make([]*rbfttypes.Peer, 0, len(nodes))
	for id, vpInfo := range nodes {
		peers = append(peers, &rbfttypes.Peer{
			ID:       id,
			Hostname: vpInfo.Pid,
			Hash:     vpInfo.Pid,
		})
	}
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].ID < peers[j].ID
	})
	return peers, nil
}

func checkConfig(config *RBFTConfig) error {
	if config.TimedGenBlock.BlockTimeout <= 0 {
		return fmt.Errorf("Illegal parameter, blockTimeout must be a positive number. ")
	}
	return nil
}

type Logger struct {
	logrus.FieldLogger
}

// Trace implements rbft.Logger.
func (lg *Logger) Trace(name string, stage string, content interface{}) {
	lg.Info(name, stage, content)
}

func (lg *Logger) Critical(v ...interface{}) {
	lg.Info(v...)
}

func (lg *Logger) Criticalf(format string, v ...interface{}) {
	lg.Infof(format, v...)
}

func (lg *Logger) Notice(v ...interface{}) {
	lg.Info(v...)
}

func (lg *Logger) Noticef(format string, v ...interface{}) {
	lg.Infof(format, v...)
}
