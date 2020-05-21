package executor

import "github.com/prometheus/client_golang/prometheus"

var (
	applyTxsDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "bitxhub",
		Subsystem: "executor",
		Name:      "apply_transactions_duration_seconds",
		Help:      "The total latency of transactions apply",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 14),
	})
	executeBlockDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "bitxhub",
		Subsystem: "executor",
		Name:      "execute_block_duration_second",
		Help:      "The total latency of block execute",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
	})
	calcMerkleDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "bitxhub",
		Subsystem: "executor",
		Name:      "calc_merkle_duration_seconds",
		Help:      "The total latency of merkle calc",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
	})
	calcBlockSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "bitxhub",
		Subsystem: "executor",
		Name:      "calc_block_size",
		Help:      "The size of current block calc",
		Buckets:   prometheus.ExponentialBuckets(1024, 2, 12),
	})
)

func init() {
	prometheus.MustRegister(applyTxsDuration)
	prometheus.MustRegister(calcMerkleDuration)
	prometheus.MustRegister(calcBlockSize)
	prometheus.MustRegister(executeBlockDuration)
}
