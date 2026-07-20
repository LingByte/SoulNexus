package realtime

import (
	"encoding/json"
	"testing"

	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
)


func TestVoiceRealtimeToolHandler_Unknown(t *testing.T) {
	h := NewToolHandler("c1", nil, stageknow.Binding{})
	out := h("no_such_tool", nil)
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil || m["ok"] != false {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestRealtimeTools_Count(t *testing.T) {
	if len(RealtimeTools(false)) != 3 {
		t.Fatalf("expected 3 tools (no transfer), got %d", len(RealtimeTools(false)))
	}
	if len(RealtimeTools(true)) != len(RealtimeTools(false))+1 {
		t.Fatalf("knowledge tool not added: base=%d with_kb=%d",
			len(RealtimeTools(false)), len(RealtimeTools(true)))
	}
}
