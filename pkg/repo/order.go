package repo

import (
	"path"
	"time"

	"github.com/pkg/errors"

	"github.com/axiomesh/axiom-kit/fileutil"
)

type ReceiveMsgLimiter struct {
	Enable bool  `mapstructure:"enable" toml:"enable"`
	Limit  int64 `mapstructure:"limit" toml:"limit"`
	Burst  int64 `mapstructure:"burst" toml:"burst"`
}

type OrderConfig struct {
	TimedGenBlock TimedGenBlock     `mapstructure:"timed_gen_block" toml:"timed_gen_block"`
	Limit         ReceiveMsgLimiter `mapstructure:"limit" toml:"limit"`
	Mempool       Mempool           `mapstructure:"mempool" toml:"mempool"`
	TxCache       TxCache           `mapstructure:"tx_cache" toml:"tx_cache"`
	Rbft          RBFT              `mapstructure:"rbft" toml:"rbft"`
}

type TimedGenBlock struct {
	Enable           bool     `mapstructure:"enable" toml:"enable"`
	NoTxBatchTimeout Duration `mapstructure:"no_tx_batch_timeout" toml:"no_tx_batch_timeout"`
}

type Mempool struct {
	PoolSize            uint64   `mapstructure:"pool_size" toml:"pool_size"`
	BatchTimeout        Duration `mapstructure:"batch_timeout" toml:"batch_timeout"`
	BatchSize           uint64   `mapstructure:"batch_size" toml:"batch_size"`
	ToleranceTime       Duration `mapstructure:"tolerance_time" toml:"tolerance_time"`
	ToleranceRemoveTime Duration `mapstructure:"tolerance_remove_time" toml:"tolerance_remove_time"`
	BatchMemLimit       bool     `mapstructure:"batch_mem_limit" toml:"batch_mem_limit"`
	BatchMaxMem         uint64   `mapstructure:"batch_max_mem" toml:"batch_max_mem"`
}

type TxCache struct {
	SetSize    int      `mapstructure:"set_size" toml:"set_size"`
	SetTimeout Duration `mapstructure:"set_timeout" toml:"set_timeout"`
}

type RBFT struct {
	CheckInterval    Duration    `mapstructure:"check_interval" toml:"check_interval"`
	VCPeriod         uint64      `mapstructure:"vc_period" toml:"vc_period"`
	CheckpointPeriod uint64      `mapstructure:"checkpoint_period" toml:"checkpoint_period"`
	Timeout          RBFTTimeout `mapstructure:"timeout" toml:"timeout"`
}

type RBFTTimeout struct {
	SyncState        Duration `mapstructure:"sync_state" toml:"sync_state"`
	SyncInterval     Duration `mapstructure:"sync_interval" toml:"sync_interval"`
	Recovery         Duration `mapstructure:"recovery" toml:"recovery"`
	FirstRequest     Duration `mapstructure:"first_request" toml:"first_request"`
	Request          Duration `mapstructure:"request" toml:"request"`
	NullRequest      Duration `mapstructure:"null_request" toml:"null_request"`
	ViewChange       Duration `mapstructure:"viewchange" toml:"viewchange"`
	ResendViewChange Duration `mapstructure:"resend_viewchange" toml:"resend_viewchange"`
	CleanViewChange  Duration `mapstructure:"clean_viewchange" toml:"clean_viewchange"`
	Update           Duration `mapstructure:"update" toml:"update"`
}

func DefaultOrderConfig() *OrderConfig {
	return &OrderConfig{
		TimedGenBlock: TimedGenBlock{
			Enable:           false,
			NoTxBatchTimeout: Duration(2 * time.Second),
		},
		Limit: ReceiveMsgLimiter{
			Enable: false,
			Limit:  10000,
			Burst:  10000,
		},
		Mempool: Mempool{
			PoolSize:            50000,
			BatchTimeout:        Duration(500 * time.Millisecond),
			BatchSize:           500,
			ToleranceTime:       Duration(5 * time.Minute),
			ToleranceRemoveTime: Duration(15 * time.Minute),
		},
		TxCache: TxCache{
			SetSize:    25,
			SetTimeout: Duration(100 * time.Millisecond),
		},
		Rbft: RBFT{
			CheckInterval:    Duration(3 * time.Minute),
			VCPeriod:         0,
			CheckpointPeriod: uint64(10),
			Timeout: RBFTTimeout{
				SyncState:        Duration(3 * time.Second),
				SyncInterval:     Duration(1 * time.Minute),
				Recovery:         Duration(15 * time.Second),
				FirstRequest:     Duration(30 * time.Second),
				Request:          Duration(6 * time.Second),
				NullRequest:      Duration(9 * time.Second),
				ViewChange:       Duration(8 * time.Second),
				ResendViewChange: Duration(10 * time.Second),
				CleanViewChange:  Duration(60 * time.Second),
				Update:           Duration(4 * time.Second),
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
