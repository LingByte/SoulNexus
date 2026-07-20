package aliyunomni

import (
	"strings"

	"github.com/LingByte/lingllm/realtime"
)

// CreateResponse asks DashScope to generate a reply after manual input_audio_buffer.commit.
func (a *agent) CreateResponse(instructions string) error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	payload := map[string]any{"type": "response.create"}
	instructions = strings.TrimSpace(instructions)
	if instructions != "" {
		payload["response"] = map[string]any{"instructions": instructions}
	}
	return a.sendJSON(payload, false)
}

// ClearInputAudio drops buffered caller audio on the omni session.
func (a *agent) ClearInputAudio() error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	return a.sendJSON(map[string]any{"type": "input_audio_buffer.clear"}, false)
}
