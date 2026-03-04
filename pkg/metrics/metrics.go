package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsOnce   sync.Once
	globalMetrics *Metrics
)

// Metrics 指标管理器
type Metrics struct {
	// HTTP请求指标
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// 数据库指标
	dbQueryDuration     *prometheus.HistogramVec
	dbConnectionsActive *prometheus.GaugeVec
	dbConnectionsTotal  *prometheus.CounterVec

	// 缓存指标
	cacheHitsTotal   *prometheus.CounterVec
	cacheMissesTotal *prometheus.CounterVec
	cacheSize        *prometheus.GaugeVec

	// 业务指标
	businessCounter   *prometheus.CounterVec
	businessGauge     *prometheus.GaugeVec
	businessHistogram *prometheus.HistogramVec

	// 系统指标
	systemMemoryUsage *prometheus.GaugeVec
	systemCPUUsage    *prometheus.GaugeVec
	systemGoroutines  *prometheus.GaugeVec
}

// NewMetrics 创建指标管理器
// 注意：由于 promauto 会自动注册到全局注册表，多次调用会导致重复注册错误
// 使用 sync.Once 确保只创建一次
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = newMetrics()
	})
	return globalMetrics
}

// newMetrics 实际创建指标管理器的内部函数
func newMetrics() *Metrics {
	m := &Metrics{
		// HTTP请求指标
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status", "handler"},
		),

		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "handler"},
		),

		httpRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),

		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path", "status"},
		),

		// 数据库指标
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table", "sql_type"},
		),

		dbConnectionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
			[]string{"database", "status"},
		),

		dbConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_connections_total",
				Help: "Total number of database connections",
			},
			[]string{"database", "operation"},
		),

		// 缓存指标
		cacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type", "operation"},
		),

		cacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type", "operation"},
		),

		cacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cache_size",
				Help: "Current cache size",
			},
			[]string{"cache_type"},
		),

		// 业务指标
		businessCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "business_operations_total",
				Help: "Total number of business operations",
			},
			[]string{"operation", "status", "user_type"},
		),

		businessGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "business_metrics",
				Help: "Business metrics",
			},
			[]string{"metric", "category"},
		),

		businessHistogram: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "business_duration_seconds",
				Help:    "Business operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "category"},
		),

		// 系统指标
		systemMemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_memory_usage_bytes",
				Help: "System memory usage in bytes",
			},
			[]string{"type"},
		),

		systemCPUUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_cpu_usage_percent",
				Help: "System CPU usage percentage",
			},
			[]string{"core"},
		),

		systemGoroutines: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_goroutines",
				Help: "Number of goroutines",
			},
			[]string{},
		),
	}

	return m
}

// RecordHTTPRequest 记录HTTP请求指标
func (m *Metrics) RecordHTTPRequest(method, path, status, handler string, duration time.Duration, requestSize, responseSize int64) {
	m.httpRequestsTotal.WithLabelValues(method, path, status, handler).Inc()
	m.httpRequestDuration.WithLabelValues(method, path, handler).Observe(duration.Seconds())
	m.httpRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	m.httpResponseSize.WithLabelValues(method, path, status).Observe(float64(responseSize))
}

// RecordDBQuery 记录数据库查询指标
func (m *Metrics) RecordDBQuery(operation, table, sqlType string, duration time.Duration) {
	m.dbQueryDuration.WithLabelValues(operation, table, sqlType).Observe(duration.Seconds())
}

// RecordDBConnection 记录数据库连接指标
func (m *Metrics) RecordDBConnection(database, operation string) {
	m.dbConnectionsTotal.WithLabelValues(database, operation).Inc()
}

// SetDBConnectionsActive 设置活跃数据库连接数
func (m *Metrics) SetDBConnectionsActive(database, status string, count int) {
	m.dbConnectionsActive.WithLabelValues(database, status).Set(float64(count))
}

// RecordCacheHit 记录缓存命中
func (m *Metrics) RecordCacheHit(cacheType, operation string) {
	m.cacheHitsTotal.WithLabelValues(cacheType, operation).Inc()
}

// RecordCacheMiss 记录缓存未命中
func (m *Metrics) RecordCacheMiss(cacheType, operation string) {
	m.cacheMissesTotal.WithLabelValues(cacheType, operation).Inc()
}

// SetCacheSize 设置缓存大小
func (m *Metrics) SetCacheSize(cacheType string, size int) {
	m.cacheSize.WithLabelValues(cacheType).Set(float64(size))
}

// RecordBusinessOperation 记录业务操作
func (m *Metrics) RecordBusinessOperation(operation, status, userType string) {
	m.businessCounter.WithLabelValues(operation, status, userType).Inc()
}

// SetBusinessMetric 设置业务指标
func (m *Metrics) SetBusinessMetric(metric, category string, value float64) {
	m.businessGauge.WithLabelValues(metric, category).Set(value)
}

// RecordBusinessDuration 记录业务操作耗时
func (m *Metrics) RecordBusinessDuration(operation, category string, duration time.Duration) {
	m.businessHistogram.WithLabelValues(operation, category).Observe(duration.Seconds())
}

// SetSystemMemoryUsage 设置系统内存使用量
func (m *Metrics) SetSystemMemoryUsage(memoryType string, bytes int64) {
	m.systemMemoryUsage.WithLabelValues(memoryType).Set(float64(bytes))
}

// SetSystemCPUUsage 设置系统CPU使用率
func (m *Metrics) SetSystemCPUUsage(core string, percentage float64) {
	m.systemCPUUsage.WithLabelValues(core).Set(percentage)
}

// SetSystemGoroutines 设置goroutine数量
func (m *Metrics) SetSystemGoroutines(count int) {
	m.systemGoroutines.WithLabelValues().Set(float64(count))
}

// GetCacheHitRate 获取缓存命中率
func (m *Metrics) GetCacheHitRate(cacheType, operation string) float64 {
	// 由于Prometheus指标是只写的，我们无法直接读取值
	// 这里返回0，实际使用时需要通过其他方式统计
	return 0.0
}

// Reset 重置所有指标
func (m *Metrics) Reset() {
	// 重置计数器
	m.httpRequestsTotal.Reset()
	m.dbConnectionsTotal.Reset()
	m.cacheHitsTotal.Reset()
	m.cacheMissesTotal.Reset()
	m.businessCounter.Reset()

	// 重置直方图
	m.httpRequestDuration.Reset()
	m.httpRequestSize.Reset()
	m.httpResponseSize.Reset()
	m.dbQueryDuration.Reset()
	m.businessHistogram.Reset()

	// 重置仪表盘
	m.dbConnectionsActive.Reset()
	m.cacheSize.Reset()
	m.businessGauge.Reset()
	m.systemMemoryUsage.Reset()
	m.systemCPUUsage.Reset()
	m.systemGoroutines.Reset()
}
