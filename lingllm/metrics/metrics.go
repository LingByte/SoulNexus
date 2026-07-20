package metrics

import "time"

// CallMetrics reports lightweight performance indicators for a single LLM call.
type CallMetrics struct {
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	StartAt          time.Time `json:"start_at"`
	FirstAt          time.Time `json:"first_at,omitempty"`
	EndAt            time.Time `json:"end_at,omitempty"`
	Bytes            int       `json:"bytes"`
	Chunks           int       `json:"chunks"`
	RequestBytes     int       `json:"request_bytes,omitempty"`
	ResponseBytes    int       `json:"response_bytes,omitempty"`
	HTTPStatus       int       `json:"http_status,omitempty"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	Error            string    `json:"error,omitempty"`
}

// Latency returns end-to-end latency (EndAt-StartAt) if available.
func (m CallMetrics) Latency() time.Duration {
	if m.StartAt.IsZero() || m.EndAt.IsZero() {
		return 0
	}
	return m.EndAt.Sub(m.StartAt)
}

// FirstTokenLatency returns time to first delta/token if available.
func (m CallMetrics) FirstTokenLatency() time.Duration {
	if m.StartAt.IsZero() || m.FirstAt.IsZero() {
		return 0
	}
	return m.FirstAt.Sub(m.StartAt)
}
