package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// handleGetMiddlewareStats retrieves middleware statistics
func (h *Handlers) handleGetMiddlewareStats(c *gin.Context) {
	stats := middleware.GetGlobalMiddlewareStats()

	response.Success(c, "Middleware statistics retrieved successfully", gin.H{
		"stats": stats,
	})
}

// handleUpdateRateLimitConfig updates rate limiting configuration
func (h *Handlers) handleUpdateRateLimitConfig(c *gin.Context) {
	var cfg config.RateLimiterConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Fail(c, "Invalid configuration format", err)
		return
	}

	mgr := middleware.GetGlobalMiddlewareManager()
	mgr.UpdateRateLimitConfig(cfg)

	response.Success(c, "Rate limit configuration updated successfully", nil)
}

// handleUpdateTimeoutConfig updates timeout configuration
func (h *Handlers) handleUpdateTimeoutConfig(c *gin.Context) {
	var cfg config.TimeoutConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Fail(c, "Invalid configuration format", err)
		return
	}

	mgr := middleware.GetGlobalMiddlewareManager()
	mgr.UpdateTimeoutConfig(cfg)

	response.Success(c, "Timeout configuration updated successfully", nil)
}

// handleUpdateCircuitBreakerConfig updates circuit breaker configuration
func (h *Handlers) handleUpdateCircuitBreakerConfig(c *gin.Context) {
	var cfg config.CircuitBreakerConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Fail(c, "Invalid configuration format", err)
		return
	}

	mgr := middleware.GetGlobalMiddlewareManager()
	mgr.UpdateCircuitBreakerConfig(cfg)

	response.Success(c, "Circuit breaker configuration updated successfully", nil)
}

// registerMiddlewareRoutes registers middleware-related routes
func (h *Handlers) registerMiddlewareRoutes(r *gin.RouterGroup) {
	middleware := r.Group("/middleware")
	{
		// Get middleware statistics (requires admin privileges)
		middleware.GET("/stats", h.handleGetMiddlewareStats)

		// Dynamic configuration updates (requires admin privileges)
		middleware.PUT("/rate-limit/config", h.handleUpdateRateLimitConfig)
		middleware.PUT("/timeout/config", h.handleUpdateTimeoutConfig)
		middleware.PUT("/circuit-breaker/config", h.handleUpdateCircuitBreakerConfig)
	}
}
