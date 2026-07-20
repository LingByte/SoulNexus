package metrics

import (
	"testing"
	"time"
)

func TestLatencyZero(t *testing.T) {
	m := CallMetrics{}
	if got := m.Latency(); got != 0 {
		t.Errorf("expected 0 latency, got %v", got)
	}

	m = CallMetrics{StartAt: time.Now()}
	if got := m.Latency(); got != 0 {
		t.Errorf("expected 0 latency with only StartAt, got %v", got)
	}
}

func TestLatency(t *testing.T) {
	start := time.Now().Add(-100 * time.Millisecond)
	end := time.Now()
	m := CallMetrics{StartAt: start, EndAt: end}

	latency := m.Latency()
	if latency < 90*time.Millisecond || latency > 200*time.Millisecond {
		t.Errorf("unexpected latency: %v", latency)
	}
}

func TestFirstTokenLatencyZero(t *testing.T) {
	m := CallMetrics{}
	if got := m.FirstTokenLatency(); got != 0 {
		t.Errorf("expected 0 first token latency, got %v", got)
	}
}

func TestFirstTokenLatency(t *testing.T) {
	start := time.Now().Add(-50 * time.Millisecond)
	first := time.Now()
	m := CallMetrics{StartAt: start, FirstAt: first}

	ttft := m.FirstTokenLatency()
	if ttft < 40*time.Millisecond || ttft > 150*time.Millisecond {
		t.Errorf("unexpected first token latency: %v", ttft)
	}
}

func TestCallMetricsFields(t *testing.T) {
	now := time.Now()
	m := CallMetrics{
		Provider:         "openai",
		Model:            "gpt-4",
		StartAt:          now,
		FirstAt:          now,
		EndAt:            now,
		Bytes:            1024,
		Chunks:           10,
		RequestBytes:     512,
		ResponseBytes:    512,
		HTTPStatus:       200,
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	if m.Provider != "openai" || m.Model != "gpt-4" || m.TotalTokens != 150 {
		t.Errorf("unexpected metrics fields: %+v", m)
	}
}
