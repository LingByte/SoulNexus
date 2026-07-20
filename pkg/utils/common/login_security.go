package common

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/validate"
	llmcache "github.com/LingByte/lingllm/cache"
	"go.uber.org/zap"
)

const (
	LoginFailMaxAttempts = 5
	LoginFailLockTTL     = 15 * time.Minute
	LoginFailCountTTL    = 30 * time.Minute
)

func loginFailCountKey(email string) string {
	return "auth:login:fail:" + validate.TrimLower(email)
}

func loginLockKey(email string) string {
	return "auth:login:lock:" + validate.TrimLower(email)
}

// CheckLoginAccountLocked reports whether the email is temporarily locked after repeated failures.
func CheckLoginAccountLocked(email string) bool {
	return llmcache.Exists(context.Background(), loginLockKey(email))
}

// RecordLoginFailure increments the failure counter and locks the account when the threshold is reached.
func RecordLoginFailure(email, clientIP string) {
	email = validate.TrimLower(email)
	if email == "" {
		return
	}
	ctx := context.Background()
	key := loginFailCountKey(email)
	var count int64 = 1
	if v, ok := llmcache.Get(ctx, key); ok {
		switch n := v.(type) {
		case int64:
			count = n + 1
		case int:
			count = int64(n) + 1
		}
	}
	_ = llmcache.Set(ctx, key, count, LoginFailCountTTL)
	if count >= LoginFailMaxAttempts {
		_ = llmcache.Set(ctx, loginLockKey(email), "1", LoginFailLockTTL)
	}
	logger.Warn("login attempt failed",
		zap.String("email", email),
		zap.String("ip", clientIP),
		zap.Int64("failCount", count))
}

// ClearLoginFailures resets failure counters after a successful login.
func ClearLoginFailures(email string) {
	email = validate.TrimLower(email)
	if email == "" {
		return
	}
	ctx := context.Background()
	_ = llmcache.Delete(ctx, loginFailCountKey(email))
	_ = llmcache.Delete(ctx, loginLockKey(email))
}
