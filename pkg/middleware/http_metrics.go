package middleware

import (
	"strconv"
	"time"

	voiceMetrics "github.com/LingByte/SoulNexus/pkg/voice/metrics"
	"github.com/LingByte/SoulNexus/pkg/utils/system"
	"github.com/gin-gonic/gin"
)

// HTTPMetricsMiddleware records request counts and latency for ops dashboards.
func HTTPMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		status := strconv.Itoa(c.Writer.Status())
		ms := float64(time.Since(start).Milliseconds())
		voiceMetrics.ObserveHTTPRequest(c.Request.Method, status, ms)
		if len(status) >= 1 {
			switch status[0] {
			case '5':
				system.IncHTTP5xx()
			case '4':
				system.IncHTTP4xx()
			}
		}
	}
}
