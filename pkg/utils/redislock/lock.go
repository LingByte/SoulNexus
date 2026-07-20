// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package redislock provides a minimal Redis SET NX lock for single-leader workers.
package redislock

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	mu     sync.RWMutex
	client *redis.Client
)

// InitFromEnv connects using REDIS_ADDR / REDIS_PASSWORD when set.
func InitFromEnv() error {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		return nil
	}
	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return err
	}
	mu.Lock()
	if client != nil {
		_ = client.Close()
	}
	client = c
	mu.Unlock()
	return nil
}

// Enabled reports whether a Redis client is configured.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return client != nil
}

// TryLock attempts to acquire key for ttl. unlock is always non-nil.
func TryLock(ctx context.Context, key string, ttl time.Duration) (unlock func(), ok bool, err error) {
	noop := func() {}
	mu.RLock()
	c := client
	mu.RUnlock()
	if c == nil {
		return noop, true, nil // no Redis → allow (single-node)
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return noop, false, fmt.Errorf("empty lock key")
	}
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	ok, err = c.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return noop, false, err
	}
	if !ok {
		return noop, false, nil
	}
	return func() {
		script := redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)
		_, _ = script.Run(context.Background(), c, []string{key}, token).Result()
	}, true, nil
}
