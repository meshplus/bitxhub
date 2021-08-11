package solo

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
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

func defaultTimedConfig() TimedGenBlock {
	return TimedGenBlock{
		Enable:       true,
		BlockTimeout: 2 * time.Second,
	}
}

func generateSoloConfig(repoRoot string) (time.Duration, MempoolConfig, *TimedGenBlock, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return 0, MempoolConfig{}, nil, err
	}
	mempoolConf := MempoolConfig{
		BatchSize:      readConfig.SOLO.MempoolConfig.BatchSize,
		PoolSize:       readConfig.SOLO.MempoolConfig.PoolSize,
		TxSliceSize:    readConfig.SOLO.MempoolConfig.TxSliceSize,
		TxSliceTimeout: readConfig.SOLO.MempoolConfig.TxSliceTimeout,
	}
	timedGenBlock := defaultTimedConfig()
	timedGenBlock = TimedGenBlock{
		Enable:       readConfig.TimedGenBlock.Enable,
		BlockTimeout: readConfig.TimedGenBlock.BlockTimeout,
	}

	if timedGenBlock.BlockTimeout < 0 {
		return 0, MempoolConfig{}, nil, fmt.Errorf("blockTimeout must be a positive number. ")
	}
	return readConfig.SOLO.BatchTimeout, mempoolConf, &timedGenBlock, nil
}

func readConfig(repoRoot string) (*SOLOConfig, error) {
	v := viper.New()
	v.SetConfigFile(filepath.Join(repoRoot, "order.toml"))
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	config := &SOLOConfig{}

	if err := v.Unmarshal(config); err != nil {
		return nil, err
	}
	return config, nil
}
