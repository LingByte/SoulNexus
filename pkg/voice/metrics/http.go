package metrics

const (
	MetricHTTPRequestsTotal = "http_requests_total"
	MetricHTTPErrorsTotal   = "http_request_errors_total"
	MetricHTTPDurationMs    = "http_request_duration_ms"
)

func init() {
	RegisterLabels(MetricHTTPRequestsTotal, "method", "status")
	RegisterLabels(MetricHTTPErrorsTotal, "kind")
}

// ObserveHTTPRequest records one HTTP request for ops dashboards.
func ObserveHTTPRequest(method, status string, durationMs float64) {
	Default.IncCounter(MetricHTTPRequestsTotal,
		"total HTTP requests",
		map[string]string{"method": method, "status": status})
	if durationMs > 0 {
		Default.Observe(MetricHTTPDurationMs, "HTTP request duration ms", durationMs)
	}
	if len(status) >= 1 && status[0] == '5' {
		Default.IncCounter(MetricHTTPErrorsTotal, "HTTP 5xx errors", map[string]string{"kind": "5xx"})
	}
	if len(status) >= 1 && status[0] == '4' {
		Default.IncCounter(MetricHTTPErrorsTotal, "HTTP 4xx errors", map[string]string{"kind": "4xx"})
	}
}
