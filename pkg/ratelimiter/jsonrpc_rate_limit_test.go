package ratelimiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewJRateLimiter(t *testing.T) {
	limiter, _ := NewJRateLimiter(10*time.Second, 5)
	for i := 0; i < 6; i++ {
		if i == 5 {
			ok := limiter.JLimit()
			assert.True(t, ok)
			return
		}
		ok := limiter.JLimit()
		assert.False(t, ok)
	}
}

func TestNewJRateLimiterWithQuantum(t *testing.T) {
	limiter, _ := NewJRateLimiterWithQuantum(50*time.Millisecond, 10000, 500)
	ok := limiter.JLimit()
	assert.False(t, ok)

	_, err := NewJRateLimiterWithQuantum(0, 10000, 500)
	assert.NotNil(t, err)

	_, err = NewJRateLimiterWithQuantum(50*time.Millisecond, -1, 500)
	assert.NotNil(t, err)

	_, err = NewJRateLimiterWithQuantum(50*time.Millisecond, 10000, -1)
	assert.NotNil(t, err)
}
