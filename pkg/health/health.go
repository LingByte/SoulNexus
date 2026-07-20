// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package health provides liveness and readiness probes for orchestrators.
package health

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Status is the JSON body for probe endpoints.
type Status struct {
	Status string            `json:"status" example:"ok" doc:"Overall probe status"`
	Checks map[string]string `json:"checks,omitempty" doc:"Per-dependency check results"`
}

// Checker is an optional readiness dependency (Redis, object store, …).
type Checker func(ctx context.Context) error

var (
	ready    atomic.Bool
	checkMu  sync.RWMutex
	checkers = map[string]Checker{}
)

// MarkReady signals that the process finished startup wiring (workers, etc.).
func MarkReady() { ready.Store(true) }

// MarkNotReady flips readiness off (e.g. during shutdown).
func MarkNotReady() { ready.Store(false) }

// IsReady reports whether MarkReady has been called.
func IsReady() bool { return ready.Load() }

// RegisterChecker adds or replaces a named readiness check.
func RegisterChecker(name string, fn Checker) {
	if name == "" || fn == nil {
		return
	}
	checkMu.Lock()
	checkers[name] = fn
	checkMu.Unlock()
}

// ClearCheckers removes all dynamic checkers (tests).
func ClearCheckers() {
	checkMu.Lock()
	checkers = map[string]Checker{}
	checkMu.Unlock()
}

// RegisterGin mounts /healthz (liveness) and /readyz (readiness) on the engine root.
// Probes are intentionally outside /api so load balancers can hit them without auth.
func RegisterGin(r *gin.Engine, db *gorm.DB) {
	r.GET("/healthz", LivenessGin())
	r.GET("/livez", LivenessGin())
	r.GET("/health", LivenessGin())
	r.GET("/readyz", ReadinessGin(db))
	r.GET("/ready", ReadinessGin(db))
}

// LivenessGin returns 200 when the process is alive (no dependency checks).
func LivenessGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, Status{Status: "ok"})
	}
}

// ReadinessGin returns 200 when startup completed, the database responds, and
// any registered checkers succeed.
func ReadinessGin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		checks := map[string]string{}
		ok := true

		if !ready.Load() {
			checks["startup"] = "not_ready"
			ok = false
		} else {
			checks["startup"] = "ok"
		}

		if db == nil {
			checks["database"] = "unavailable"
			ok = false
		} else {
			sqlDB, err := db.DB()
			if err != nil {
				checks["database"] = err.Error()
				ok = false
			} else {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
				defer cancel()
				if err := sqlDB.PingContext(ctx); err != nil {
					checks["database"] = err.Error()
					ok = false
				} else {
					checks["database"] = "ok"
				}
			}
		}

		checkMu.RLock()
		names := make([]string, 0, len(checkers))
		for name := range checkers {
			names = append(names, name)
		}
		local := make(map[string]Checker, len(checkers))
		for _, name := range names {
			local[name] = checkers[name]
		}
		checkMu.RUnlock()

		for name, fn := range local {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			err := fn(ctx)
			cancel()
			if err != nil {
				checks[name] = err.Error()
				ok = false
			} else {
				checks[name] = "ok"
			}
		}

		body := Status{Status: "ok", Checks: checks}
		if !ok {
			body.Status = "not_ready"
			c.JSON(http.StatusServiceUnavailable, body)
			return
		}
		c.JSON(http.StatusOK, body)
	}
}
