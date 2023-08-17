package ratelimiter

import (
	"errors"
	"time"

	"github.com/juju/ratelimit"
)

const (
	JOnceTakeCount = 1
	JOncePutCount  = 1
	// DefaultFillInterval = 50 * time.Millisecond
	// DefaultCapacity     = 10000
	// DefaultQuantum      = 500
)

type JRateLimiter struct {
	ratelimit.Bucket
}

// returns a new token bucket that fills at the rate of one token every fillInterval,
// up to the given maximum capacity.Both arguments must be positive.
// The bucket is initially full.
func NewJRateLimiter(fillInterval time.Duration, capacity int64) (*JRateLimiter, error) {
	return NewJRateLimiterWithQuantum(fillInterval, capacity, OncePutCount)
}

// allows the specification of the quantum size - quantum tokens are added every fillInterval.
func NewJRateLimiterWithQuantum(fillInterval time.Duration, capacity, quantum int64) (*JRateLimiter, error) {
	if fillInterval == 0 {
		return nil, errors.New("invalid interval value to init rate_limit")
	}
	if capacity <= 0 {
		return nil, errors.New("invalid capacity value to init rate_limit")
	}
	if quantum <= 0 {
		return nil, errors.New("invalid quantum value to init rate_limit")
	}

	return &JRateLimiter{Bucket: *ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum)}, nil
}

func (l *JRateLimiter) JLimit() bool {
	return l.TakeAvailable(OnceTakeCount) == 0
}
