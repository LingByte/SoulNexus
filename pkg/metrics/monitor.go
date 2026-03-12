package metrics

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	_ "embed"
	"net/http"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Monitor 监控管理器
type Monitor struct {
	metrics       *Metrics
	tracer        *Tracer
	sqlAnalyzer   *SQLAnalyzer
	systemMonitor *SystemMonitor
	mu            sync.RWMutex
	config        *MonitorConfig
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 指标收集配置
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics" default:"true"`

	// 链路追踪配置
	EnableTracing bool `json:"enable_tracing" yaml:"enable_tracing" default:"true"`
	MaxSpans      int  `json:"max_spans" yaml:"max_spans" default:"10000"`

	// SQL分析配置
	EnableSQLAnalysis bool          `json:"enable_sql_analysis" yaml:"enable_sql_analysis" default:"true"`
	MaxQueries        int           `json:"max_queries" yaml:"max_queries" default:"10000"`
	SlowThreshold     time.Duration `json:"slow_threshold" yaml:"slow_threshold" default:"100ms"`

	// 系统监控配置
	EnableSystemMonitor bool          `json:"enable_system_monitor" yaml:"enable_system_monitor" default:"true"`
	MaxStats            int           `json:"max_stats" yaml:"max_stats" default:"1000"`
	MonitorInterval     time.Duration `json:"monitor_interval" yaml:"monitor_interval" default:"30s"`
}

// DefaultMonitorConfig 默认监控配置
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		EnableMetrics:       true,
		EnableTracing:       true,
		MaxSpans:            10000,
		EnableSQLAnalysis:   true,
		MaxQueries:          10000,
		SlowThreshold:       100 * time.Millisecond,
		EnableSystemMonitor: true,
		MaxStats:            1000,
		MonitorInterval:     30 * time.Second,
	}
}

// NewMonitor 创建监控管理器
func NewMonitor(config *MonitorConfig) *Monitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	monitor := &Monitor{
		config: config,
	}

	// 初始化指标收集
	if config.EnableMetrics {
		monitor.metrics = NewMetrics()
	}

	// 初始化链路追踪
	if config.EnableTracing {
		monitor.tracer = NewTracer(config.MaxSpans)
	}

	// 初始化SQL分析
	if config.EnableSQLAnalysis {
		monitor.sqlAnalyzer = NewSQLAnalyzer(config.MaxQueries, config.SlowThreshold)
	}

	// 初始化系统监控
	if config.EnableSystemMonitor {
		monitor.systemMonitor = NewSystemMonitor(config.MaxStats, config.MonitorInterval)
	}

	return monitor
}

// RegisterMonitorUI 绑定监控 UI 和 UI JSON
func RegisterMonitorUI(grp *gin.RouterGroup, api *MonitorAPI) {
	grp.GET("/metric", gin.WrapH(promhttp.Handler()))
	grp.GET("/ui", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(LingEcho.MonitorUIHTML))
	})
	grp.GET("/ui.json", func(c *gin.Context) {
		m := api.monitor
		serverName := config.GlobalConfig.Server.Name
		if serverName == "" {
			serverName = "LingEcho"
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"site": gin.H{
					"name":        serverName + " Monitor",
					"server_name": serverName,
					"description": "系统与业务可观测性面板",
				},
				"capabilities": gin.H{
					"metrics":        m != nil && m.GetMetrics() != nil,
					"tracing":        m != nil && m.GetTracer() != nil,
					"sql_analysis":   m != nil && m.GetSQLAnalyzer() != nil,
					"system_monitor": m != nil && m.GetSystemMonitor() != nil,
				},
				"defaults": gin.H{
					"refresh_seconds": 30,
					"limits": gin.H{
						"system":  50,
						"slow":    50,
						"pattern": 50,
						"traces":  50,
					},
				},
			},
		})
	})
}

// Start 启动监控
func (m *Monitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.systemMonitor != nil {
		m.systemMonitor.Start()
	}
}

// Stop 停止监控
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.systemMonitor != nil {
		m.systemMonitor.Stop()
	}
}

// GetMetrics 获取指标管理器
func (m *Monitor) GetMetrics() *Metrics {
	return m.metrics
}

// GetTracer 获取链路追踪器
func (m *Monitor) GetTracer() *Tracer {
	return m.tracer
}

// GetSQLAnalyzer 获取SQL分析器
func (m *Monitor) GetSQLAnalyzer() *SQLAnalyzer {
	return m.sqlAnalyzer
}

// GetSystemMonitor 获取系统监控器
func (m *Monitor) GetSystemMonitor() *SystemMonitor {
	return m.systemMonitor
}

// StartSpan 开始链路追踪跨度
func (m *Monitor) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	if m.tracer == nil {
		return ctx, nil
	}
	return m.tracer.StartSpan(ctx, name, opts...)
}

