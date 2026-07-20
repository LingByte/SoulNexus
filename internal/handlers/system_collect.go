package handlers

import (
	"database/sql"

	voiceMetrics "github.com/LingByte/SoulNexus/pkg/voice/metrics"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) collectBusinessMetrics(db *gorm.DB) gin.H {
	out := gin.H{
		"http": gin.H{
			"requestsTotal": voiceMetrics.Default.CounterTotal(voiceMetrics.MetricHTTPRequestsTotal),
			"errors5xx": voiceMetrics.Default.CounterWithLabels(voiceMetrics.MetricHTTPErrorsTotal,
				map[string]string{"kind": "5xx"}),
			"errors4xx": voiceMetrics.Default.CounterWithLabels(voiceMetrics.MetricHTTPErrorsTotal,
				map[string]string{"kind": "4xx"}),
			"latencyMs": voiceMetrics.Default.SummaryQuantiles(voiceMetrics.MetricHTTPDurationMs),
		},
	}

	if db != nil {
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			out["database"] = sqlDBStats(sqlDB)
		}
	}
	if h.kbWorker != nil {
		st := h.kbWorker.Stats()
		out["knowledgeWorker"] = gin.H{
			"queued": st.Queued, "running": st.Running, "unfinished": st.Unfinished,
		}
	}
	return out
}

func sqlDBStats(db *sql.DB) gin.H {
	st := db.Stats()
	return gin.H{
		"openConnections": st.OpenConnections,
		"inUse":           st.InUse,
		"idle":            st.Idle,
		"waitCount":       st.WaitCount,
		"waitDurationMs":  st.WaitDuration.Milliseconds(),
		"maxOpen":         st.MaxOpenConnections,
	}
}
