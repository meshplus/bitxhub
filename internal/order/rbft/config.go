package rbft

import (
	"time"

	"go.opentelemetry.io/otel/trace"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-bft/common/metrics/disabled"
	"github.com/axiomesh/axiom-bft/common/metrics/prometheus"
	"github.com/axiomesh/axiom-bft/mempool"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom-ledger/internal/order/common"
)

func defaultRbftConfig() rbft.Config {
	return rbft.Config{
		Applied:                 0,
		SetSize:                 1000,
		SetTimeout:              100 * time.Millisecond,
		BatchTimeout:            200 * time.Millisecond,
		RequestTimeout:          6 * time.Second,
		NullRequestTimeout:      9 * time.Second,
		VcResendTimeout:         8 * time.Second,
		CleanVCTimeout:          60 * time.Second,
		NewViewTimeout:          1 * time.Second,
		SyncStateTimeout:        3 * time.Second,
		SyncStateRestartTimeout: 40 * time.Second,
		FetchCheckpointTimeout:  5 * time.Second,
		FetchViewTimeout:        1 * time.Second,
		CheckPoolTimeout:        100 * time.Second,
		FlowControl:             false,
		FlowControlMaxMem:       0,
		MetricsProv:             &disabled.Provider{},
		Tracer:                  trace.NewNoopTracerProvider().Tracer("axiom-ledger"),
		DelFlag:                 make(chan bool, 10),
		Logger:                  nil,
		NoTxBatchTimeout:        0,
		CheckPoolRemoveTimeout:  30 * time.Minute,
	}
}

func generateRbftConfig(config *common.Config) (rbft.Config, mempool.Config) {
	readConfig := config.Config

	defaultConfig := defaultRbftConfig()
	defaultConfig.GenesisEpochInfo = config.GenesisEpochInfo
	defaultConfig.SelfAccountAddress = config.SelfAccountAddress
	defaultConfig.Applied = config.Applied
	defaultConfig.GenesisBlockDigest = config.GenesisDigest
	defaultConfig.Logger = &common.Logger{FieldLogger: config.Logger}

	if readConfig.TimedGenBlock.NoTxBatchTimeout > 0 {
		defaultConfig.NoTxBatchTimeout = readConfig.TimedGenBlock.NoTxBatchTimeout.ToDuration()
	}
	if readConfig.Rbft.CheckInterval > 0 {
		defaultConfig.CheckPoolTimeout = readConfig.Rbft.CheckInterval.ToDuration()
	}
	if readConfig.Mempool.ToleranceRemoveTime > 0 {
		defaultConfig.CheckPoolRemoveTimeout = readConfig.Mempool.ToleranceRemoveTime.ToDuration()
	}
	if readConfig.TxCache.SetSize > 0 {
		defaultConfig.SetSize = readConfig.TxCache.SetSize
	}
	if readConfig.Rbft.Timeout.SyncState > 0 {
		defaultConfig.SyncStateTimeout = readConfig.Rbft.Timeout.SyncState.ToDuration()
	}
	if readConfig.Rbft.Timeout.SyncInterval > 0 {
		defaultConfig.SyncStateRestartTimeout = readConfig.Rbft.Timeout.SyncInterval.ToDuration()
	}
	if readConfig.Mempool.BatchTimeout > 0 {
		defaultConfig.BatchTimeout = readConfig.Mempool.BatchTimeout.ToDuration()
	}
	if readConfig.Rbft.Timeout.Request > 0 {
		defaultConfig.RequestTimeout = readConfig.Rbft.Timeout.Request.ToDuration()
	}
	if readConfig.Rbft.Timeout.NullRequest > 0 {
		defaultConfig.NullRequestTimeout = readConfig.Rbft.Timeout.NullRequest.ToDuration()
	}
	if readConfig.Rbft.Timeout.ViewChange > 0 {
		defaultConfig.NewViewTimeout = readConfig.Rbft.Timeout.ViewChange.ToDuration()
	}
	if readConfig.Rbft.Timeout.ResendViewChange > 0 {
		defaultConfig.VcResendTimeout = readConfig.Rbft.Timeout.ResendViewChange.ToDuration()
	}
	if readConfig.Rbft.Timeout.CleanViewChange > 0 {
		defaultConfig.CleanVCTimeout = readConfig.Rbft.Timeout.CleanViewChange.ToDuration()
	}
	if readConfig.TxCache.SetTimeout > 0 {
		defaultConfig.SetTimeout = readConfig.TxCache.SetTimeout.ToDuration()
	}
	if readConfig.Rbft.EnableMetrics {
		defaultConfig.MetricsProv = &prometheus.Provider{
			Name: "rbft",
		}
	}
	fn := func(addr string) uint64 {
		return config.GetAccountNonce(types.NewAddressByStr(addr))
	}
	mempoolConf := mempool.Config{
		Logger:              defaultConfig.Logger,
		BatchSize:           defaultConfig.GenesisEpochInfo.ConsensusParams.BlockMaxTxNum,
		PoolSize:            readConfig.Mempool.PoolSize,
		ToleranceTime:       readConfig.Mempool.ToleranceTime.ToDuration(),
		ToleranceRemoveTime: readConfig.Mempool.ToleranceRemoveTime.ToDuration(),
		GetAccountNonce:     fn,
		IsTimed:             defaultConfig.GenesisEpochInfo.ConsensusParams.EnableTimedGenEmptyBlock,
	}
	return defaultConfig, mempoolConf
}
