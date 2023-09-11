package ledger

import "github.com/prometheus/client_golang/prometheus"

var (
	persistBlockDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "axiom",
		Subsystem: "ledger",
		Name:      "persist_block_duration_second",
		Help:      "The total latency of block persist",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
	})

	blockHeightMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "axiom",
		Subsystem: "ledger",
		Name:      "block_height",
		Help:      "the latest block height in axiom",
	})
)

func init() {
	prometheus.MustRegister(persistBlockDuration)
	prometheus.MustRegister(blockHeightMetric)
}
