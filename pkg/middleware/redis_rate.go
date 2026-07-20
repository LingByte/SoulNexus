// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisRateMu     sync.RWMutex
	redisRateClient *redis.Client
)

func initRedisRateBackend(addr, password string) error {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return err
	}
	redisRateMu.Lock()
	if redisRateClient != nil {
		_ = redisRateClient.Close()
	}
	redisRateClient = client
	redisRateMu.Unlock()
	return nil
}

func redisRateEnabled() bool {
	redisRateMu.RLock()
	defer redisRateMu.RUnlock()
	return redisRateClient != nil
}

// redisAllow is a minimal fixed-window counter shared across replicas.
func redisAllow(key string, limit int, window time.Duration) bool {
	if limit <= 0 {
		return true
	}
	redisRateMu.RLock()
	client := redisRateClient
	redisRateMu.RUnlock()
	if client == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	fullKey := "rl:" + key
	n, err := client.Incr(ctx, fullKey).Result()
	if err != nil {
		return true // fail open
	}
	if n == 1 {
		_ = client.Expire(ctx, fullKey, window).Err()
	}
	return n <= int64(limit)
}
