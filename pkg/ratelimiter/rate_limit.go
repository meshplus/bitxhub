package ratelimiter

import (
	"fmt"
	"time"

	"github.com/juju/ratelimit"
)

const (
	OnceTakeCount = 1
	OncePutCount  = 1
	// DefaultFillInterval = 50 * time.Millisecond
	// DefaultCapacity     = 10000
	// DefaultQuantum      = 500
)

type RateLimiter struct {
	ratelimit.Bucket
}

// returns a new token bucket that fills at the rate of one token every fillInterval,
// up to the given maximum capacity.Both arguments must be positive.
// The bucket is initially full.
func NewRateLimiter(fillInterval time.Duration, capacity int64) (*RateLimiter, error) {
	return NewRateLimiterWithQuantum(fillInterval, capacity, OncePutCount)
}

// allows the specification of the quantum size - quantum tokens are added every fillInterval.
func NewRateLimiterWithQuantum(fillInterval time.Duration, capacity, quantum int64) (*RateLimiter, error) {
	if fillInterval == 0 {
		return nil, fmt.Errorf("invalid interval value to init rate_limit")
	}
	if capacity <= 0 {
		return nil, fmt.Errorf("invalid capacity value to init rate_limit")
	}
	if quantum <= 0 {
		return nil, fmt.Errorf("invalid quantum value to init rate_limit")
	}

	return &RateLimiter{*ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum)}, nil
}

func (l *RateLimiter) Limit() bool {
	return l.TakeAvailable(OnceTakeCount) == 0
}
