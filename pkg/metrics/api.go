package metrics

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// MonitorAPI 监控API处理器
type MonitorAPI struct {
	monitor *Monitor
}

// NewMonitorAPI 创建监控API处理器
func NewMonitorAPI(monitor *Monitor) *MonitorAPI {
	return &MonitorAPI{
		monitor: monitor,
	}
}

// RegisterRoutes 注册监控API路由
func (api *MonitorAPI) RegisterRoutes(r *gin.RouterGroup) {
	// 系统概览
	r.GET("/overview", api.GetOverview)

	// 系统监控
	r.GET("/system", api.GetSystemStats)
	r.GET("/system/latest", api.GetLatestSystemStats)

	// SQL分析
	r.GET("/sql/slow", api.GetSlowQueries)
	r.GET("/sql/patterns", api.GetQueryPatterns)
	r.GET("/sql/stats", api.GetSQLStats)
	r.GET("/sql/table/:table", api.GetQueriesByTable)
	r.GET("/sql/operation/:operation", api.GetQueriesByOperation)

	// 链路追踪
	r.GET("/traces", api.GetTraces)
	r.GET("/traces/:traceID", api.GetTraceDetail)

	// 指标数据
	r.GET("/metrics", api.GetMetrics)
	r.GET("/metrics/prometheus", api.GetPrometheusMetrics)

	RegisterMonitorUI(r, api)
}

// GetOverview 获取系统概览
func (api *MonitorAPI) GetOverview(c *gin.Context) {
	summary := api.monitor.GetSystemSummary()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// GetSystemStats 获取系统统计
func (api *MonitorAPI) GetSystemStats(c *gin.Context) {
	if api.monitor.GetSystemMonitor() == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
		return
	}
	if since := c.Query("since"); since != "" {
		ms, _ := strconv.ParseInt(since, 10, 64)
		all := api.monitor.GetSystemMonitor().GetStatsHistory(0)
		out := make([]*SystemStats, 0, len(all))
		after := time.UnixMilli(ms)
		for _, s := range all {
			if s.Timestamp.After(after) {
				out = append(out, s)
			}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	stats := api.monitor.GetSystemStats(limit)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// GetLatestSystemStats 获取最新系统统计
func (api *MonitorAPI) GetLatestSystemStats(c *gin.Context) {
	stats := api.monitor.GetLatestSystemStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetSlowQueries 获取慢查询列表
func (api *MonitorAPI) GetSlowQueries(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}

	all := api.monitor.GetSlowQueries(0) // 拿全量，再分页
	// 已按耗时排序，可保持
	start := (page - 1) * limit
	if start >= len(all) {
		all = []*SQLQuery{}
	} else {
		end := start + limit
		if end > len(all) {
			end = len(all)
		}
		all = all[start:end]
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": all, "page": page, "limit": limit})
}

func (api *MonitorAPI) GetQueryPatterns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}

	all := api.monitor.GetQueryPatterns(0)
	// 已按 AvgTime 排序，可保持
	start := (page - 1) * limit
	if start >= len(all) {
		all = []*QueryPattern{}
	} else {
		end := start + limit
		if end > len(all) {
			end = len(all)
		}
		all = all[start:end]
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": all, "page": page, "limit": limit})
}

// GetSQLStats 获取SQL统计信息
func (api *MonitorAPI) GetSQLStats(c *gin.Context) {
	if api.monitor.GetSQLAnalyzer() == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	stats := api.monitor.GetSQLAnalyzer().GetQueryStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetQueriesByTable 按表获取查询
func (api *MonitorAPI) GetQueriesByTable(c *gin.Context) {
	table := c.Param("table")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	queries := api.monitor.GetQueriesByTable(table, limit)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    queries,
	})
}

// GetQueriesByOperation 按操作类型获取查询
func (api *MonitorAPI) GetQueriesByOperation(c *gin.Context) {
	operation := c.Param("operation")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	queries := api.monitor.GetQueriesByOperation(operation, limit)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    queries,
	})
}

// GetTraces 获取追踪列表（新增 page/limit/status/name 前缀过滤）
func (api *MonitorAPI) GetTraces(c *gin.Context) {
	if api.monitor.GetTracer() == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}

	statusFilter := strings.ToUpper(c.DefaultQuery("status", "")) // OK/ERROR
	namePrefix := c.DefaultQuery("name", "")                      // 前缀匹配

	all := api.monitor.GetTracer().GetSpans()
	// 过滤
	filtered := make([]*Span, 0, len(all))
	for _, s := range all {
		if statusFilter != "" {
			if (statusFilter == "OK" && s.Status != SpanStatusOK) || (statusFilter == "ERROR" && s.Status != SpanStatusError) {
				continue
			}
		}
		if namePrefix != "" && !strings.HasPrefix(strings.ToLower(s.Name), strings.ToLower(namePrefix)) {
			continue
		}
		filtered = append(filtered, s)
	}
	// 按开始时间倒序
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].StartTime.After(filtered[j].StartTime) })

	// 分页
	start := (page - 1) * limit
	if start >= len(filtered) {
		filtered = []*Span{}
	} else {
		end := start + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[start:end]
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    filtered,
		"page":    page,
		"limit":   limit,
		"total":   len(all),
	})
}

// GetTraceDetail 获取追踪详情
func (api *MonitorAPI) GetTraceDetail(c *gin.Context) {
	traceID := c.Param("traceID")

	if api.monitor.GetTracer() == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	spans := api.monitor.GetTraceSpans(traceID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    spans,
	})
}

// GetMetrics 获取指标数据
func (api *MonitorAPI) GetMetrics(c *gin.Context) {
	if api.monitor.GetMetrics() == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	// 这里可以返回自定义的指标数据
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"timestamp": time.Now(),
			"message":   "Metrics collection is enabled",
		},
	})
}

// GetPrometheusMetrics 获取Prometheus格式的指标
func (api *MonitorAPI) GetPrometheusMetrics(c *gin.Context) {
	// 这里应该返回Prometheus格式的指标数据
	// 由于Prometheus指标是自动注册的，我们只需要返回一个说明
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "# Prometheus metrics are automatically exposed at /metrics endpoint\n# This endpoint is for compatibility only")
}
