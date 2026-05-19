package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

// HealthCheck health check endpoint
func (h *Handlers) HealthCheck(c *gin.Context) {
	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database connection failed"})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database ping failed"})
		return
	}

	// Return health status
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
