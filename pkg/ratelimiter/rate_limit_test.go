package ratelimiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(10*time.Second, 5)
	for i := 0; i < 6; i++ {
		if i == 5 {
			ok := limiter.Limit()
			assert.True(t, ok)
			return
		}
		ok := limiter.Limit()
		assert.False(t, ok)
	}
}

func TestNewRateLimiterWithQuantum(t *testing.T) {
	limiter := NewRateLimiterWithQuantum(0, 0, 0)
	ok := limiter.Limit()
	assert.False(t, ok)
}
