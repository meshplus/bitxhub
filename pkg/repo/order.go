package repo

import (
	"path"
	"time"

	"github.com/pkg/errors"

	"github.com/axiomesh/axiom-kit/fileutil"
)

type OrderConfig struct {
	TimedGenBlock TimedGenBlock `mapstructure:"timed_gen_block" toml:"timed_gen_block"`
	Rbft          RBFT          `mapstructure:"rbft" toml:"rbft"`
	Solo          Solo          `mapstructure:"solo" toml:"solo"`
}

type TimedGenBlock struct {
	Enable           bool     `mapstructure:"enable" toml:"enable"`
	NoTxBatchTimeout Duration `mapstructure:"no_tx_batch_timeout" toml:"no_tx_batch_timeout"`
}

type Solo struct {
	Mempool SoloMempool `mapstructure:"mempool" toml:"mempool"`
}

type SoloMempool struct {
	BatchTimeout   Duration `mapstructure:"batch_timeout" toml:"batch_timeout"`
	BatchSize      uint64   `mapstructure:"batch_size" toml:"batch_size"`
	PoolSize       uint64   `mapstructure:"pool_size" toml:"pool_size"`
	TxSliceSize    uint64   `mapstructure:"tx_slice_size" toml:"tx_slice_size"`
	TxSliceTimeout Duration `mapstructure:"tx_slice_timeout" toml:"tx_slice_timeout"`
}

type RBFT struct {
	SetSize             int         `mapstructure:"set_size" toml:"set_size"`
	BatchSize           uint64      `mapstructure:"batch_size" toml:"batch_size"`
	PoolSize            uint64      `mapstructure:"pool_size" toml:"pool_size"`
	CheckInterval       Duration    `mapstructure:"check_interval" toml:"check_interval"`
	ToleranceTime       Duration    `mapstructure:"tolerance_time" toml:"tolerance_time"`
	ToleranceRemoveTime Duration    `mapstructure:"tolerance_remove_time" toml:"tolerance_remove_time"`
	BatchMemLimit       bool        `mapstructure:"batch_mem_limit" toml:"batch_mem_limit"`
	BatchMaxMem         uint64      `mapstructure:"batch_max_mem" toml:"batch_max_mem"`
	VCPeriod            uint64      `mapstructure:"vc_period" toml:"vc_period"`
	CheckpointPeriod    uint64      `mapstructure:"checkpoint_period" toml:"checkpoint_period"`
	Timeout             RBFTTimeout `mapstructure:"timeout" toml:"timeout"`
}

type RBFTTimeout struct {
	SyncState        Duration `mapstructure:"sync_state" toml:"sync_state"`
	SyncInterval     Duration `mapstructure:"sync_interval" toml:"sync_interval"`
	Recovery         Duration `mapstructure:"recovery" toml:"recovery"`
	FirstRequest     Duration `mapstructure:"first_request" toml:"first_request"`
	Batch            Duration `mapstructure:"batch" toml:"batch"`
	Request          Duration `mapstructure:"request" toml:"request"`
	NullRequest      Duration `mapstructure:"null_request" toml:"null_request"`
	ViewChange       Duration `mapstructure:"viewchange" toml:"viewchange"`
	ResendViewChange Duration `mapstructure:"resend_viewchange" toml:"resend_viewchange"`
	CleanViewChange  Duration `mapstructure:"clean_viewchange" toml:"clean_viewchange"`
	Update           Duration `mapstructure:"update" toml:"update"`
	Set              Duration `mapstructure:"set" toml:"set"`
}

func DefaultOrderConfig() *OrderConfig {
	return &OrderConfig{
		TimedGenBlock: TimedGenBlock{
			Enable:           true,
			NoTxBatchTimeout: Duration(2 * time.Second),
		},
		Rbft: RBFT{
			SetSize:             25,
			BatchSize:           500,
			PoolSize:            50000,
			CheckInterval:       Duration(3 * time.Minute),
			ToleranceTime:       Duration(5 * time.Minute),
			ToleranceRemoveTime: Duration(15 * time.Minute),
			BatchMemLimit:       false,
			BatchMaxMem:         10000,
			VCPeriod:            0,
			CheckpointPeriod:    uint64(10),
			Timeout: RBFTTimeout{
				SyncState:        Duration(3 * time.Second),
				SyncInterval:     Duration(1 * time.Minute),
				Recovery:         Duration(15 * time.Second),
				FirstRequest:     Duration(30 * time.Second),
				Batch:            Duration(500 * time.Millisecond),
				Request:          Duration(6 * time.Second),
				NullRequest:      Duration(9 * time.Second),
				ViewChange:       Duration(8 * time.Second),
				ResendViewChange: Duration(10 * time.Second),
				CleanViewChange:  Duration(60 * time.Second),
				Update:           Duration(4 * time.Second),
				Set:              Duration(100 * time.Millisecond),
			},
		},
		Solo: Solo{
			Mempool: SoloMempool{
				BatchTimeout:   Duration(300 * time.Millisecond),
				BatchSize:      200,
				PoolSize:       50000,
				TxSliceSize:    10,
				TxSliceTimeout: Duration(100 * time.Millisecond),
			},
		},
	}
}

func LoadOrderConfig(repoRoot string) (*OrderConfig, error) {
	cfg, err := func() (*OrderConfig, error) {
		cfg := DefaultOrderConfig()
		cfgPath := path.Join(repoRoot, orderCfgFileName)
		existConfig := fileutil.Exist(cfgPath)
		if !existConfig {
			if err := writeConfig(cfgPath, cfg); err != nil {
				return nil, errors.Wrap(err, "failed to build default order config")
			}
		} else {
			if err := readConfig(cfgPath, cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	}()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load network config")
	}
	return cfg, nil
}
