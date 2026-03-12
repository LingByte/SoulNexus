package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MonitorMiddleware 监控中间件
func MonitorMiddleware(monitor *Monitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 开始链路追踪
		ctx, span := monitor.StartSpan(c.Request.Context(), c.HandlerName(),
			WithTags(map[string]string{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
				"ip":     c.ClientIP(),
			}),
		)

		// 将span添加到上下文
		c.Request = c.Request.WithContext(ctx)

		// 记录请求开始
		if span != nil {
			span.AddEvent("request_started", map[string]interface{}{
				"user_agent": c.Request.UserAgent(),
				"referer":    c.Request.Referer(),
			})
		}

		// 处理请求
		c.Next()

		// 计算请求耗时
		duration := time.Since(start)

		// 记录HTTP请求指标
		status := c.Writer.Status()
		requestSize := c.Request.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}
		responseSize := int64(c.Writer.Size())

		monitor.RecordHTTPRequest(
			c.Request.Method,
			c.Request.URL.Path,
			strconv.Itoa(c.Writer.Status()),
			c.HandlerName(),
			duration,
			requestSize,
			responseSize,
		)

		// 结束链路追踪
		var err error
		if status >= 400 {
			err = gin.Error{Err: fmt.Errorf("HTTP %d", status)}
		}
		monitor.EndSpan(span, err)

		// 记录请求完成事件
		if span != nil {
			span.AddEvent("request_completed", map[string]interface{}{
				"status_code": status,
				"duration_ms": duration.Milliseconds(),
			})
		}
	}
}