// EndSpan 结束链路追踪跨度
func (m *Monitor) EndSpan(span *Span, err error) {
	if m.tracer == nil || span == nil {
		return
	}
	m.tracer.EndSpan(span, err)
}

// RecordSQLQuery 记录SQL查询
func (m *Monitor) RecordSQLQuery(ctx context.Context, sql string, params []interface{}, table, operation string, duration time.Duration, rowsAffected int64, err error) {
	if m.sqlAnalyzer == nil {
		return
	}
	m.sqlAnalyzer.RecordQuery(ctx, sql, params, table, operation, duration, rowsAffected, err)
}

// RecordHTTPRequest 记录HTTP请求指标
func (m *Monitor) RecordHTTPRequest(method, path, status, handler string, duration time.Duration, requestSize, responseSize int64) {
	if m.metrics == nil {
		return
	}
	m.metrics.RecordHTTPRequest(method, path, status, handler, duration, requestSize, responseSize)
}

// RecordDBQuery 记录数据库查询指标
func (m *Monitor) RecordDBQuery(operation, table, sqlType string, duration time.Duration) {
	if m.metrics == nil {
		return
	}
	m.metrics.RecordDBQuery(operation, table, sqlType, duration)
}

// RecordCacheHit 记录缓存命中
func (m *Monitor) RecordCacheHit(cacheType, operation string) {
	if m.metrics == nil {
		return
	}
	m.metrics.RecordCacheHit(cacheType, operation)
}

// RecordCacheMiss 记录缓存未命中
func (m *Monitor) RecordCacheMiss(cacheType, operation string) {
	if m.metrics == nil {
		return
	}
	m.metrics.RecordCacheMiss(cacheType, operation)
}

// SetSystemMetric 设置系统指标
func (m *Monitor) SetSystemMetric(metric, category string, value float64) {
	if m.metrics == nil {
		return
	}
	m.metrics.SetBusinessMetric(metric, category, value)
}

// GetSystemSummary 获取系统摘要
func (m *Monitor) GetSystemSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"timestamp": time.Now(),
		"config":    m.config,
	}

	if m.systemMonitor != nil {
		if sysSummary := m.systemMonitor.GetSystemSummary(); sysSummary != nil {
			summary["system"] = sysSummary
		}
	}

	if m.sqlAnalyzer != nil {
		if sqlStats := m.sqlAnalyzer.GetQueryStats(); sqlStats != nil {
			summary["sql"] = sqlStats
		}
	}

	if m.tracer != nil {
		spans := m.tracer.GetSpans()
		summary["tracing"] = map[string]interface{}{
			"total_spans": len(spans),
		}
	}

	return summary
}

// GetSlowQueries 获取慢查询列表
func (m *Monitor) GetSlowQueries(limit int) []*SQLQuery {
	if m.sqlAnalyzer == nil {
		return nil
	}
	return m.sqlAnalyzer.GetSlowQueries(limit)
}

// GetQueryPatterns 获取查询模式
func (m *Monitor) GetQueryPatterns(limit int) []*QueryPattern {
	if m.sqlAnalyzer == nil {
		return nil
	}
	return m.sqlAnalyzer.GetQueryPatterns(limit)
}

// GetTraceSpans 获取追踪跨度
func (m *Monitor) GetTraceSpans(traceID string) []*Span {
	if m.tracer == nil {
		return nil
	}
	return m.tracer.GetTraceSpans(traceID)
}

// GetSystemStats 获取系统统计
func (m *Monitor) GetSystemStats(limit int) []*SystemStats {
	if m.systemMonitor == nil {
		return nil
	}
	return m.systemMonitor.GetStatsHistory(limit)
}

// GetLatestSystemStats 获取最新系统统计
func (m *Monitor) GetLatestSystemStats() *SystemStats {
	if m.systemMonitor == nil {
		return nil
	}
	return m.systemMonitor.GetLatestStats()
}

// IsEnabled 检查监控是否启用
func (m *Monitor) IsEnabled() bool {
	return m.config.EnableMetrics || m.config.EnableTracing || m.config.EnableSQLAnalysis || m.config.EnableSystemMonitor
}

// GetConfig 获取监控配置
func (m *Monitor) GetConfig() *MonitorConfig {
	return m.config
}

// GetQueriesByTable 按表获取查询
func (m *Monitor) GetQueriesByTable(table string, limit int) []*SQLQuery {
	if m.sqlAnalyzer == nil {
		return nil
	}
	return m.sqlAnalyzer.GetQueriesByTable(table, limit)
}

// GetQueriesByOperation 按操作类型获取查询
func (m *Monitor) GetQueriesByOperation(operation string, limit int) []*SQLQuery {
	if m.sqlAnalyzer == nil {
		return nil
	}
	return m.sqlAnalyzer.GetQueriesByOperation(operation, limit)
}
