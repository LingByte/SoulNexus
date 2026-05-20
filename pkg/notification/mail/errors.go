package mail

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"strings"
)

// permanentMailErrorKeywords are upstream errors that should not be retried on the same channel;
// failover to the next configured channel immediately.
var permanentMailErrorKeywords = []string{
	"额度不足",
	"quota",
	"insufficient",
	"balance",
	"余额不足",
	"余额",
	"欠费",
	"account disabled",
	"account suspended",
	"suspended",
	"disabled",
	"unauthorized",
	"forbidden",
	"invalid api",
	"api key",
	"apikey",
	"密钥",
	"authentication failed",
	"auth failed",
	"blacklist",
	"blacklisted",
	"黑名单",
	"invalid recipient",
	"recipient rejected",
	"收件人",
	"不存在",
	"rate limit exceeded",
	"too many requests",
	"请求过于频繁",
}

// isPermanentMailError reports whether err is a definitive upstream failure where retrying
// the same channel is unlikely to help (quota, auth, invalid address, etc.).
func isPermanentMailError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, kw := range permanentMailErrorKeywords {
		if strings.Contains(msg, strings.ToLower(kw)) {
			return true
		}
	}
	// HTTP status hints embedded in wrapped errors (e.g. sendcloud http 401).
	for _, code := range []string{" 401:", " 403:", " 402:", "http 401", "http 403", "http 402"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

// isTransientMailError reports errors worth retrying on the same channel (network blips, 5xx).
func isTransientMailError(err error) bool {
	if err == nil {
		return false
	}
	if isPermanentMailError(err) {
		return false
	}
	msg := strings.ToLower(err.Error())
	transient := []string{
		"timeout",
		"timed out",
		"temporary",
		"connection reset",
		"connection refused",
		"eof",
		"network",
		"http 5",
		"http 502",
		"http 503",
		"http 504",
	}
	for _, kw := range transient {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return errors.Is(err, context.DeadlineExceeded)
}
