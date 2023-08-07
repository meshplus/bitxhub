package rbft

import (
	"sort"
	"time"

	rbft "github.com/axiomesh/axiom-bft"
	"github.com/axiomesh/axiom-bft/common/metrics/disabled"
	"github.com/axiomesh/axiom-bft/mempool"
	rbfttypes "github.com/axiomesh/axiom-bft/types"
	"github.com/axiomesh/axiom-kit/types"
	"github.com/axiomesh/axiom/internal/order"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

func defaultRbftConfig() rbft.Config[types.Transaction, *types.Transaction] {
	return rbft.Config[types.Transaction, *types.Transaction]{
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
		CheckPoolRemoveTimeout:  30 * time.Minute,
		External:                nil,
		RequestPool:             nil,
		FlowControl:             false,
		FlowControlMaxMem:       0,
		MetricsProv:             &disabled.Provider{},
		Tracer:                  trace.NewNoopTracerProvider().Tracer("axiom"),
		DelFlag:                 make(chan bool, 10),
		Logger:                  nil,
	}
}

func generateRbftConfig(config *order.Config) (rbft.Config[types.Transaction, *types.Transaction], mempool.Config, error) {
	var err error
	readConfig := config.Config

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
		return rbft.Config[types.Transaction, *types.Transaction]{}, mempool.Config{}, err
	}
	defaultConfig.IsNew = config.IsNew
	defaultConfig.Applied = config.Applied
	defaultConfig.Logger = &Logger{config.Logger}
	defaultConfig.IsTimed = readConfig.TimedGenBlock.Enable

	if readConfig.TimedGenBlock.NoTxBatchTimeout > 0 {
		defaultConfig.NoTxBatchTimeout = readConfig.TimedGenBlock.NoTxBatchTimeout.ToDuration()
	}
	if readConfig.Rbft.CheckInterval > 0 {
		defaultConfig.CheckPoolTimeout = readConfig.Rbft.CheckInterval.ToDuration()
	}
	if readConfig.Rbft.ToleranceRemoveTime > 0 {
		defaultConfig.CheckPoolRemoveTimeout = readConfig.Rbft.ToleranceRemoveTime.ToDuration()
	}
	if readConfig.Rbft.SetSize > 0 {
		defaultConfig.SetSize = readConfig.Rbft.SetSize
	}
	if readConfig.Rbft.VCPeriod > 0 {
		defaultConfig.VCPeriod = readConfig.Rbft.VCPeriod
	}
	if readConfig.Rbft.Timeout.SyncState > 0 {
		defaultConfig.SyncStateTimeout = readConfig.Rbft.Timeout.SyncState.ToDuration()
	}
	if readConfig.Rbft.CheckpointPeriod > 0 {
		defaultConfig.K = readConfig.Rbft.CheckpointPeriod
	}
	if readConfig.Rbft.Timeout.SyncInterval > 0 {
		defaultConfig.SyncStateRestartTimeout = readConfig.Rbft.Timeout.SyncInterval.ToDuration()
	}
	if readConfig.Rbft.Timeout.Batch > 0 {
		defaultConfig.BatchTimeout = readConfig.Rbft.Timeout.Batch.ToDuration()
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
	if readConfig.Rbft.Timeout.Set > 0 {
		defaultConfig.SetTimeout = readConfig.Rbft.Timeout.Set.ToDuration()
	}
	fn := func(addr string) uint64 {
		return config.GetAccountNonce(types.NewAddressByStr(addr))
	}
	mempoolConf := mempool.Config{
		ID:                  config.ID,
		Logger:              defaultConfig.Logger,
		BatchSize:           readConfig.Rbft.BatchSize,
		PoolSize:            readConfig.Rbft.PoolSize,
		BatchMemLimit:       readConfig.Rbft.BatchMemLimit,
		BatchMaxMem:         readConfig.Rbft.BatchMaxMem,
		ToleranceTime:       readConfig.Rbft.ToleranceTime.ToDuration(),
		ToleranceRemoveTime: readConfig.Rbft.ToleranceRemoveTime.ToDuration(),
		GetAccountNonce:     fn,
		IsTimed:             readConfig.TimedGenBlock.Enable,
	}
	return defaultConfig, mempoolConf, nil
}

func generateRbftPeers(config *order.Config) ([]*rbfttypes.Peer, error) {
	return sortPeers(config.Nodes)
}

func sortPeers(nodes map[uint64]*types.VpInfo) ([]*rbfttypes.Peer, error) {
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

type Logger struct {
	logrus.FieldLogger
}

// Trace implements rbft.Logger.
func (lg *Logger) Trace(name string, stage string, content any) {
	lg.Info(name, stage, content)
}

func (lg *Logger) Critical(v ...any) {
	lg.Info(v...)
}

func (lg *Logger) Criticalf(format string, v ...any) {
	lg.Infof(format, v...)
}

func (lg *Logger) Notice(v ...any) {
	lg.Info(v...)
}

func (lg *Logger) Noticef(format string, v ...any) {
	lg.Infof(format, v...)
}
