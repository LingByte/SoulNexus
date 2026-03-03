package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	cfg := RateLimiterConfig{
		GlobalRPS:   10,
		GlobalBurst: 20,
	}

	rl := NewRateLimiter(cfg)
	assert.NotNil(t, rl)
}

func TestRateLimiter_Allow(t *testing.T) {
	cfg := RateLimiterConfig{
		GlobalRPS:   100,
		GlobalBurst: 200,
		UserRPS:     10,
		UserBurst:   20,
		IPRPS:       50,
		IPBurst:     100,
	}

	rl := NewRateLimiter(cfg)

	// Test normal request
	allowed, reason := rl.Allow(1, "127.0.0.1", "/test")
	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestDefaultRateLimiterConfig(t *testing.T) {
	cfg := DefaultRateLimiterConfig()
	assert.Greater(t, cfg.GlobalRPS, 0)
	assert.Greater(t, cfg.GlobalBurst, 0)
	assert.Greater(t, cfg.UserRPS, 0)
	assert.Greater(t, cfg.UserBurst, 0)
}

func TestGetRateLimiter(t *testing.T) {
	rl := GetRateLimiter()
	assert.NotNil(t, rl)

	// Should return the same instance
	rl2 := GetRateLimiter()
	assert.Equal(t, rl, rl2)
}
