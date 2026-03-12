package metrics

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// SQLQuery SQL查询记录
type SQLQuery struct {
	ID           string                 `json:"id"`
	TraceID      string                 `json:"trace_id"`
	SQL          string                 `json:"sql"`
	Params       []interface{}          `json:"params"`
	Table        string                 `json:"table"`
	Operation    string                 `json:"operation"`
	Duration     time.Duration          `json:"duration"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	RowsAffected int64                  `json:"rows_affected"`
	Error        error                  `json:"error,omitempty"`
	ExplainPlan  *ExplainPlan           `json:"explain_plan,omitempty"`
	Tags         map[string]string      `json:"tags"`
	Attributes   map[string]interface{} `json:"attributes"`
}

// ExplainPlan 执行计划
type ExplainPlan struct {
	ID           int     `json:"id"`
	SelectType   string  `json:"select_type"`
	Table        string  `json:"table"`
	Partitions   string  `json:"partitions"`
	Type         string  `json:"type"`
	PossibleKeys string  `json:"possible_keys"`
	Key          string  `json:"key"`
	KeyLen       int     `json:"key_len"`
	Ref          string  `json:"ref"`
	Rows         int64   `json:"rows"`
	Filtered     float64 `json:"filtered"`
	Extra        string  `json:"extra"`
	Cost         float64 `json:"cost"`
}

// SQLAnalyzer SQL分析器
type SQLAnalyzer struct {
	queries       map[string]*SQLQuery
	slowQueries   []*SQLQuery
	mu            sync.RWMutex
	maxQueries    int
	slowThreshold time.Duration
	patterns      map[string]*QueryPattern
}

// QueryPattern 查询模式
type QueryPattern struct {
	Pattern    string         `json:"pattern"`
	Count      int            `json:"count"`
	TotalTime  time.Duration  `json:"total_time"`
	AvgTime    time.Duration  `json:"avg_time"`
	MaxTime    time.Duration  `json:"max_time"`
	MinTime    time.Duration  `json:"min_time"`
	LastSeen   time.Time      `json:"last_seen"`
	Tables     map[string]int `json:"tables"`
	Operations map[string]int `json:"operations"`
}

// NewSQLAnalyzer 创建SQL分析器
func NewSQLAnalyzer(maxQueries int, slowThreshold time.Duration) *SQLAnalyzer {
	return &SQLAnalyzer{
		queries:       make(map[string]*SQLQuery),
		slowQueries:   make([]*SQLQuery, 0),
		maxQueries:    maxQueries,
		slowThreshold: slowThreshold,
		patterns:      make(map[string]*QueryPattern),
	}
}

// RecordQuery 记录SQL查询
func (sa *SQLAnalyzer) RecordQuery(ctx context.Context, sql string, params []interface{}, table, operation string, duration time.Duration, rowsAffected int64, err error) *SQLQuery {
	query := &SQLQuery{
		ID:           generateQueryID(),
		TraceID:      getTraceIDFromContext(ctx),
		SQL:          sql,
		Params:       params,
		Table:        table,
		Operation:    operation,
		Duration:     duration,
		StartTime:    time.Now().Add(-duration),
		EndTime:      time.Now(),
		RowsAffected: rowsAffected,
		Error:        err,
		Tags:         make(map[string]string),
		Attributes:   make(map[string]interface{}),
	}

	// 从上下文获取跨度信息
	if span := getSpanFromContext(ctx); span != nil {
		query.Tags["handler"] = span.Name
		query.Tags["path"] = span.Tags["path"]
		query.Tags["method"] = span.Tags["method"]
	}

	sa.mu.Lock()
	defer sa.mu.Unlock()

	// 存储查询
	if len(sa.queries) >= sa.maxQueries {
		sa.cleanupOldQueries()
	}
	sa.queries[query.ID] = query

	// 检查是否为慢查询
	if duration >= sa.slowThreshold {
		sa.slowQueries = append(sa.slowQueries, query)
		// 保持慢查询列表在合理范围内
		if len(sa.slowQueries) > 1000 {
			sa.slowQueries = sa.slowQueries[1:]
		}
	}

	// 分析查询模式
	sa.analyzeQueryPattern(query)

	return query
}

// analyzeQueryPattern 分析查询模式
func (sa *SQLAnalyzer) analyzeQueryPattern(query *SQLQuery) {
	// 生成查询模式（去除具体值，保留结构）
	pattern := sa.normalizeSQL(query.SQL)

	if existing, exists := sa.patterns[pattern]; exists {
		existing.Count++
		existing.TotalTime += query.Duration
		existing.AvgTime = existing.TotalTime / time.Duration(existing.Count)
		if query.Duration > existing.MaxTime {
			existing.MaxTime = query.Duration
		}
		if query.Duration < existing.MinTime || existing.MinTime == 0 {
			existing.MinTime = query.Duration
		}
		existing.LastSeen = query.EndTime
		existing.Tables[query.Table]++
		existing.Operations[query.Operation]++
	} else {
		sa.patterns[pattern] = &QueryPattern{
			Pattern:    pattern,
			Count:      1,
			TotalTime:  query.Duration,
			AvgTime:    query.Duration,
			MaxTime:    query.Duration,
			MinTime:    query.Duration,
			LastSeen:   query.EndTime,
			Tables:     map[string]int{query.Table: 1},
			Operations: map[string]int{query.Operation: 1},
		}
	}
}

// normalizeSQL 标准化SQL语句
func (sa *SQLAnalyzer) normalizeSQL(sql string) string {
	// 转换为小写
	sql = strings.ToLower(sql)

	// 移除字符串字面量
	sql = regexp.MustCompile(`'[^']*'`).ReplaceAllString(sql, "?")

	// 移除数字字面量
	sql = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(sql, "?")

	// 移除多余空格
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")

	return strings.TrimSpace(sql)
}

