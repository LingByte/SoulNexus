package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const slowHTTPThreshold = 300 * time.Millisecond

// skipAccessLogPaths are high-churn paths that should not emit access logs.
var skipAccessLogPaths = []string{
	"/metrics",
	"/monitor",
	"/static",
	"/favicon.ico",
	"/uploads",
}

func shouldSkipAccessLog(path string) bool {
	for _, p := range skipAccessLogPaths {
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

// LoggerMiddleware logs completed HTTP requests (requires RequestIDMiddleware first).
// log is kept for API compatibility; logging uses logger.*Ctx on the global logger.
func LoggerMiddleware(log *zap.Logger) gin.HandlerFunc {
	_ = log
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method

		c.Next()

		// 读数据的 GET 不打访问日志；变更类请求仍记录。
		if method == http.MethodGet || shouldSkipAccessLog(path) {
			return
		}
		if logger.Lg == nil {
			return
		}

		latency := time.Since(start)
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
		}
		if errMsg := c.Errors.ByType(gin.ErrorTypePrivate).String(); errMsg != "" {
			fields = append(fields, zap.String("errors", errMsg))
		}

		ctx := c.Request.Context()
		logger.InfoCtx(ctx, "http request", fields...)

		if latency > slowHTTPThreshold {
			logger.WarnCtx(ctx, "http slow request", fields...)
		}
	}
}
