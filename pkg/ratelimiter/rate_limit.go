package ratelimiter

import (
	"time"

	"github.com/juju/ratelimit"
)

const (
	OnceTakeCount       = 1
	OncePutCount        = 1
	DefaultFillInterval = 50 * time.Millisecond
	DefaultCapacity     = 10000
	DefaultQuantum      = 500
)

type RateLimiter struct {
	ratelimit.Bucket
}

// returns a new token bucket that fills at the rate of one token every fillInterval,
// up to the given maximum capacity.Both arguments must be positive.
// The bucket is initially full.
func NewRateLimiter(fillInterval time.Duration, capacity int64) *RateLimiter {
	return NewRateLimiterWithQuantum(fillInterval, capacity, OncePutCount)
}

// allows the specification of the quantum size - quantum tokens are added every fillInterval.
func NewRateLimiterWithQuantum(fillInterval time.Duration, capacity, quantum int64) *RateLimiter {
	if fillInterval == 0 {
		fillInterval = DefaultFillInterval
	}
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	if quantum <= 0 {
		quantum = DefaultQuantum
	}

	return &RateLimiter{*ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum)}
}

func (l *RateLimiter) Limit() bool {
	return l.TakeAvailable(OnceTakeCount) == 0
}