// GetSlowQueries 获取慢查询列表
func (sa *SQLAnalyzer) GetSlowQueries(limit int) []*SQLQuery {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	if limit <= 0 || limit > len(sa.slowQueries) {
		limit = len(sa.slowQueries)
	}

	// 按耗时排序
	queries := make([]*SQLQuery, limit)
	copy(queries, sa.slowQueries[:limit])

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].Duration > queries[j].Duration
	})

	return queries
}

// GetQueryPatterns 获取查询模式
func (sa *SQLAnalyzer) GetQueryPatterns(limit int) []*QueryPattern {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	patterns := make([]*QueryPattern, 0, len(sa.patterns))
	for _, pattern := range sa.patterns {
		patterns = append(patterns, pattern)
	}

	// 按平均耗时排序
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].AvgTime > patterns[j].AvgTime
	})

	if limit > 0 && limit < len(patterns) {
		patterns = patterns[:limit]
	}

	return patterns
}

// GetQueriesByTable 按表获取查询
func (sa *SQLAnalyzer) GetQueriesByTable(table string, limit int) []*SQLQuery {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	var queries []*SQLQuery
	for _, query := range sa.queries {
		if query.Table == table {
			queries = append(queries, query)
		}
	}

	// 按时间排序
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].StartTime.After(queries[j].StartTime)
	})

	if limit > 0 && limit < len(queries) {
		queries = queries[:limit]
	}

	return queries
}

// GetQueriesByOperation 按操作类型获取查询
func (sa *SQLAnalyzer) GetQueriesByOperation(operation string, limit int) []*SQLQuery {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	var queries []*SQLQuery
	for _, query := range sa.queries {
		if query.Operation == operation {
			queries = append(queries, query)
		}
	}

	// 按时间排序
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].StartTime.After(queries[j].StartTime)
	})

	if limit > 0 && limit < len(queries) {
		queries = queries[:limit]
	}

	return queries
}

// GetQueryStats 获取查询统计信息
func (sa *SQLAnalyzer) GetQueryStats() map[string]interface{} {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	stats := map[string]interface{}{
		"total_queries":  len(sa.queries),
		"slow_queries":   len(sa.slowQueries),
		"patterns":       len(sa.patterns),
		"slow_threshold": sa.slowThreshold.String(),
	}

	// 计算总体统计
	var totalDuration time.Duration
	var totalRows int64
	var errorCount int
	tables := make(map[string]int)
	operations := make(map[string]int)

	for _, query := range sa.queries {
		totalDuration += query.Duration
		totalRows += query.RowsAffected
		if query.Error != nil {
			errorCount++
		}
		tables[query.Table]++
		operations[query.Operation]++
	}

	if len(sa.queries) > 0 {
		stats["avg_duration"] = (totalDuration / time.Duration(len(sa.queries))).String()
		stats["total_duration"] = totalDuration.String()
		stats["total_rows"] = totalRows
		stats["error_rate"] = float64(errorCount) / float64(len(sa.queries))
	}

	stats["tables"] = tables
	stats["operations"] = operations

	return stats
}

// cleanupOldQueries 清理旧查询
func (sa *SQLAnalyzer) cleanupOldQueries() {
	// 按时间排序，删除最旧的
	queries := make([]*SQLQuery, 0, len(sa.queries))
	for _, query := range sa.queries {
		queries = append(queries, query)
	}

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].StartTime.Before(queries[j].StartTime)
	})

	// 删除最旧的查询，保留一半
	keepCount := len(queries) / 2
	for i := 0; i < keepCount; i++ {
		delete(sa.queries, queries[i].ID)
	}
}

// generateQueryID 生成查询ID
func generateQueryID() string {
	return fmt.Sprintf("query_%d", time.Now().UnixNano())
}
