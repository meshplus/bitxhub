package solo

import (
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type SOLOConfig struct {
	SOLO SOLO
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

func generateSoloConfig(repoRoot string) (time.Duration, MempoolConfig, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return 0, MempoolConfig{}, err
	}
	mempoolConf := MempoolConfig{}
	mempoolConf.BatchSize = readConfig.SOLO.MempoolConfig.BatchSize
	mempoolConf.PoolSize = readConfig.SOLO.MempoolConfig.PoolSize
	mempoolConf.TxSliceSize = readConfig.SOLO.MempoolConfig.TxSliceSize
	mempoolConf.TxSliceTimeout = readConfig.SOLO.MempoolConfig.TxSliceTimeout
	return readConfig.SOLO.BatchTimeout, mempoolConf, nil
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
