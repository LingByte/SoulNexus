// Package exutil provides shared helpers for example programs.
package exutil

import (
	"log"
	"time"

	"github.com/LingByte/lingllm/metrics"
	"github.com/LingByte/lingllm/protocol"
)

// LogChat logs provider-reported LLM call latency and end-to-end wall time for a sync Chat.
// e2eStart should be captured immediately before client.Chat (or chain/tool invocation).
func LogChat(label string, resp *protocol.ChatResponse, e2eStart time.Time) {
	if resp == nil {
		return
	}
	m := resp.Metrics
	log.Printf("[%s] llm_call=%s e2e=%s model=%s tokens=%d",
		label, fmtDur(m.Latency()), fmtDur(time.Since(e2eStart)), m.Model, resp.Usage.TotalTokens)
}

// LogStream logs provider-reported stream latency, TTFT, and end-to-end wall time.
// e2eStart should be captured immediately before client.StreamChat.
func LogStream(label string, stream protocol.ChatStream, e2eStart time.Time) {
	if stream == nil {
		return
	}
	m := stream.Metrics()
	log.Printf("[%s] llm_call=%s ttft=%s e2e=%s model=%s",
		label, fmtDur(m.Latency()), fmtDur(m.FirstTokenLatency()), fmtDur(time.Since(e2eStart)), m.Model)
}

// LogBatch logs batch end-to-end time and per-response LLM call latency.
func LogBatch(label string, responses []*protocol.ChatResponse, e2eStart time.Time) {
	log.Printf("[%s] batch_e2e=%s count=%d", label, fmtDur(time.Since(e2eStart)), len(responses))
	for i, resp := range responses {
		if resp == nil {
			continue
		}
		m := resp.Metrics
		log.Printf("[%s] item=%d llm_call=%s model=%s tokens=%d",
			label, i+1, fmtDur(m.Latency()), m.Model, resp.Usage.TotalTokens)
	}
}

// LogTurn logs voice/dialog turn latency. ttft is time from turn start to first token;
// use a negative duration when no token was received.
func LogTurn(label, callID string, stream protocol.ChatStream, e2eStart time.Time, ttft time.Duration) {
	m := metrics.CallMetrics{}
	if stream != nil {
		m = stream.Metrics()
	}
	ttftStr := fmtDur(ttft)
	if ttft < 0 {
		ttftStr = "-"
	}
	log.Printf("[%s] call=%s llm_call=%s ttft=%s e2e=%s model=%s",
		label, callID, fmtDur(m.Latency()), ttftStr, fmtDur(time.Since(e2eStart)), m.Model)
}

// LogE2E logs end-to-end wall time when provider metrics are unavailable (e.g. mock models).
func LogE2E(label string, e2eStart time.Time) {
	log.Printf("[%s] e2e=%s (no provider metrics)", label, fmtDur(time.Since(e2eStart)))
}

func fmtDur(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(10 * time.Millisecond).String()
}
