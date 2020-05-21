package ledger

import "github.com/prometheus/client_golang/prometheus"

var (
	PersistBlockDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "bitxhub",
		Subsystem: "ledger",
		Name:      "persist_block_duration_second",
		Help:      "The total latency of block persist",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
	})
)

func init() {
	prometheus.MustRegister(PersistBlockDuration)
}
